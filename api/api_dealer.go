package api

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/piecestore"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
)

type StorageDealer interface {
	Common
	ActorAddress(context.Context) (address.Address, error)

	PiecesListPieces(ctx context.Context) ([]cid.Cid, error)
	PiecesListCidInfos(ctx context.Context) ([]cid.Cid, error)
	PiecesGetPieceInfo(ctx context.Context, pieceCid cid.Cid) (*piecestore.PieceInfo, error)
	PiecesGetCIDInfo(ctx context.Context, payloadCid cid.Cid) (*piecestore.CIDInfo, error)

	MarketImportDealData(ctx context.Context, propcid cid.Cid, path string) error
	MarketListDeals(ctx context.Context) ([]MarketDeal, error)
	MarketListRetrievalDeals(ctx context.Context) ([]retrievalmarket.ProviderDealState, error)
	MarketGetDealUpdates(ctx context.Context) (<-chan storagemarket.MinerDeal, error)
	MarketListIncompleteDeals(ctx context.Context) ([]storagemarket.MinerDeal, error)
	MarketSetAsk(ctx context.Context, price types.BigInt, verifiedPrice types.BigInt, duration abi.ChainEpoch, minPieceSize abi.PaddedPieceSize, maxPieceSize abi.PaddedPieceSize) error
	MarketGetAsk(ctx context.Context) (*storagemarket.SignedStorageAsk, error)
	MarketSetRetrievalAsk(ctx context.Context, rask *retrievalmarket.Ask) error
	MarketGetRetrievalAsk(ctx context.Context) (*retrievalmarket.Ask, error)
	MarketListDataTransfers(ctx context.Context) ([]DataTransferChannel, error)
	MarketDataTransferUpdates(ctx context.Context) (<-chan DataTransferChannel, error)
	// MinerRestartDataTransfer attempts to restart a data transfer with the given transfer ID and other peer
	MarketRestartDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error
	// ClientCancelDataTransfer cancels a data transfer with the given transfer ID and other peer
	MarketCancelDataTransfer(ctx context.Context, transferID datatransfer.TransferID, otherPeer peer.ID, isInitiator bool) error

	DealsImportData(ctx context.Context, dealPropCid cid.Cid, file string) error
	DealsList(ctx context.Context) ([]MarketDeal, error)
	DealsConsiderOnlineStorageDeals(context.Context) (bool, error)
	DealsSetConsiderOnlineStorageDeals(context.Context, bool) error
	DealsConsiderOnlineRetrievalDeals(context.Context) (bool, error)
	DealsSetConsiderOnlineRetrievalDeals(context.Context, bool) error
	DealsPieceCidBlocklist(context.Context) ([]cid.Cid, error)
	DealsSetPieceCidBlocklist(context.Context, []cid.Cid) error
	DealsConsiderOfflineStorageDeals(context.Context) (bool, error)
	DealsSetConsiderOfflineStorageDeals(context.Context, bool) error
	DealsConsiderOfflineRetrievalDeals(context.Context) (bool, error)
	DealsSetConsiderOfflineRetrievalDeals(context.Context, bool) error

	CreateBackup(ctx context.Context, fpath string) error
}
