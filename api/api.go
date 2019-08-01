package api

import (
	"context"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/filecoin-project/go-lotus/chain"
	"github.com/filecoin-project/go-lotus/chain/address"
	"github.com/filecoin-project/go-lotus/chain/store"
	"github.com/filecoin-project/go-lotus/chain/types"

	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-filestore"
)

// Version provides various build-time information
type Version struct {
	Version string

	// APIVersion is a binary encoded semver version of the remote implementing
	// this api
	//
	// See APIVersion in build/version.go
	APIVersion uint32

	// TODO: git commit / os / genesis cid?
}

type Import struct {
	Status   filestore.Status
	Key      cid.Cid
	FilePath string
	Size     uint64
}

type MsgWait struct {
	InBlock cid.Cid
	Receipt types.MessageReceipt
}

type Common interface {
	// Auth
	AuthVerify(ctx context.Context, token string) ([]string, error)
	AuthNew(ctx context.Context, perms []string) ([]byte, error)

	// network

	NetConnectedness(context.Context, peer.ID) (network.Connectedness, error)
	NetPeers(context.Context) ([]peer.AddrInfo, error)
	NetConnect(context.Context, peer.AddrInfo) error
	NetAddrsListen(context.Context) (peer.AddrInfo, error)
	NetDisconnect(context.Context, peer.ID) error

	// ID returns peerID of libp2p node backing this API
	ID(context.Context) (peer.ID, error)

	// Version provides information about API provider
	Version(context.Context) (Version, error)
}

// FullNode API is a low-level interface to the Filecoin network full node
type FullNode interface {
	Common

	// chain
	ChainNotify(context.Context) (<-chan *store.HeadChange, error)
	ChainHead(context.Context) (*types.TipSet, error)                // TODO: check serialization
	ChainSubmitBlock(ctx context.Context, blk *chain.BlockMsg) error // TODO: check serialization
	ChainGetRandomness(context.Context, *types.TipSet) ([]byte, error)
	ChainWaitMsg(context.Context, cid.Cid) (*MsgWait, error)
	ChainGetBlock(context.Context, cid.Cid) (*types.BlockHeader, error)
	ChainGetBlockMessages(context.Context, cid.Cid) ([]*types.Message, []*types.SignedMessage, error)

	// if tipset is nil, we'll use heaviest
	ChainCall(context.Context, *types.Message, *types.TipSet) (*types.MessageReceipt, error)

	// messages

	MpoolPending(context.Context, *types.TipSet) ([]*types.SignedMessage, error)
	MpoolPush(context.Context, *types.SignedMessage) error
	MpoolGetNonce(context.Context, address.Address) (uint64, error)

	// FullNodeStruct

	// miner

	MinerStart(context.Context, address.Address) error
	MinerCreateBlock(context.Context, address.Address, *types.TipSet, []types.Ticket, types.ElectionProof, []*types.SignedMessage) (*chain.BlockMsg, error)

	// // UX ?

	// wallet

	WalletNew(context.Context, string) (address.Address, error)
	WalletList(context.Context) ([]address.Address, error)
	WalletBalance(context.Context, address.Address) (types.BigInt, error)
	WalletSign(context.Context, address.Address, []byte) (*types.Signature, error)
	WalletDefaultAddress(context.Context) (address.Address, error)

	// Other

	// ClientImport imports file under the specified path into filestore
	ClientImport(ctx context.Context, path string) (cid.Cid, error)

	// ClientUnimport removes references to the specified file from filestore
	//ClientUnimport(path string)

	// ClientListImports lists imported files and their root CIDs
	ClientListImports(ctx context.Context) ([]Import, error)

	//ClientListAsks() []Ask
}

// Full API is a low-level interface to the Filecoin network storage miner node
type StorageMiner interface {
	Common

	// Temp api for testing
	StoreGarbageData(context.Context) (uint64, error)

	// Get the status of a given sector by ID
	SectorsStatus(context.Context, uint64) (sectorbuilder.SectorSealingStatus, error)

	// List all staged sectors
	SectorsStagedList(context.Context) ([]sectorbuilder.StagedSectorMetadata, error)

	// Seal all staged sectors
	SectorsStagedSeal(context.Context) error
}
