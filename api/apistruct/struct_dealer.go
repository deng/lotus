package apistruct

import (
	"context"

	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/piecestore"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

type StorageDealerStruct struct {
	CommonStruct

	Internal struct {
		ActorAddress func(context.Context) (address.Address, error) `perm:"read"`

		PiecesListPieces   func(ctx context.Context) ([]cid.Cid, error)                               `perm:"read"`
		PiecesListCidInfos func(ctx context.Context) ([]cid.Cid, error)                               `perm:"read"`
		PiecesGetPieceInfo func(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error) `perm:"read"`
		PiecesGetCIDInfo   func(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error) `perm:"read"`

		MarketImportDealData      func(context.Context, cid.Cid, string) error                                                                                                                                 `perm:"write"`
		MarketListDeals           func(ctx context.Context) ([]api.MarketDeal, error)                                                                                                                          `perm:"read"`
		MarketListRetrievalDeals  func(ctx context.Context) ([]retrievalmarket.ProviderDealState, error)                                                                                                       `perm:"read"`
		MarketGetDealUpdates      func(ctx context.Context) (<-chan storagemarket.MinerDeal, error)                                                                                                            `perm:"read"`
		MarketListIncompleteDeals func(ctx context.Context) ([]storagemarket.MinerDeal, error)                                                                                                                 `perm:"read"`
		MarketSetAsk              func(ctx context.Context, price types.BigInt, verifiedPrice types.BigInt, duration abi.ChainEpoch, minPieceSize abi.PaddedPieceSize, maxPieceSize abi.PaddedPieceSize) error `perm:"admin"`
		MarketGetAsk              func(ctx context.Context) (*storagemarket.SignedStorageAsk, error)                                                                                                           `perm:"read"`
		MarketSetRetrievalAsk     func(ctx context.Context, rask *retrievalmarket.Ask) error                                                                                                                   `perm:"admin"`
		MarketGetRetrievalAsk     func(ctx context.Context) (*retrievalmarket.Ask, error)                                                                                                                      `perm:"read"`
		MarketListDataTransfers   func(ctx context.Context) ([]api.DataTransferChannel, error)                                                                                                                 `perm:"write"`
		MarketDataTransferUpdates func(ctx context.Context) (<-chan api.DataTransferChannel, error)                                                                                                            `perm:"write"`
		MarketRestartDataTransfer func(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error                                                                     `perm:"read"`
		MarketCancelDataTransfer  func(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error                                                                     `perm:"read"`

		DealsImportData                       func(ctx context.Context, dealPropCid cid.Cid, file string) error `perm:"write"`
		DealsList                             func(ctx context.Context) ([]api.MarketDeal, error)               `perm:"read"`
		DealsConsiderOnlineStorageDeals       func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderOnlineStorageDeals    func(context.Context, bool) error                                 `perm:"admin"`
		DealsConsiderOnlineRetrievalDeals     func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderOnlineRetrievalDeals  func(context.Context, bool) error                                 `perm:"admin"`
		DealsConsiderOfflineStorageDeals      func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderOfflineStorageDeals   func(context.Context, bool) error                                 `perm:"admin"`
		DealsConsiderOfflineRetrievalDeals    func(context.Context) (bool, error)                               `perm:"read"`
		DealsSetConsiderOfflineRetrievalDeals func(context.Context, bool) error                                 `perm:"admin"`
		DealsPieceCidBlocklist                func(context.Context) ([]cid.Cid, error)                          `perm:"read"`
		DealsSetPieceCidBlocklist             func(context.Context, []cid.Cid) error                            `perm:"admin"`

		CreateBackup func(ctx context.Context, fpath string) error `perm:"admin"`
	}
}

func (c *StorageDealerStruct) ActorAddress(ctx context.Context) (address.Address, error) {
	return c.Internal.ActorAddress(ctx)
}

func (c *StorageDealerStruct) PiecesListPieces(ctx context.Context) ([]cid.Cid, error) {
	return c.Internal.PiecesListPieces(ctx)
}

func (c *StorageDealerStruct) PiecesListCidInfos(ctx context.Context) ([]cid.Cid, error) {
	return c.Internal.PiecesListCidInfos(ctx)
}

func (c *StorageDealerStruct) PiecesGetPieceInfo(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error) {
	return c.Internal.PiecesGetPieceInfo(ctx, pieceCid)
}

func (c *StorageDealerStruct) PiecesGetCIDInfo(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error) {
	return c.Internal.PiecesGetCIDInfo(ctx, payloadCid)
}

func (c *StorageDealerStruct) MarketImportDealData(ctx context.Context, propcid cid.Cid, path string) error {
	return c.Internal.MarketImportDealData(ctx, propcid, path)
}

func (c *StorageDealerStruct) MarketListDeals(ctx context.Context) ([]api.MarketDeal, error) {
	return c.Internal.MarketListDeals(ctx)
}

func (c *StorageDealerStruct) MarketListRetrievalDeals(ctx context.Context) ([]retrievalmarket.ProviderDealState, error) {
	return c.Internal.MarketListRetrievalDeals(ctx)
}

func (c *StorageDealerStruct) MarketGetDealUpdates(ctx context.Context) (<-chan storagemarket.MinerDeal, error) {
	return c.Internal.MarketGetDealUpdates(ctx)
}

func (c *StorageDealerStruct) MarketListIncompleteDeals(ctx context.Context) ([]storagemarket.MinerDeal, error) {
	return c.Internal.MarketListIncompleteDeals(ctx)
}

func (c *StorageDealerStruct) MarketSetAsk(ctx context.Context, price types.BigInt, verifiedPrice types.BigInt, duration abi.ChainEpoch, minPieceSize abi.PaddedPieceSize, maxPieceSize abi.PaddedPieceSize) error {
	return c.Internal.MarketSetAsk(ctx, price, verifiedPrice, duration, minPieceSize, maxPieceSize)
}

func (c *StorageDealerStruct) MarketGetAsk(ctx context.Context) (*storagemarket.SignedStorageAsk, error) {
	return c.Internal.MarketGetAsk(ctx)
}

func (c *StorageDealerStruct) MarketSetRetrievalAsk(ctx context.Context, rask *retrievalmarket.Ask) error {
	return c.Internal.MarketSetRetrievalAsk(ctx, rask)
}

func (c *StorageDealerStruct) MarketGetRetrievalAsk(ctx context.Context) (*retrievalmarket.Ask, error) {
	return c.Internal.MarketGetRetrievalAsk(ctx)
}

func (c *StorageDealerStruct) MarketListDataTransfers(ctx context.Context) ([]api.DataTransferChannel, error) {
	return c.Internal.MarketListDataTransfers(ctx)
}

func (c *StorageDealerStruct) MarketDataTransferUpdates(ctx context.Context) (<-chan api.DataTransferChannel, error) {
	return c.Internal.MarketDataTransferUpdates(ctx)
}

func (c *StorageDealerStruct) MarketRestartDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error {
	return c.Internal.MarketRestartDataTransfer(ctx, transferID, otherPeer, isInitiator)
}

func (c *StorageDealerStruct) MarketCancelDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error {
	return c.Internal.MarketCancelDataTransfer(ctx, transferID, otherPeer, isInitiator)
}

func (c *StorageDealerStruct) DealsImportData(ctx context.Context, dealPropCid cid.Cid, file string) error {
	return c.Internal.DealsImportData(ctx, dealPropCid, file)
}

func (c *StorageDealerStruct) DealsList(ctx context.Context) ([]api.MarketDeal, error) {
	return c.Internal.DealsList(ctx)
}

func (c *StorageDealerStruct) DealsConsiderOnlineStorageDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderOnlineStorageDeals(ctx)
}

func (c *StorageDealerStruct) DealsSetConsiderOnlineStorageDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderOnlineStorageDeals(ctx, b)
}

func (c *StorageDealerStruct) DealsConsiderOnlineRetrievalDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderOnlineRetrievalDeals(ctx)
}

func (c *StorageDealerStruct) DealsSetConsiderOnlineRetrievalDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderOnlineRetrievalDeals(ctx, b)
}

func (c *StorageDealerStruct) DealsPieceCidBlocklist(ctx context.Context) ([]cid.Cid, error) {
	return c.Internal.DealsPieceCidBlocklist(ctx)
}

func (c *StorageDealerStruct) DealsSetPieceCidBlocklist(ctx context.Context, cids []cid.Cid) error {
	return c.Internal.DealsSetPieceCidBlocklist(ctx, cids)
}

func (c *StorageDealerStruct) DealsConsiderOfflineStorageDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderOfflineStorageDeals(ctx)
}

func (c *StorageDealerStruct) DealsSetConsiderOfflineStorageDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderOfflineStorageDeals(ctx, b)
}

func (c *StorageDealerStruct) DealsConsiderOfflineRetrievalDeals(ctx context.Context) (bool, error) {
	return c.Internal.DealsConsiderOfflineRetrievalDeals(ctx)
}

func (c *StorageDealerStruct) DealsSetConsiderOfflineRetrievalDeals(ctx context.Context, b bool) error {
	return c.Internal.DealsSetConsiderOfflineRetrievalDeals(ctx, b)
}

func (c *StorageDealerStruct) CreateBackup(ctx context.Context, fpath string) error {
	return c.Internal.CreateBackup(ctx, fpath)
}
