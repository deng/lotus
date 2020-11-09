package config

import (
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
)

type StorageDealer struct {
	Common

	Dealmaking DealmakingConfig
	Fees       MinerFeeConfig
}

func DefaultStorageDealer() *StorageDealer {
	cfg := &StorageDealer{
		Common: defCommon(),

		Dealmaking: DealmakingConfig{
			ConsiderOnlineStorageDeals:    true,
			ConsiderOfflineStorageDeals:   true,
			ConsiderOnlineRetrievalDeals:  true,
			ConsiderOfflineRetrievalDeals: true,
			PieceCidBlocklist:             []cid.Cid{},
			// TODO: It'd be nice to set this based on sector size
			ExpectedSealDuration: Duration(time.Hour * 24),
		},

		Fees: MinerFeeConfig{
			MaxPreCommitGasFee:     types.MustParseFIL("0.025"),
			MaxCommitGasFee:        types.MustParseFIL("0.05"),
			MaxWindowPoStGasFee:    types.MustParseFIL("5"),
			MaxPublishDealsFee:     types.MustParseFIL("0.05"),
			MaxMarketBalanceAddFee: types.MustParseFIL("0.007"),
		},
	}
	cfg.Common.API.ListenAddress = "/ip4/127.0.0.1/tcp/2335/http"
	cfg.Common.API.RemoteListenAddress = "127.0.0.1:2335"
	return cfg
}
