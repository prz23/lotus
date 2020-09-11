package umrpc

import (
	"bytes"
	"context"
	"fmt"
	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-address"
	_ "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/extern/miningstate/rpcclient"
	state "github.com/filecoin-project/lotus/extern/miningstate/rpcsectorstate"
	rpctypes "github.com/filecoin-project/lotus/extern/miningstate/types"
	idstore "github.com/filecoin-project/lotus/extern/sector-id-store"
	sectorstorage "github.com/filecoin-project/lotus/extern/sector-storage"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"go.uber.org/fx"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

// VanillaProof
type WindowPoSt struct {
	sealer sectorstorage.SectorManager
}


func (sb *WindowPoSt) DoVanillaProof(req rpcclient.WindowPoStRequest, res *rpcclient.WindowPoStResponse) error {

	minerid := abi.ActorID(req.MinerID)

	var sectorinfos []abi.SectorInfo
	for _,sinfo := range req.SectorInfo{
		var v abi.SectorInfo
		if err := v.UnmarshalCBOR(bytes.NewBuffer(sinfo)); err != nil {
			return err
		}
		sectorinfos = append(sectorinfos,v)
	}

	rand := abi.PoStRandomness(req.Randomness)

	var SectorIndexInfo ffi.SectorIndexInfo
	_ = SectorIndexInfo.UnmarshalJSON(req.Index)
	
	vanillaproof,skip,_ := sb.sealer.GenerateWindowPoStVanilla(context.Background(), minerid, sectorinfos, rand, SectorIndexInfo)

	var postproofs [][]byte
	for _,proof := range vanillaproof{
		buf := new(bytes.Buffer)
		if err := proof.MarshalCBOR(buf); err != nil {
			return err
		}
		postproofs = append(postproofs,buf.Bytes())
	}

	var skipped [][]byte
	for _,eachskip := range skip{
		buf := new(bytes.Buffer)
		if err := eachskip.MarshalCBOR(buf); err != nil {
			return err
		}
		skipped = append(skipped,buf.Bytes())
	}

	res.VanillaProof = postproofs
	res.Index = req.Index
	res.Skipped = skipped
	return nil
}


// Register
type Register struct {
	startRange uint64
}

func (a *Register)Reg(req rpcclient.RegisterRequest, res *rpcclient.RegisterResponse) error{
	s := state.GetIns()
	if s.HasRegister(req.Ip) {
		fmt.Println("Already Registered")
		return nil
	}
	s.AddNewSlave(req.Ip)
	rpcclient.Log.Info("=========REGISTER==Received========",req.Ip)
	rpcclient.Log.Info("[Master Server Register] receive a Register from IP ----> ",req.Ip)
	rpcclient.Log.Info("=========REGISTER==Received========",req.Ip)
	res.Ranges = a.startRange
	a.startRange = a.startRange + 2000000
	return nil
}


// Commit Message
type Commit struct {
	api api.FullNode
	ids idstore.SectorIpRecord
}

func (s *Commit)PushMsg(req rpcclient.CommitReq, res *rpcclient.CommitRes) error{
	rpcclient.Log.Info("==========PushMsg==1=============")

	var v types.Message
	if err := v.UnmarshalCBOR(bytes.NewBuffer(req.Msg)); err != nil {
		return err
	}

	var v2 abi.TokenAmount
	if err := v2.UnmarshalCBOR(bytes.NewBuffer(req.Spec)); err != nil {
		return err
	}

	smsg,err := s.api.MpoolPushMessage(context.Background(),&v,&api.MessageSendSpec{MaxFee: v2})
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := smsg.MarshalCBOR(buf); err != nil {
		return err
	}

	res.Smsg = buf.Bytes()
	// old way
	getIns := GetIns()
	getIns.Insert(req.SectorId,req.SlaveIp)
    // new way
	err = s.ids.Insert(idstore.SectorId(req.SectorId), idstore.SlaveIP(req.SlaveIp))

	return nil
}


type CheckSectors struct{
	faulttracker  sectorstorage.FaultTracker
	minerid dtypes.MinerAddress
	idc sealing.SectorIDCounter
}

func (s *CheckSectors)CheckSector(req rpcclient.CheckSectorsRequest, res *rpcclient.CheckSectorsResponse) error{

	_,start,end := s.idc.Now()

	var tocheckb bitfield.BitField
	if err := tocheckb.UnmarshalCBOR(bytes.NewBuffer(req.Toproof)); err != nil {
		return err
	}

	filtertochecks := make([]uint64,1)
	a,_ := tocheckb.AllMap(80000000)
    for i := start + 1; i <= end; i = i + 1 {
		if a[i] == true{
			filtertochecks = append(filtertochecks,i)
		}
	}
	filtertocheck := bitfield.NewFromSet(filtertochecks)

	mid, _ := address.IDFromAddress(address.Address(s.minerid))

	sectors := make(map[abi.SectorID]struct{})
	var tocheck []abi.SectorID
	err := filtertocheck.ForEach(func(snum uint64) error {
		s := abi.SectorID{
			Miner:  abi.ActorID(mid),
			Number: abi.SectorNumber(snum),
		}

		tocheck = append(tocheck, s)
		sectors[s] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}

	rst := abi.RegisteredSealProof(req.Regitype)

	checkedlocalbad, err := s.faulttracker.CheckProvable(context.Background(),rst,tocheck)
	if err != nil {
		return err
	}

	var sids [][]byte
	for _,proof := range checkedlocalbad {
		buf := new(bytes.Buffer)
		if err := proof.MarshalCBOR(buf); err != nil {
			return err
		}
		sids = append(sids,buf.Bytes())
	}

	res.Bad = sids

	return nil
}


func NewStartMasterRpc(lc fx.Lifecycle, localIpAddress rpctypes.LocalServerAddr,
	sealer sectorstorage.SectorManager, api api.FullNode,
	maddr dtypes.MinerAddress, save idstore.SectorIpRecord, idc sealing.SectorIDCounter) error {

	rpcclient.Log.Info("==================================================")
	rpcclient.Log.Info("[NewStartMasterRpc] localIpAddress is ",localIpAddress)
	rpcclient.Log.Info("==================================================")
	rpcclient.Log.Info("[NewStartMasterRpc] MinerAddress is ",maddr)

	newServer := rpc.NewServer()

	err := newServer.Register(&WindowPoSt{sealer: sealer})
	if err != nil {
		log.Fatalf("net.Listen tcp :0: %v", err)
	}

	err = newServer.Register(&Register{2000000})
	if err != nil {
		log.Fatalf("net.Listen tcp :0: %v", err)
	}

	err = newServer.Register(&Commit{api: api, ids: save})
	if err != nil {
		log.Fatalf("net.Listen tcp :0: %v", err)
	}

	err = newServer.Register(&CheckSectors{faulttracker:sealer, minerid:maddr, idc:idc})
	if err != nil {
		log.Fatalf("net.Listen tcp :0: %v", err)
	}


	l, err := net.Listen("tcp", string(localIpAddress)) // any available address
	if err != nil {
		log.Fatalf("net.Listen tcp :0: %v", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(i context.Context) error {
			newServer.HandleHTTP("/foo", "/bar")
			go http.Serve(l,newServer)
			rpcclient.Log.Info("[Master Server]----[",string(localIpAddress),"]----[Start]")
			return nil
		},
		OnStop: func(i context.Context) error {
			fmt.Println("stop")
			return nil
		},
	})
	return nil
}

