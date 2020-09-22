package storage

import (
	"context"
	"errors"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/host"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/config"
)

type Dealer struct {
	api    storageDealerApi
	feeCfg config.MinerFeeConfig
	h      host.Host
	ds     datastore.Batching

	maddr  address.Address
	worker address.Address
}

type storageDealerApi interface {
	StateMarketStorageDeal(context.Context, abi.DealID, types.TipSetKey) (*api.MarketDeal, error)
	WalletHas(context.Context, address.Address) (bool, error)
}

//TODO: 使用Dealer替换Miner
func NewDealer(api storageMinerApi, maddr, worker address.Address, h host.Host, ds datastore.Batching, feeCfg config.MinerFeeConfig) (*Miner, error) {
	m := &Miner{
		api:    api,
		feeCfg: feeCfg,
		h:      h,
		ds:     ds,

		maddr:  maddr,
		worker: worker,
	}

	return m, nil
}

func (m *Miner) RunDealer(ctx context.Context) error {
	if err := m.runPreflightChecksDealer(ctx); err != nil {
		return xerrors.Errorf("miner preflight checks failed: %w", err)
	}
	return nil
}

func (m *Miner) StopDealer(ctx context.Context) error {
	return nil
}

func (m *Miner) runPreflightChecksDealer(ctx context.Context) error {
	has, err := m.api.WalletHas(ctx, m.worker)
	if err != nil {
		return xerrors.Errorf("failed to check wallet for worker key: %w", err)
	}

	if !has {
		return errors.New("key for worker not found in local wallet")
	}

	log.Infof("starting up dealer with miner ID %s, worker addr %s", m.maddr, m.worker)
	return nil
}

func (m *Miner) ActorAddress() address.Address {
	return m.maddr
}
