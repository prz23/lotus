package umrpc

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/extern/miningstate/rpcclient"
	state "github.com/filecoin-project/lotus/extern/miningstate/rpcsectorstate"
	rpctypes "github.com/filecoin-project/lotus/extern/miningstate/types"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	sectorstorage "github.com/filecoin-project/sector-storage"
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

	//fn := func(all []rpcclient.SectorIdIndex) []uint64{
	//	s := make([]uint64,1,1)
	//	for _,each := range all {
	//		s = append(s,each.PrzIndex)
	//	}
	//	return s
	//}


	//res.VanillaProof,_,_ = sb.sealer.GenerateWindowPoStVanilla(context.Background(),req.MinerID,req.SectorInfo,req.Randomness,fn(req.Index))
	//res.Index = req.Index
	return nil
}

// Sector
type SectorRpc struct {
}

func (a *SectorRpc)DoSector(req rpcclient.SectorRequest, res *rpcclient.SectorResponse) error{
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
}

func (s *Commit)PushMsg(req rpcclient.CommitReq, res *rpcclient.CommitRes) error{
	smsg,err := s.api.MpoolPushMessage(context.Background(),&req.Msg,&req.Spec)
	if err != nil {
		res.Smsg = *smsg
		getIns := GetIns()
		getIns.Insert(req.SectorId,req.SlaveIp)
	}
	return err
}


type MinerAddress struct {
	Maddr address.Address
}

func (s *MinerAddress)GetAddress(req rpcclient.MinerAddressReq, res *rpcclient.MindrAddressRes ) error{
	res.Maddr = s.Maddr
	rpcclient.Log.Info("=====Maddr=====",s.Maddr.String())
	return nil
}


func NewStartMasterRpc(lc fx.Lifecycle, localIpAddress rpctypes.LocalServerAddr,
	sealer sectorstorage.SectorManager, api api.FullNode, maddr dtypes.MinerAddress) error {

	rpcclient.Log.Info("==================================================")
	rpcclient.Log.Info("[NewStartMasterRpc] localIpAddress is ",localIpAddress)
	rpcclient.Log.Info("==================================================")
	rpcclient.Log.Info("[NewStartMasterRpc] MinerAddress is ",maddr)

	newServer := rpc.NewServer()

	err := newServer.Register(new(SectorRpc))
	if err != nil {
		log.Fatalf("net.Listen tcp :0: %v", err)
	}

	err = newServer.Register(&WindowPoSt{sealer: sealer})
	if err != nil {
		log.Fatalf("net.Listen tcp :0: %v", err)
	}

	err = newServer.Register(&Register{2000000})
	if err != nil {
		log.Fatalf("net.Listen tcp :0: %v", err)
	}

	err = newServer.Register(&Commit{api: api})
	if err != nil {
		log.Fatalf("net.Listen tcp :0: %v", err)
	}

	err = newServer.Register(&MinerAddress{Maddr: address.Address(maddr)})
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

