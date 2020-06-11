package dtypes

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/abi"
)

type MinerAddress address.Address
type MinerID abi.ActorID

// IsAcceptingStorageDealsFunc is a function which reads from miner config to
// determine if the user has disabled storage deals (or not).
type IsAcceptingStorageDealsFunc func() (bool, error)
