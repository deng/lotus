// +build !debug
// +build !2k
// +build !testground

package build

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
)

const UpgradeBreezeHeight = -1
const BreezeGasTampingDuration = 0

const UpgradeSmokeHeight = -1

var DrandSchedule = map[abi.ChainEpoch]DrandEnum{
	0: DrandMainnet,
}

func init() {
	//power.ConsensusMinerMinPower = big.NewInt(1 << 30)
	power.ConsensusMinerMinPower = big.NewInt(8 << 20)
	miner.SupportedProofTypes = map[abi.RegisteredSealProof]struct{}{
		abi.RegisteredSealProof_StackedDrg8MiBV1:   {},
		abi.RegisteredSealProof_StackedDrg512MiBV1: {},
		abi.RegisteredSealProof_StackedDrg32GiBV1:  {},
		abi.RegisteredSealProof_StackedDrg64GiBV1:  {},
	}
}

const BlockDelaySecs = uint64(builtin.EpochDurationSeconds)

const PropagationDelaySecs = uint64(6)
