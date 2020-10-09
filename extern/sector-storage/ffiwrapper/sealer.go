package ffiwrapper

import (
	"github.com/filecoin-project/go-state-types/abi"
	idstore "github.com/filecoin-project/lotus/extern/sector-id-store"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("ffiwrapper")

type Sealer struct {
	sealProofType abi.RegisteredSealProof
	ssize         abi.SectorSize // a function of sealProofType and postProofType

	sectors  SectorProvider
	stopping chan struct{}
	ids idstore.SectorIpRecord
}

func (sb *Sealer) Stop() {
	close(sb.stopping)
}

func (sb *Sealer) SectorSize() abi.SectorSize {
	return sb.ssize
}

func (sb *Sealer) SealProofType() abi.RegisteredSealProof {
	return sb.sealProofType
}
