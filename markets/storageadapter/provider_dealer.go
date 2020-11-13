package storageadapter

// this file implements storagemarket.StorageProviderNode

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	market2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/exitcode"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/builtin/market"
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/chain/events/state"
	"github.com/filecoin-project/lotus/chain/types"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/filecoin-project/lotus/lib/sigs"
	"github.com/filecoin-project/lotus/markets/utils"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
)

type ProviderNodeAdapterDealer struct {
	api.FullNode

	// this goes away with the data transfer module
	dag dtypes.StagingDAG

	sealingApi api.Sealer
	ev         *events.Events

	publishSpec, addBalanceSpec *api.MessageSendSpec
	dsMatcher                   *dealStateMatcher
}

func NewProviderNodeAdapterDealer(fc *config.MinerFeeConfig) func(dag dtypes.StagingDAG, sealingApi api.Sealer, full api.FullNode) storagemarket.StorageProviderNode {
	return func(dag dtypes.StagingDAG, sealingApi api.Sealer, full api.FullNode) storagemarket.StorageProviderNode {
		na := &ProviderNodeAdapterDealer{
			FullNode: full,

			dag:        dag,
			sealingApi: sealingApi,
			ev:         events.NewEvents(context.TODO(), full),
			dsMatcher:  newDealStateMatcher(state.NewStatePredicates(full)),
		}
		if fc != nil {
			na.publishSpec = &api.MessageSendSpec{MaxFee: abi.TokenAmount(fc.MaxPublishDealsFee)}
			na.addBalanceSpec = &api.MessageSendSpec{MaxFee: abi.TokenAmount(fc.MaxMarketBalanceAddFee)}
		}
		return na
	}
}

func (n *ProviderNodeAdapterDealer) PublishDeals(ctx context.Context, deal storagemarket.MinerDeal) (cid.Cid, error) {
	log.Info("publishing deal")

	mi, err := n.StateMinerInfo(ctx, deal.Proposal.Provider, types.EmptyTSK)
	if err != nil {
		return cid.Undef, err
	}

	params, err := actors.SerializeParams(&market2.PublishStorageDealsParams{
		Deals: []market2.ClientDealProposal{deal.ClientDealProposal},
	})

	if err != nil {
		return cid.Undef, xerrors.Errorf("serializing PublishStorageDeals params failed: %w", err)
	}

	// TODO: We may want this to happen after fetching data
	smsg, err := n.MpoolPushMessage(ctx, &types.Message{
		To:     market.Address,
		From:   mi.Worker,
		Value:  types.NewInt(0),
		Method: market.Methods.PublishStorageDeals,
		Params: params,
	}, n.publishSpec)
	if err != nil {
		return cid.Undef, err
	}
	return smsg.Cid(), nil
}

func (n *ProviderNodeAdapterDealer) OnDealComplete(ctx context.Context, deal storagemarket.MinerDeal, pieceSize abi.UnpaddedPieceSize, pieceData io.Reader) (*storagemarket.PackingResult, error) {
	if deal.PublishCid == nil {
		return nil, xerrors.Errorf("deal.PublishCid can't be nil")
	}

	sdInfo := sealing.DealInfo{
		DealID:     deal.DealID,
		PublishCid: deal.PublishCid,
		DealSchedule: sealing.DealSchedule{
			StartEpoch: deal.ClientDealProposal.Proposal.StartEpoch,
			EndEpoch:   deal.ClientDealProposal.Proposal.EndEpoch,
		},
		KeepUnsealed: deal.FastRetrieval,
	}

	piecePath, haPiecePath := os.LookupEnv("LOTUS_PIECE_PATH")
	if !haPiecePath || piecePath == "" {
		piecePath = "/tmp/piece-data"
	}

	if _, err := os.Stat(piecePath); os.IsNotExist(err) {
		os.MkdirAll(piecePath, os.ModePerm)
	}

	filename := path.Join(piecePath, strconv.FormatUint(uint64(deal.DealID), 10))
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	var bufferRead bytes.Buffer
	tee := io.TeeReader(pieceData, &bufferRead)
	buf := make([]byte, 1024)
	for {
		// read a chunk
		n, err := tee.Read(buf)
		if err != nil && err != io.EOF {
			file.Close()
			return nil, err
		}
		if n == 0 {
			break
		}
		// write a chunk
		if _, err := file.Write(buf[:n]); err != nil {
			file.Close()
			return nil, err
		}
	}
	file.Close()

	return n.sealingApi.AddPieceOnDealComplete(ctx, piecePath, sdInfo)
}

func (n *ProviderNodeAdapterDealer) VerifySignature(ctx context.Context, sig crypto.Signature, addr address.Address, input []byte, encodedTs shared.TipSetToken) (bool, error) {
	addr, err := n.StateAccountKey(ctx, addr, types.EmptyTSK)
	if err != nil {
		return false, err
	}

	err = sigs.Verify(&sig, addr, input)
	return err == nil, err
}

func (n *ProviderNodeAdapterDealer) GetMinerWorkerAddress(ctx context.Context, miner address.Address, tok shared.TipSetToken) (address.Address, error) {
	tsk, err := types.TipSetKeyFromBytes(tok)
	if err != nil {
		return address.Undef, err
	}

	mi, err := n.StateMinerInfo(ctx, miner, tsk)
	if err != nil {
		return address.Address{}, err
	}
	return mi.Worker, nil
}

func (n *ProviderNodeAdapterDealer) SignBytes(ctx context.Context, signer address.Address, b []byte) (*crypto.Signature, error) {
	signer, err := n.StateAccountKey(ctx, signer, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	localSignature, err := n.WalletSign(ctx, signer, b)
	if err != nil {
		return nil, err
	}
	return localSignature, nil
}

func (n *ProviderNodeAdapterDealer) ReserveFunds(ctx context.Context, wallet, addr address.Address, amt abi.TokenAmount) (cid.Cid, error) {
	return n.MarketReserveFunds(ctx, wallet, addr, amt)
}

func (n *ProviderNodeAdapterDealer) ReleaseFunds(ctx context.Context, addr address.Address, amt abi.TokenAmount) error {
	return n.MarketReleaseFunds(ctx, addr, amt)
}

// Adds funds with the StorageMinerActor for a storage participant.  Used by both providers and clients.
func (n *ProviderNodeAdapterDealer) AddFunds(ctx context.Context, addr address.Address, amount abi.TokenAmount) (cid.Cid, error) {
	// (Provider Node API)
	smsg, err := n.MpoolPushMessage(ctx, &types.Message{
		To:     market.Address,
		From:   addr,
		Value:  amount,
		Method: market.Methods.AddBalance,
	}, n.addBalanceSpec)
	if err != nil {
		return cid.Undef, err
	}

	return smsg.Cid(), nil
}

func (n *ProviderNodeAdapterDealer) GetBalance(ctx context.Context, addr address.Address, encodedTs shared.TipSetToken) (storagemarket.Balance, error) {
	tsk, err := types.TipSetKeyFromBytes(encodedTs)
	if err != nil {
		return storagemarket.Balance{}, err
	}

	bal, err := n.StateMarketBalance(ctx, addr, tsk)
	if err != nil {
		return storagemarket.Balance{}, err
	}

	return utils.ToSharedBalance(bal), nil
}

// TODO: why doesnt this method take in a sector ID?
func (n *ProviderNodeAdapterDealer) LocatePieceForDealWithinSector(ctx context.Context, dealID abi.DealID, encodedTs shared.TipSetToken) (sectorID abi.SectorNumber, offset abi.PaddedPieceSize, length abi.PaddedPieceSize, err error) {
	lp, err := n.sealingApi.LocatePieceForDealWithinSector(ctx, dealID, encodedTs)
	if err != nil {
		return 0, 0, 0, err
	}
	return lp.SectorID, lp.Offset, lp.Length, nil
}

func (n *ProviderNodeAdapterDealer) DealProviderCollateralBounds(ctx context.Context, size abi.PaddedPieceSize, isVerified bool) (abi.TokenAmount, abi.TokenAmount, error) {
	bounds, err := n.StateDealProviderCollateralBounds(ctx, size, isVerified, types.EmptyTSK)
	if err != nil {
		return abi.TokenAmount{}, abi.TokenAmount{}, err
	}

	return bounds.Min, bounds.Max, nil
}

func (n *ProviderNodeAdapterDealer) OnDealSectorCommitted(ctx context.Context, provider address.Address, dealID abi.DealID, proposal market2.DealProposal, publishCid *cid.Cid, cb storagemarket.DealSectorCommittedCallback) error {
	return OnDealSectorCommitted(ctx, n, n.ev, provider, dealID, market.DealProposal(proposal), publishCid, cb)
}

func (n *ProviderNodeAdapterDealer) GetChainHead(ctx context.Context) (shared.TipSetToken, abi.ChainEpoch, error) {
	head, err := n.ChainHead(ctx)
	if err != nil {
		return nil, 0, err
	}

	return head.Key().Bytes(), head.Height(), nil
}

func (n *ProviderNodeAdapterDealer) WaitForMessage(ctx context.Context, mcid cid.Cid, cb func(code exitcode.ExitCode, bytes []byte, finalCid cid.Cid, err error) error) error {
	receipt, err := n.StateWaitMsg(ctx, mcid, 2*build.MessageConfidence)
	if err != nil {
		return cb(0, nil, cid.Undef, err)
	}
	return cb(receipt.Receipt.ExitCode, receipt.Receipt.Return, receipt.Message, nil)
}

func (n *ProviderNodeAdapterDealer) GetDataCap(ctx context.Context, addr address.Address, encodedTs shared.TipSetToken) (*abi.StoragePower, error) {
	tsk, err := types.TipSetKeyFromBytes(encodedTs)
	if err != nil {
		return nil, err
	}

	sp, err := n.StateVerifiedClientStatus(ctx, addr, tsk)
	return sp, err
}

func (n *ProviderNodeAdapterDealer) OnDealExpiredOrSlashed(ctx context.Context, dealID abi.DealID, onDealExpired storagemarket.DealExpiredCallback, onDealSlashed storagemarket.DealSlashedCallback) error {
	head, err := n.ChainHead(ctx)
	if err != nil {
		return xerrors.Errorf("client: failed to get chain head: %w", err)
	}

	sd, err := n.StateMarketStorageDeal(ctx, dealID, head.Key())
	if err != nil {
		return xerrors.Errorf("client: failed to look up deal %d on chain: %w", dealID, err)
	}

	// Called immediately to check if the deal has already expired or been slashed
	checkFunc := func(ts *types.TipSet) (done bool, more bool, err error) {
		if ts == nil {
			// keep listening for events
			return false, true, nil
		}

		// Check if the deal has already expired
		if sd.Proposal.EndEpoch <= ts.Height() {
			onDealExpired(nil)
			return true, false, nil
		}

		// If there is no deal assume it's already been slashed
		if sd.State.SectorStartEpoch < 0 {
			onDealSlashed(ts.Height(), nil)
			return true, false, nil
		}

		// No events have occurred yet, so return
		// done: false, more: true (keep listening for events)
		return false, true, nil
	}

	// Called when there was a match against the state change we're looking for
	// and the chain has advanced to the confidence height
	stateChanged := func(ts *types.TipSet, ts2 *types.TipSet, states events.StateChange, h abi.ChainEpoch) (more bool, err error) {
		// Check if the deal has already expired
		if sd.Proposal.EndEpoch <= ts2.Height() {
			onDealExpired(nil)
			return false, nil
		}

		// Timeout waiting for state change
		if states == nil {
			log.Error("timed out waiting for deal expiry")
			return false, nil
		}

		changedDeals, ok := states.(state.ChangedDeals)
		if !ok {
			panic("Expected state.ChangedDeals")
		}

		deal, ok := changedDeals[dealID]
		if !ok {
			// No change to deal
			return true, nil
		}

		// Deal was slashed
		if deal.To == nil {
			onDealSlashed(ts2.Height(), nil)
			return false, nil
		}

		return true, nil
	}

	// Called when there was a chain reorg and the state change was reverted
	revert := func(ctx context.Context, ts *types.TipSet) error {
		// TODO: Is it ok to just ignore this?
		log.Warn("deal state reverted; TODO: actually handle this!")
		return nil
	}

	// Watch for state changes to the deal
	match := n.dsMatcher.matcher(ctx, dealID)

	// Wait until after the end epoch for the deal and then timeout
	timeout := (sd.Proposal.EndEpoch - head.Height()) + 1
	if err := n.ev.StateChanged(checkFunc, stateChanged, revert, int(build.MessageConfidence)+1, timeout, match); err != nil {
		return xerrors.Errorf("failed to set up state changed handler: %w", err)
	}

	return nil
}

var _ storagemarket.StorageProviderNode = &ProviderNodeAdapterDealer{}
