package main

import (
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/xerrors"
	"gopkg.in/urfave/cli.v2"

	"github.com/filecoin-project/go-lotus/build"
	"github.com/filecoin-project/go-lotus/chain/actors"
	"github.com/filecoin-project/go-lotus/chain/address"
	"github.com/filecoin-project/go-lotus/chain/types"
	lcli "github.com/filecoin-project/go-lotus/cli"
	"github.com/filecoin-project/go-lotus/node/repo"
)

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize a lotus storage miner repo",
	Action: func(cctx *cli.Context) error {
		log.Info("Initializing lotus storage miner")
		log.Info("Checking if repo exists")

		r, err := repo.NewFS(cctx.String(FlagStorageRepo))
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if ok {
			return xerrors.Errorf("repo at '%s' is already initialized", cctx.String(FlagStorageRepo))
		}

		log.Info("Trying to connect to full node RPC")

		api, err := lcli.GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		ctx := lcli.ReqContext(cctx)

		log.Info("Checking full node version")

		v, err := api.Version(ctx)
		if err != nil {
			return err
		}

		if v.APIVersion&build.MinorMask != build.APIVersion&build.MinorMask {
			return xerrors.Errorf("Remote API version didn't match (local %x, remote %x)", build.APIVersion, v.APIVersion)
		}

		log.Info("Initializing repo")

		if err := r.Init(); err != nil {
			return err
		}

		lr, err := r.Lock()
		if err != nil {
			return err
		}
		defer lr.Close()

		log.Info("Initializing libp2p identity")

		p2pSk, err := lr.Libp2pIdentity()
		if err != nil {
			return err
		}

		peerid, err := peer.IDFromPrivateKey(p2pSk)
		if err != nil {
			return err
		}

		log.Info("Creating StorageMarket.CreateStorageMiner message")

		defOwner, err := api.WalletDefaultAddress(ctx)
		if err != nil {
			return err
		}

		nonce, err := api.MpoolGetNonce(ctx, defOwner)
		if err != nil {
			return err
		}

		k, err := api.WalletNew(ctx, types.KTSecp256k1)
		if err != nil {
			return err
		}

		collateral := types.NewInt(1000) // TODO: Get this from params

		params, err := actors.SerializeParams(actors.CreateStorageMinerParams{
			Owner:      defOwner,
			Worker:     k,
			SectorSize: types.NewInt(actors.SectorSize),
			PeerID:     peerid,
		})
		if err != nil {
			return err
		}

		createStorageMinerMsg := types.Message{
			To:   actors.StorageMarketAddress,
			From: defOwner,

			Nonce: nonce,

			Value: collateral,

			Method: actors.SMAMethods.CreateStorageMiner,
			Params: params,
		}

		unsigned, err := createStorageMinerMsg.Serialize()
		if err != nil {
			return err
		}

		log.Info("Signing StorageMarket.CreateStorageMiner")

		sig, err := api.WalletSign(ctx, defOwner, unsigned)
		if err != nil {
			return err
		}

		signed := &types.SignedMessage{
			Message:   createStorageMinerMsg,
			Signature: *sig,
		}

		log.Infof("Pushing %s to Mpool", signed.Cid())

		err = api.MpoolPush(ctx, signed)
		if err != nil {
			return err
		}

		log.Infof("Waiting for confirmation")

		mw, err := api.ChainWaitMsg(ctx, signed.Cid())
		if err != nil {
			return err
		}

		addr, err := address.NewFromBytes(mw.Receipt.Return)
		if err != nil {
			return err
		}

		log.Infof("New storage miners address is: %s", addr)

		ds, err := lr.Datastore("/metadata")
		if err != nil {
			return err
		}
		if err := ds.Put(datastore.NewKey("miner-address"), addr.Bytes()); err != nil {
			return err
		}

		// TODO: Point to setting storage price, maybe do it interactively or something
		log.Info("Storage miner successfully created, you can now start it with 'lotus-storage-miner run'")

		return nil
	},
}
