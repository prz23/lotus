//+build cgo

package ffiwrapper

import (
	"bytes"
	"context"
	//state "github.com/filecoin-project/lotus/extern/miningstate/rpcsectorstate"
	idstore "github.com/filecoin-project/lotus/extern/sector-id-store"
	"net/rpc"

	"github.com/filecoin-project/specs-actors/actors/runtime/proof"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"

	ffi "github.com/filecoin-project/filecoin-ffi"

	"github.com/filecoin-project/lotus/extern/sector-storage/stores"

	"github.com/filecoin-project/lotus/extern/miningstate/rpcclient"
	"go.opencensus.io/trace"
	"sort"

	proof0 "github.com/filecoin-project/specs-actors/actors/runtime/proof"

)

func (sb *Sealer) GenerateWinningPoSt(ctx context.Context, minerID abi.ActorID, sectorInfo []proof.SectorInfo, randomness abi.PoStRandomness) ([]proof.PoStProof, error) {
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

func (sb *Sealer) GenerateWindowPoSt(ctx context.Context, minerID abi.ActorID, sectorInfo []proof.SectorInfo, randomness abi.PoStRandomness) ([]proof.PoStProof, []abi.SectorID, error) {
	//randomness[31] &= 0x3f
	//privsectors, skipped, done, err := sb.pubSectorToPriv(ctx, minerID, sectorInfo, nil, abi.RegisteredSealProof.RegisteredWindowPoStProof)
	//if err != nil {
	//	return nil, nil, xerrors.Errorf("gathering sector info: %w", err)
	//}
	//defer done()
	//
	//if len(skipped) > 0 {
	//	return nil, skipped, xerrors.Errorf("pubSectorToPriv skipped some sectors")
	//}
	//
	//proof, faulty, err := ffi.GenerateWindowPoSt(minerID, privsectors, randomness)
	//
	//var faultyIDs []abi.SectorID
	//for _, f := range faulty {
	//	faultyIDs = append(faultyIDs, abi.SectorID{
	//		Miner:  minerID,
	//		Number: f,
	//	})
	//}
	//
	//return proof, faultyIDs, err
	return nil, nil, nil
}

func (m *Sealer) GenerateWindowPoStPlus(ctx context.Context, minerID abi.ActorID, sectorInfo []proof0.SectorInfo, randomness abi.PoStRandomness) (proof []proof0.PoStProof, skipped []abi.SectorID, err error) {

	sectorslen := len(sectorInfo)

	log.Info("[GenerateWindowPoStPlus] sector len = ",sectorslen)
	log.Infow("[GenerateWindowPoStPlus]  ",
		"sector len =",sectorslen)

	var SortedSectorNumber sort.IntSlice
	SectorNumberSectorInfoMap := make(map[abi.SectorNumber]proof0.SectorInfo)
	for _, SectorInfo := range sectorInfo{
		SortedSectorNumber = append(SortedSectorNumber,int(SectorInfo.SectorNumber))
		SectorNumberSectorInfoMap[SectorInfo.SectorNumber] = SectorInfo
	}
	//  SortedSectorNumber is now in increasing order
	SortedSectorNumber.Sort()

	log.Info("[GenerateWindowPoStPlus] SortedSectorNumber = ",SortedSectorNumber)

	var query []idstore.SectorId
	sectormap := make(map[int]int)  // map[SortedSectorNumber]przindex

	for przindex, secortnumber := range SortedSectorNumber {
		sectormap[secortnumber] = przindex
		query = append(query,idstore.SectorId(secortnumber))
	}

	secipmap, _ := m.ids.FindAllSort(query)
	//stateinstance := state.GetIns()
	//secipmap := stateinstance.FindAllSort(query)

	log.Info("[GenerateWindowPoStPlus] SlaveIP & Its SectorNumbers = ",SortedSectorNumber)

	var chanlist []chan *rpc.Call

	for ip,sectorids := range secipmap {

		var localindex []ffi.SectorIndex
		var localcectorinfo [][]byte

		for _,id := range sectorids {
			localindex = append(localindex, ffi.SectorIndex{ SectorNum: abi.SectorNumber(id),
				Index: uint64(sectormap[int(id)])})

			tomasrshal := SectorNumberSectorInfoMap[abi.SectorNumber(id)]
			buf := new(bytes.Buffer)
			if err := tomasrshal.MarshalCBOR(buf); err != nil {
				return nil,nil,nil
			}

			localcectorinfo = append(localcectorinfo, buf.Bytes())
		}

		log.Info("[GenerateWindowPoStPlus] WindowPoStRequest localcectorinfo = ",localcectorinfo)
		log.Info("[GenerateWindowPoStPlus] WindowPoStRequest localindex = ",localindex)

		index := ffi.SectorIndexInfo{Indexes:localindex}
		indexbytes,_ :=index.MarshalJSON()

		req := rpcclient.WindowPoStRequest{MinerID:  uint64(minerID), SectorInfo: localcectorinfo,
			Randomness: randomness, Index:indexbytes}
		response := rpcclient.RpcCallWindowPoStSync(string(ip),"WindowPoSt.DoVanillaProof",req)
		chanlist = append(chanlist,response)
	}

	log.Info("[GenerateWindowPoStPlus] WindowPoStRequest 1 ")
	var ResponseData []*rpcclient.WindowPoStResponse
	for _,eachchan := range chanlist{
		select {
		case data := <-eachchan:{
			    log.Info("========[GenerateWindowPoStPlus]==========")
			    ResponseData = append(ResponseData,data.Reply.(*rpcclient.WindowPoStResponse))
		    }
		}
	}
	// all channels are returned

	log.Info("[GenerateWindowPoStPlus] WindowPoStRequest 2")

	var allproofs []proof0.PoStProof
	var allindexinfo []ffi.SectorIndexInfo
	var allskipped []abi.SectorID
	for _,proofdata := range ResponseData{
		for _,proof := range proofdata.VanillaProof{
			var v proof0.PoStProof
			if err := v.UnmarshalCBOR(bytes.NewBuffer(proof)); err != nil {
				return nil,nil,nil
			}
			allproofs = append(allproofs,v)
		}

		var SectorIndexInfo ffi.SectorIndexInfo
		_ = SectorIndexInfo.UnmarshalJSON(proofdata.Index)

		allindexinfo = append(allindexinfo,SectorIndexInfo)

		for _,skip := range proofdata.Skipped{
			var v abi.SectorID
			if err := v.UnmarshalCBOR(bytes.NewBuffer(skip)); err != nil {
				return nil,nil,nil
			}
			allskipped = append(allskipped,v)
		}
	}


	log.Info("[GenerateWindowPoStPlus] Start -> GenerateWindowPoStSnark Local Snark Merge")
	result,_,_ := m.GenerateWindowPoStSnark(ctx,minerID,sectorInfo,randomness,allproofs,allindexinfo)

	log.Info("[GenerateWindowPoStPlus] SnarkWindowPoStM Local Snark Merge Finish ")
	return result, allskipped, nil
}

func (sb *Sealer)GenerateWindowPoStVanilla(ctx context.Context, minerID abi.ActorID, sectorInfo []proof0.SectorInfo, randomness abi.PoStRandomness, index ffi.SectorIndexInfo) (proof []proof0.PoStProof, skipped []abi.SectorID, err error){
	randomness[31] &= 0x3f
	privsectors, skipped, done, err := sb.pubSectorToPriv(ctx, minerID, sectorInfo, nil, abi.RegisteredSealProof.RegisteredWindowPoStProof)
	if err != nil {
		return nil, nil, xerrors.Errorf("gathering sector info: %w", err)
	}
	defer done()

	// TODO:: Replace with new ffi
	proof, err = ffi.GenerateWindowPoStVanilla(minerID, privsectors, randomness, index)
	return proof, skipped, err
}

func (sb *Sealer)GenerateWindowPoStSnark(ctx context.Context, minerID abi.ActorID, sectorInfo []proof0.SectorInfo,randomness abi.PoStRandomness, proofs []proof0.PoStProof,index []ffi.SectorIndexInfo) (proof []proof0.PoStProof, skipped []abi.SectorID, err error){
	log.Info("[GenerateWindowPoStSnark] GenerateWindowPoStSnark start")
	randomness[31] &= 0x3f
	proof, err = ffi.GenerateWindowPoStSnark(minerID, sectorInfo, randomness,ffi.VanillaProofs{Proofs: proofs},ffi.VanillaInfos{Infos:index})

	return proof, nil, nil
}

func (sb *Sealer) pubSectorToPriv(ctx context.Context, mid abi.ActorID, sectorInfo []proof0.SectorInfo, faults []abi.SectorNumber, rpt func(abi.RegisteredSealProof) (abi.RegisteredPoStProof, error)) (ffi.SortedPrivateSectorInfo, []abi.SectorID, func(), error) {
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

func (proofVerifier) VerifySeal(info proof.SealVerifyInfo) (bool, error) {
	return ffi.VerifySeal(info)
}

func (proofVerifier) VerifyWinningPoSt(ctx context.Context, info proof.WinningPoStVerifyInfo) (bool, error) {
	info.Randomness[31] &= 0x3f
	_, span := trace.StartSpan(ctx, "VerifyWinningPoSt")
	defer span.End()

	return ffi.VerifyWinningPoSt(info)
}

func (proofVerifier) VerifyWindowPoSt(ctx context.Context, info proof.WindowPoStVerifyInfo) (bool, error) {
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