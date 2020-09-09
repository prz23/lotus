//+build cgo

package ffiwrapper

import (
	"context"
	state "github.com/filecoin-project/lotus/extern/miningstate/rpcsectorstate"
	"net/rpc"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/specs-actors/actors/abi"

	ffi "github.com/filecoin-project/filecoin-ffi"

	"github.com/filecoin-project/lotus/extern/sector-storage/stores"

	"go.opencensus.io/trace"
	"sort"
	umrpc "github.com/filecoin-project/lotus/extern/mining/rpcserver"
	"github.com/filecoin-project/lotus/extern/miningstate/rpcclient"
)

func (sb *Sealer) GenerateWinningPoSt(ctx context.Context, minerID abi.ActorID, sectorInfo []abi.SectorInfo, randomness abi.PoStRandomness) ([]abi.PoStProof, error) {
	randomness[31] &= 0x3f
	privsectors, skipped, done, err := sb.pubSectorToPriv(ctx, minerID, sectorInfo, nil, abi.RegisteredSealProof.RegisteredWinningPoStProof) // TODO: FAULTS?
	if err != nil {
		return nil, err
	}
	defer done()
	if len(skipped) > 0 {
		return nil, xerrors.Errorf("pubSectorToPriv skipped sectors: %+v", skipped)
	}

	return ffi.GenerateWinningPoSt(minerID, privsectors, randomness)
}

func (sb *Sealer) GenerateWindowPoSt(ctx context.Context, minerID abi.ActorID, sectorInfo []abi.SectorInfo, randomness abi.PoStRandomness) ([]abi.PoStProof, []abi.SectorID, error) {
	randomness[31] &= 0x3f
	privsectors, skipped, done, err := sb.pubSectorToPriv(ctx, minerID, sectorInfo, nil, abi.RegisteredSealProof.RegisteredWindowPoStProof)
	if err != nil {
		return nil, nil, xerrors.Errorf("gathering sector info: %w", err)
	}
	defer done()

	proof, err := ffi.GenerateWindowPoSt(minerID, privsectors, randomness)
	return proof, skipped, err
}

func (m *Sealer) GenerateWindowPoStPlus(ctx context.Context, minerID abi.ActorID, sectorInfo []abi.SectorInfo, randomness abi.PoStRandomness) (proof []abi.PoStProof, skipped []abi.SectorID, err error) {

	sectorslen := len(sectorInfo)

	log.Info("[GenerateWindowPoStPlus] sector len = ",sectorslen)

	var SortedSectorNumber sort.IntSlice
	SectorNumberSectorInfoMap := make(map[abi.SectorNumber]abi.SectorInfo)
	for _, SectorInfo := range sectorInfo{
		SortedSectorNumber = append(SortedSectorNumber,int(SectorInfo.SectorNumber))
		SectorNumberSectorInfoMap[SectorInfo.SectorNumber] = SectorInfo
	}
	//  SortedSectorNumber is now in increasing order
	SortedSectorNumber.Sort()

	log.Info("[GenerateWindowPoStPlus] SortedSectorNumber = ",SortedSectorNumber)

	query := make([]state.SectorId,1,1)
	sectormap := make(map[int]int)  // map[SortedSectorNumber]przindex

	for przindex, secortnumber := range SortedSectorNumber {
		sectormap[secortnumber] = przindex
		query = append(query,state.SectorId(secortnumber))
	}

	stateinstance := state.GetIns()
	secipmap := stateinstance.FindAllSort(query)

	log.Info("[GenerateWindowPoStPlus] SlaveIP & Its SectorNumbers = ",SortedSectorNumber)

	var chanlist []chan *rpc.Call
	chanlist = make([]chan *rpc.Call,1,1)

	for ip,sectorids := range secipmap {

		localindex := make([]rpcclient.SectorIdIndex,1,1)
		localcectorinfo := make([]abi.SectorInfo,1,1)

		for _,id := range sectorids {
			localindex = append(localindex, rpcclient.SectorIdIndex{ SectorId: abi.SectorNumber(id),
				PrzIndex: uint64(sectormap[int(id)])})
			localcectorinfo = append(localcectorinfo, SectorNumberSectorInfoMap[abi.SectorNumber(id)])
		}

		log.Info("[GenerateWindowPoStPlus] WindowPoStRequest localcectorinfo = ",localcectorinfo)
		log.Info("[GenerateWindowPoStPlus] WindowPoStRequest localindex = ",localindex)

		req := rpcclient.WindowPoStRequest{MinerID: minerID, SectorInfo: localcectorinfo,
			Randomness: randomness, Index: localindex}
		response := rpcclient.RpcCallWindowPoStSync(ip,"WindowPoSt.DoVanillaProof",req)
		chanlist = append(chanlist,response)
	}

	ResponseData := make([]rpcclient.WindowPoStResponse,1,1)
	for _,eachchan := range chanlist{
		select {
		case data := <-eachchan: ResponseData = append(ResponseData,data.Reply.(rpcclient.WindowPoStResponse))
		}
	}
	// all channels are returned

	// put each vanilla proof in its index
	var ProofInOrder []abi.PoStProof
	ProofInOrder = make([]abi.PoStProof,sectorslen)
	for _,proofdata := range ResponseData{
		for i,index := range proofdata.Index{
			ProofInOrder[index.PrzIndex] = proofdata.VanillaProof[i]
		}
	}


	ProofForSnark := make([][]abi.PoStProof,1,1)
	IndexForSnark := make([][]uint64,1,1)
	for _,proofdata := range ResponseData {
		ProofForSnark = append(ProofForSnark,proofdata.VanillaProof)
		IndexForSnark = append(IndexForSnark,getindex(proofdata.Index))
	}


	if sectorslen != len(ProofInOrder) {
		log.Info("Error:: proof len is incorrect!!!!!!!!!!!!!!!!!")
	}

	log.Info("[GenerateWindowPoStPlus] ProofForSnark len = ",len(ProofForSnark))

	//TODO:: do Snark Proof
	//m.SnarkWindowPoSt(ProofInOrder)
	log.Info("[GenerateWindowPoStPlus] Start -> GenerateWindowPoStSnark Local Snark Merge")
	result,_,_ := m.GenerateWindowPoStSnark(ctx,minerID,ProofForSnark,IndexForSnark)

	log.Info("[GenerateWindowPoStPlus] SnarkWindowPoStM Local Snark Merge Finish ")
	return result, nil, nil
}

func (sb *Sealer)GenerateWindowPoStVanilla(ctx context.Context, minerID abi.ActorID, sectorInfo []abi.SectorInfo, randomness abi.PoStRandomness, index []uint64) (proof []abi.PoStProof, skipped []abi.SectorID, err error){
	randomness[31] &= 0x3f
	privsectors, skipped, done, err := sb.pubSectorToPriv(ctx, minerID, sectorInfo, nil, abi.RegisteredSealProof.RegisteredWindowPoStProof)
	if err != nil {
		return nil, nil, xerrors.Errorf("gathering sector info: %w", err)
	}
	defer done()

	// TODO:: Replace with new ffi
	proof, err = ffi.GenerateWindowPoSt(minerID, privsectors, randomness)
	return proof, skipped, err
}

func (sb *Sealer)GenerateWindowPoStSnark(ctx context.Context, minerID abi.ActorID, sectorInfo [][]abi.PoStProof, index [][]uint64) (proof []abi.PoStProof, skipped []abi.SectorID, err error){
	log.Info("[GenerateWindowPoStSnark] GenerateWindowPoStSnark start")
	// TODO:: Replace with new ffi
	return nil, nil, nil
}

func (sb *Sealer) pubSectorToPriv(ctx context.Context, mid abi.ActorID, sectorInfo []abi.SectorInfo, faults []abi.SectorNumber, rpt func(abi.RegisteredSealProof) (abi.RegisteredPoStProof, error)) (ffi.SortedPrivateSectorInfo, []abi.SectorID, func(), error) {
	fmap := map[abi.SectorNumber]struct{}{}
	for _, fault := range faults {
		fmap[fault] = struct{}{}
	}

	var doneFuncs []func()
	done := func() {
		for _, df := range doneFuncs {
			df()
		}
	}

	var skipped []abi.SectorID
	var out []ffi.PrivateSectorInfo
	for _, s := range sectorInfo {
		if _, faulty := fmap[s.SectorNumber]; faulty {
			continue
		}

		sid := abi.SectorID{Miner: mid, Number: s.SectorNumber}

		paths, d, err := sb.sectors.AcquireSector(ctx, sid, stores.FTCache|stores.FTSealed, 0, stores.PathStorage)
		if err != nil {
			log.Warnw("failed to acquire sector, skipping", "sector", sid, "error", err)
			skipped = append(skipped, sid)
			continue
		}
		doneFuncs = append(doneFuncs, d)

		postProofType, err := rpt(s.SealProof)
		if err != nil {
			done()
			return ffi.SortedPrivateSectorInfo{}, nil, nil, xerrors.Errorf("acquiring registered PoSt proof from sector info %+v: %w", s, err)
		}

		out = append(out, ffi.PrivateSectorInfo{
			CacheDirPath:     paths.Cache,
			PoStProofType:    postProofType,
			SealedSectorPath: paths.Sealed,
			SectorInfo:       s,
		})
	}

	return ffi.NewSortedPrivateSectorInfo(out...), skipped, done, nil
}

var _ Verifier = ProofVerifier

type proofVerifier struct{}

var ProofVerifier = proofVerifier{}

func (proofVerifier) VerifySeal(info abi.SealVerifyInfo) (bool, error) {
	return ffi.VerifySeal(info)
}

func (proofVerifier) VerifyWinningPoSt(ctx context.Context, info abi.WinningPoStVerifyInfo) (bool, error) {
	info.Randomness[31] &= 0x3f
	_, span := trace.StartSpan(ctx, "VerifyWinningPoSt")
	defer span.End()

	return ffi.VerifyWinningPoSt(info)
}

func (proofVerifier) VerifyWindowPoSt(ctx context.Context, info abi.WindowPoStVerifyInfo) (bool, error) {
	info.Randomness[31] &= 0x3f
	_, span := trace.StartSpan(ctx, "VerifyWindowPoSt")
	defer span.End()

	return ffi.VerifyWindowPoSt(info)
}

func (proofVerifier) GenerateWinningPoStSectorChallenge(ctx context.Context, proofType abi.RegisteredPoStProof, minerID abi.ActorID, randomness abi.PoStRandomness, eligibleSectorCount uint64) ([]uint64, error) {
	randomness[31] &= 0x3f
	return ffi.GenerateWinningPoStSectorChallenge(proofType, minerID, randomness, eligibleSectorCount)
}


func getindex(a []rpcclient.SectorIdIndex) []uint64 {
	index := make([]uint64,1,1)
	for _,ind := range a{
		index = append(index,ind.PrzIndex)
	}
	return index
}