package rpcclient

import (
	rpctypes "github.com/filecoin-project/lotus/extern/miningstate/types"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"net/rpc"
)
var Log = logging.Logger("main")

var MasterIP string
var LocalIP string

type Offset uint64
// Register
type Register struct {
	StartRange uint64
}
type RegisterRequest struct {
	Ip string
}
type RegisterResponse struct {
	Ranges uint64
}

func RegisterToMasterSidOffset(role rpctypes.Role,
	masterIp rpctypes.RemoteServerAddr,localIp rpctypes.LocalServerAddr) Offset {

	MasterIP = string(masterIp)
	LocalIP =  string(localIp)

	var res RegisterResponse
	if role == rpctypes.Role_Master {
		Log.Info("[RegisterToMaster]---Local[",string(localIp),"] This is Master Self")
		res.Ranges = 0
	}else if role == rpctypes.Role_Slave {
		res = RpcCallRegister(string(masterIp),"Register.Reg",RegisterRequest{string(localIp)})
		Log.Info("[RegisterToMaster]---Local[",string(localIp),"]-----Reg------->MasterIp[",string(masterIp),"]")
		Log.Info("get offset --------------->",res.Ranges)
	}

	return Offset(res.Ranges)
}

func RpcCallRegister(ipaddress string, method string, req RegisterRequest) RegisterResponse {
	conn, err := rpc.DialHTTP("tcp", ipaddress)
	if err != nil {
		fmt.Println("dailing error: ", err)
	}

	var res RegisterResponse

	err = conn.Call(method, req, &res)
	if err != nil {
		fmt.Println("WindowPoSt error: ", err)
	}

	return res
}

func RpcCallWindowPoStSync(ipaddress string, method string, req WindowPoStRequest) chan *rpc.Call {
	conn, err := rpc.DialHTTP("tcp", MasterIP)
	if err != nil {
		fmt.Println("dailing error: ", err)
	}

	var res WindowPoStResponse

	done := make(chan *rpc.Call, 1)
	conn.Go(method, req, &res, done)

	return done
}

type SectorIdIndex struct {
	SectorId abi.SectorNumber
	PrzIndex uint64
}

// VanillaProof
type WindowPoStRequest struct {
	MinerID abi.ActorID
	SectorInfo []abi.SectorInfo
	Randomness abi.PoStRandomness
	Index []SectorIdIndex
}

// VanillaProof
type WindowPoStResponse struct {
	VanillaProof []abi.PoStProof
	Index []SectorIdIndex
}

func getindex(a []SectorIdIndex) []uint64 {
	index := make([]uint64,1,1)
	for _,ind := range a{
		index = append(index,ind.PrzIndex)
	}
	return index
}



//Commit
type CommitReq struct {
	Msg      types.Message
	Spec     api.MessageSendSpec
	SectorId uint64
	SlaveIp  string
}
type CommitRes struct {
	Smsg types.SignedMessage
}

func RpcCallCommit(req CommitReq) (CommitRes,error) {
	req.SlaveIp = LocalIP
	conn, err := rpc.DialHTTP("tcp", MasterIP)
	if err != nil {
		fmt.Println("dailing error: ", err)
	}

	var res CommitRes

	err = conn.Call("Commit.PushMsg", req, &res)
	if err != nil {
		fmt.Println("WindowPoSt error: ", err)
	}

	return res,nil
}


type SectorRequest struct {
	SectorId uint64
	Data     []byte
}

type SectorResponse struct {
	SectorId uint64
	Data     []byte
}


// to get MinerAddress from Master MinerDS
type MinerAddress struct {
	Maddr address.Address
}
type MinerAddressReq struct {
}
type MindrAddressRes struct {
	Maddr address.Address
}

// not used
func RpcCallMinerAddress(role rpctypes.Role, ds dtypes.MetadataDS,addr dtypes.MinerAddressIntermediate) dtypes.MinerAddress {

	if role == rpctypes.Role_Master {
		Log.Info("Master return loacl MinerAddress")
		return dtypes.MinerAddress(addr)
	}

	req := MinerAddressReq{}
	method := "MinerAddress.GetAddress"

	conn, err := rpc.DialHTTP("tcp", MasterIP)
	if err != nil {
		fmt.Println("dailing error: ", err)
	}

	var res MindrAddressRes

	err = conn.Call(method, req, &res)
	if err != nil {
		fmt.Println("WindowPoSt error: ", err)
	}


	// ReSaveTheAddress
	if err := ds.Put(datastore.NewKey("miner-address"), res.Maddr.Bytes()); err != nil {
		return dtypes.MinerAddress(address.Undef)
	}

	return dtypes.MinerAddress(res.Maddr)
}

// used in miner init!!!!!!! ip and role required
func RpcCallMinerAddressInit(role rpctypes.Role,ip string) address.Address {

	if role == rpctypes.Role_Master {
		Log.Info("Master return loacl MinerAddress")
		return address.Undef
	}

	req := MinerAddressReq{}
	method := "MinerAddress.GetAddress"

	conn, err := rpc.DialHTTP("tcp", ip)
	if err != nil {
		fmt.Println("dailing error: ", err)
	}

	var res MindrAddressRes

	err = conn.Call(method, req, &res)
	if err != nil {
		fmt.Println("addr error: ", err)
	}

	Log.Info("Get Maddr From Master =",res.Maddr)
	return res.Maddr
}