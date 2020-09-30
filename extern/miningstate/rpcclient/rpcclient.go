package rpcclient

import (
	"fmt"
	rpctypes "github.com/filecoin-project/lotus/extern/miningstate/types"
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
	SectorId uint64 /*abi.SectorNumber*/
	PrzIndex uint64
}

// VanillaProof
type WindowPoStRequest struct {
	MinerID uint64 /*abi.ActorID*/
	SectorInfo [][]byte /*[]abi.SectorInfo*/
	Randomness []byte /*abi.PoStRandomness*/
	Index []byte /*SectorIdIndex*/
}

// VanillaProof
type WindowPoStResponse struct {
	VanillaProof [][]byte /*[]abi.PoStProof*/
	Index []byte
	Skipped [][]byte
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
	Msg      [] byte /*types.Message*/
	Spec     [] byte /*api.MessageSendSpec*/
	SectorId uint64
	SlaveIp  string
}
type CommitRes struct {
	Smsg [] byte /*types.SignedMessage*/
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




type CheckSectorsRequest struct {
	Toproof []byte
	Regitype uint64
}

type CheckSectorsResponse struct {
    Bad [][]byte
}

func RpcCallCheck(req CheckSectorsRequest,ip string) (CheckSectorsResponse,error) {
	conn, err := rpc.DialHTTP("tcp", ip)
	if err != nil {
		fmt.Println("dailing error: ", err)
	}

	var res CheckSectorsResponse

	err = conn.Call("CheckSectors.CheckSector", req, &res)
	if err != nil {
		fmt.Println("WindowPoSt error: ", err)
	}

	return res,nil
}