package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/filecoin-project/lotus/node/modules/lp2p"
	"github.com/libp2p/go-libp2p-core/crypto"

	"github.com/filecoin-project/go-state-types/big"

	"github.com/docker/go-units"
	"github.com/google/uuid"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	paramfetch "github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"

	market2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	power2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"

	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/actors/builtin/power"
	"github.com/filecoin-project/lotus/chain/actors/policy"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/filecoin-project/lotus/genesis"
	"github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
)

var initCmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize a lotus dealer repo",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "actor",
			Usage: "specify the address of an already created miner actor",
		},
		&cli.BoolFlag{
			Name:   "genesis-miner",
			Usage:  "enable genesis mining (DON'T USE ON BOOTSTRAPPED NETWORK)",
			Hidden: true,
		},
		&cli.BoolFlag{
			Name:  "create-worker-key",
			Usage: "create separate worker key",
		},
		&cli.StringFlag{
			Name:    "worker",
			Aliases: []string{"w"},
			Usage:   "worker key to use (overrides --create-worker-key)",
		},
		&cli.StringFlag{
			Name:    "owner",
			Aliases: []string{"o"},
			Usage:   "owner key to use",
		},
		&cli.StringFlag{
			Name:  "sector-size",
			Usage: "specify sector size to use",
			Value: units.BytesSize(float64(policy.GetDefaultSectorSize())),
		},
		&cli.StringSliceFlag{
			Name:  "pre-sealed-sectors",
			Usage: "specify set of presealed sectors for starting as a genesis miner",
		},
		&cli.StringFlag{
			Name:  "pre-sealed-metadata",
			Usage: "specify the metadata file for the presealed sectors",
		},
		&cli.BoolFlag{
			Name:  "nosync",
			Usage: "don't check full-node sync status",
		},
		&cli.BoolFlag{
			Name:  "symlink-imported-sectors",
			Usage: "attempt to symlink to presealed sectors instead of copying them into place",
		},
		&cli.BoolFlag{
			Name:  "no-local-storage",
			Usage: "don't use storageminer repo for sector storage",
		},
		&cli.StringFlag{
			Name:  "gas-premium",
			Usage: "set gas premium for initialization messages in AttoFIL",
			Value: "0",
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "select which address to send actor creation message from",
		},
		&cli.Uint64Flag{
			Name:  "sector-start",
			Usage: "sector with start",
			Value: 0,
		},
		&cli.BoolFlag{
			Name:  "clear-sector",
			Usage: "clear pre sector",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Initializing lotus dealer")

		sectorSizeInt, err := units.RAMInBytes(cctx.String("sector-size"))
		if err != nil {
			return err
		}
		ssize := abi.SectorSize(sectorSizeInt)

		gasPrice, err := types.BigFromString(cctx.String("gas-premium"))
		if err != nil {
			return xerrors.Errorf("failed to parse gas-price flag: %s", err)
		}

		symlink := cctx.Bool("symlink-imported-sectors")
		if symlink {
			log.Info("will attempt to symlink to imported sectors")
		}

		ctx := lcli.ReqContext(cctx)

		log.Info("Checking proof parameters")

		if err := paramfetch.GetParams(ctx, build.ParametersJSON(), uint64(ssize)); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}

		log.Info("Trying to connect to full node RPC")

		api, closer, err := lcli.GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
		if err != nil {
			return err
		}
		defer closer()

		log.Info("Checking full node sync status")

		if !cctx.Bool("genesis-miner") && !cctx.Bool("nosync") {
			if err := lcli.SyncWait(ctx, api, false); err != nil {
				return xerrors.Errorf("sync wait: %w", err)
			}
		}

		log.Info("Checking if repo exists")

		repoPath := cctx.String(FlagDealerRepo)
		r, err := repo.NewFS(repoPath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if ok {
			return xerrors.Errorf("repo at '%s' is already initialized", cctx.String(FlagDealerRepo))
		}

		log.Info("Checking full node version")

		v, err := api.Version(ctx)
		if err != nil {
			return err
		}

		if !v.APIVersion.EqMajorMinor(build.FullAPIVersion) {
			return xerrors.Errorf("Remote API version didn't match (expected %s, remote %s)", build.FullAPIVersion, v.APIVersion)
		}

		log.Info("Initializing repo")

		if err := r.Init(repo.StorageDealer); err != nil {
			return err
		}

		{
			lr, err := r.Lock(repo.StorageDealer)
			if err != nil {
				return err
			}

			var localPaths []stores.LocalPath

			if pssb := cctx.StringSlice("pre-sealed-sectors"); len(pssb) != 0 {
				log.Infof("Setting up storage config with presealed sectors: %v", pssb)

				for _, psp := range pssb {
					psp, err := homedir.Expand(psp)
					if err != nil {
						return err
					}
					localPaths = append(localPaths, stores.LocalPath{
						Path: psp,
					})
				}
			}

			if !cctx.Bool("no-local-storage") {
				b, err := json.MarshalIndent(&stores.LocalStorageMeta{
					ID:       stores.ID(uuid.New().String()),
					Weight:   10,
					CanSeal:  true,
					CanStore: true,
				}, "", "  ")
				if err != nil {
					return xerrors.Errorf("marshaling storage config: %w", err)
				}

				if err := ioutil.WriteFile(filepath.Join(lr.Path(), "sectorstore.json"), b, 0644); err != nil {
					return xerrors.Errorf("persisting storage metadata (%s): %w", filepath.Join(lr.Path(), "sectorstore.json"), err)
				}

				localPaths = append(localPaths, stores.LocalPath{
					Path: lr.Path(),
				})
			}

			if err := lr.SetStorage(func(sc *stores.StorageConfig) {
				sc.StoragePaths = append(sc.StoragePaths, localPaths...)
			}); err != nil {
				return xerrors.Errorf("set storage config: %w", err)
			}

			if err := lr.Close(); err != nil {
				return err
			}
		}

		if err := storageMinerInit(ctx, cctx, api, r, ssize, gasPrice); err != nil {
			log.Errorf("Failed to initialize lotus-dealer: %+v", err)
			path, err := homedir.Expand(repoPath)
			if err != nil {
				return err
			}
			log.Infof("Cleaning up %s after attempt...", path)
			if err := os.RemoveAll(path); err != nil {
				log.Errorf("Failed to clean up failed storage repo: %s", err)
			}
			return xerrors.Errorf("Storage-dealer init failed")
		}

		// TODO: Point to setting storage price, maybe do it interactively or something
		log.Info("Dealer successfully created, you can now start it with 'lotus-dealer run'")

		return nil
	},
}

func migratePreSealMeta(ctx context.Context, api lapi.FullNode, metadata string, maddr address.Address, mds, mfds dtypes.MetadataDS) error {
	metadata, err := homedir.Expand(metadata)
	if err != nil {
		return xerrors.Errorf("expanding preseal dir: %w", err)
	}

	b, err := ioutil.ReadFile(metadata)
	if err != nil {
		return xerrors.Errorf("reading preseal metadata: %w", err)
	}

	psm := map[string]genesis.Miner{}
	if err := json.Unmarshal(b, &psm); err != nil {
		return xerrors.Errorf("unmarshaling preseal metadata: %w", err)
	}

	meta, ok := psm[maddr.String()]
	if !ok {
		return xerrors.Errorf("preseal file didn't contain metadata for dealer %s", maddr)
	}

	maxSectorID := abi.SectorNumber(0)
	for _, sector := range meta.Sectors {
		sectorKey := datastore.NewKey(sealing.SectorStorePrefix).ChildString(fmt.Sprint(sector.SectorID))

		dealID, err := findMarketDealID(ctx, api, sector.Deal)
		if err != nil {
			return xerrors.Errorf("finding storage deal for pre-sealed sector %d: %w", sector.SectorID, err)
		}
		commD := sector.CommD
		commR := sector.CommR

		info := &sealing.SectorInfo{
			State:        sealing.Proving,
			SectorNumber: sector.SectorID,
			Pieces: []sealing.Piece{
				{
					Piece: abi.PieceInfo{
						Size:     abi.PaddedPieceSize(meta.SectorSize),
						PieceCID: commD,
					},
					DealInfo: &sealing.DealInfo{
						DealID: dealID,
						DealSchedule: sealing.DealSchedule{
							StartEpoch: sector.Deal.StartEpoch,
							EndEpoch:   sector.Deal.EndEpoch,
						},
					},
				},
			},
			CommD:            &commD,
			CommR:            &commR,
			Proof:            nil,
			TicketValue:      abi.SealRandomness{},
			TicketEpoch:      0,
			PreCommitMessage: nil,
			SeedValue:        abi.InteractiveSealRandomness{},
			SeedEpoch:        0,
			CommitMessage:    nil,
		}

		b, err := cborutil.Dump(info)
		if err != nil {
			return err
		}

		if err := mds.Put(sectorKey, b); err != nil {
			return err
		}

		if sector.SectorID > maxSectorID {
			maxSectorID = sector.SectorID
		}

		/* // TODO: Import deals into market
		pnd, err := cborutil.AsIpld(sector.Deal)
		if err != nil {
			return err
		}

		dealKey := datastore.NewKey(deals.ProviderDsPrefix).ChildString(pnd.Cid().String())

		deal := &deals.MinerDeal{
			MinerDeal: storagemarket.MinerDeal{
				ClientDealProposal: sector.Deal,
				ProposalCid: pnd.Cid(),
				State:       storagemarket.StorageDealActive,
				Ref:         &storagemarket.DataRef{Root: proposalCid}, // TODO: This is super wrong, but there
				// are no params for CommP CIDs, we can't recover unixfs cid easily,
				// and this isn't even used after the deal enters Complete state
				DealID: dealID,
			},
		}

		b, err = cborutil.Dump(deal)
		if err != nil {
			return err
		}

		if err := mds.Put(dealKey, b); err != nil {
			return err
		}*/
	}

	buf := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(buf, uint64(maxSectorID))
	return mfds.Put(datastore.NewKey(modules.StorageCounterDSPrefix), buf[:size])
}

func findMarketDealID(ctx context.Context, api lapi.FullNode, deal market2.DealProposal) (abi.DealID, error) {
	// TODO: find a better way
	//  (this is only used by genesis miners)

	deals, err := api.StateMarketDeals(ctx, types.EmptyTSK)
	if err != nil {
		return 0, xerrors.Errorf("getting market deals: %w", err)
	}

	for k, v := range deals {
		if v.Proposal.PieceCID.Equals(deal.PieceCID) {
			id, err := strconv.ParseUint(k, 10, 64)
			return abi.DealID(id), err
		}
	}

	return 0, xerrors.New("deal not found")
}

func storageMinerInit(ctx context.Context, cctx *cli.Context, api lapi.FullNode, r repo.Repo, ssize abi.SectorSize, gasPrice types.BigInt) error {
	act := cctx.String("actor")
	lr, err := r.Lock(repo.StorageSealer)
	if err != nil {
		return err
	}
	defer lr.Close() //nolint:errcheck

	mfds, err := lr.Datastore("/metadata")
	if err != nil {
		return err
	}
	//check if init already
	exist, err := mfds.Has(datastore.NewKey("miner-address"))
	if err != nil {
		return err
	}
	if exist {
		return xerrors.Errorf("miner-address exist")
	}

	var mds dtypes.MetadataDS
	var ks types.KeyStore

	if cctx.String(FlagPostgresURL) == "" {
		//使用本地文件
		mds = mfds
		ks, err = lr.KeyStore()
		if err != nil {
			return err
		}

		log.Info("Initializing libp2p identity")
		p2pSk, err := lp2p.PrivKey(ks)
		if err != nil {
			return xerrors.Errorf("make host key: %w", err)
		}

		peerid, err := peer.IDFromPrivateKey(p2pSk)
		if err != nil {
			return xerrors.Errorf("peer ID from private key: %w", err)
		}
		var addr address.Address
		if act != "" {
			a, err := address.NewFromString(act)
			if err != nil {
				return xerrors.Errorf("failed parsing actor flag value (%q): %w", act, err)
			}

			if pssb := cctx.String("pre-sealed-metadata"); pssb != "" {
				pssb, err := homedir.Expand(pssb)
				if err != nil {
					return err
				}

				log.Infof("Importing pre-sealed sector metadata for %s", a)

				if err := migratePreSealMeta(ctx, api, pssb, a, mds, mfds); err != nil {
					return xerrors.Errorf("migrating presealed sector metadata: %w", err)
				}
			}

			//不对peerId进行任何操作，请从sealer目录中拷贝keystore文件。

			addr = a
		} else {
			a, err := createStorageMiner(ctx, api, peerid, gasPrice, cctx)
			if err != nil {
				return xerrors.Errorf("creating dealer failed: %w", err)
			}

			addr = a
		}

		if err := mds.Put(datastore.NewKey("miner-address"), addr.Bytes()); err != nil {
			return err
		}
	} else {
		//使用数据库
		db, err := sql.Open("postgres", cctx.String(FlagPostgresURL))
		if err != nil {
			return err
		}
		defer db.Close()
		if act != "" { //如果使用了 act 那么除非数据库没有 peerID,否则不需要链上确认
			a, err := address.NewFromString(act)
			if err != nil {
				return xerrors.Errorf("failed parsing actor flag value (%q): %w", act, err)
			}

			mds, err = modules.DataBase(db, act)
			if err != nil {
				return err
			}
			ks, err = repo.NewDBKeyStore(db, act)
			if err != nil {
				return err
			}

			if pssb := cctx.String("pre-sealed-metadata"); pssb != "" {
				pssb, err := homedir.Expand(pssb)
				if err != nil {
					return err
				}

				log.Infof("Importing pre-sealed sector metadata for %s", a)

				if err := migratePreSealMeta(ctx, api, pssb, a, mds, mfds); err != nil {
					return xerrors.Errorf("migrating presealed sector metadata: %w", err)
				}
			}

			//获取peerID,如果有peerID就直接返回，没有的话就发送消息到链上
			if _, err = ks.Get(lp2p.KLibp2pHost); err != nil {
				if err != types.ErrKeyInfoNotFound {
					return err
				}
				//没有则生成一个新的 peerId
				log.Info("Initializing libp2p identity")
				p2pSk, _, err := crypto.GenerateEd25519Key(rand.Reader)
				if err != nil {
					return err
				}
				//peerid, err := peer.IDFromPrivateKey(p2pSk)
				//if err != nil {
				//	return xerrors.Errorf("peer ID from private key: %w", err)
				//}
				////通知链上更改peerId
				//if err := configureStorageMiner(ctx, api, a, peerid, gasPrice); err != nil {
				//	return xerrors.Errorf("failed to configure miner: %w", err)
				//}
				//保存新的 peerId
				kbytes, err := p2pSk.Bytes()
				if err != nil {
					return err
				}
				if err := ks.Put(lp2p.KLibp2pHost, types.KeyInfo{
					Type:       lp2p.KTLibp2pHost,
					PrivateKey: kbytes,
				}); err != nil {
					return err
				}
				if err := mds.Put(datastore.NewKey("miner-address"), a.Bytes()); err != nil {
					return err
				}
			}
		} else {
			//创建一个新actor
			//生成一个peerId
			log.Info("Initializing libp2p identity")
			p2pSk, _, err := crypto.GenerateEd25519Key(rand.Reader)
			if err != nil {
				return err
			}
			peerid, err := peer.IDFromPrivateKey(p2pSk)
			if err != nil {
				return xerrors.Errorf("peer ID from private key: %w", err)
			}
			//创建一个新的minerid
			addr, err := createStorageMiner(ctx, api, peerid, gasPrice, cctx)
			if err != nil {
				return xerrors.Errorf("creating miner failed: %w", err)
			}
			//保存peerId
			mds, err = modules.DataBase(db, addr.String())
			if err != nil {
				return err
			}
			ks, err = repo.NewDBKeyStore(db, addr.String())
			if err != nil {
				return err
			}
			kbytes, err := p2pSk.Bytes()
			if err != nil {
				return err
			}
			if err := ks.Put(lp2p.KLibp2pHost, types.KeyInfo{
				Type:       lp2p.KTLibp2pHost,
				PrivateKey: kbytes,
			}); err != nil {
				return err
			}
			if err := mds.Put(datastore.NewKey("miner-address"), addr.Bytes()); err != nil {
				return err
			}
			log.Infof("Created new miner: %s", addr)
		}
	}

	minSectorID := cctx.Uint64("sector-start")

	log.Infof("=========> set sector start : %d", minSectorID)
	buf := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(buf, minSectorID)
	if err := mfds.Put(datastore.NewKey(modules.StorageSectorStart), buf[:size]); err != nil {
		return err
	}
	key := datastore.NewKey(modules.StorageCounterDSPrefix)
	has, err := mfds.Has(key)
	if err != nil {
		return err
	}
	var cur uint64 = 0
	if has {
		curBytes, err := mfds.Get(key)
		if err != nil {
			return err
		}
		cur, _ = binary.Uvarint(curBytes)
	}
	if minSectorID > cur {
		log.Infof("=========> set sector number : %d", minSectorID)
		buf := make([]byte, binary.MaxVarintLen64)
		size := binary.PutUvarint(buf, minSectorID)
		if err := mfds.Put(datastore.NewKey(modules.StorageCounterDSPrefix), buf[:size]); err != nil {
			return err
		}
	}
	return nil
}

func configureStorageMiner(ctx context.Context, api lapi.FullNode, addr address.Address, peerid peer.ID, gasPrice types.BigInt) error {
	mi, err := api.StateMinerInfo(ctx, addr, types.EmptyTSK)
	if err != nil {
		return xerrors.Errorf("getWorkerAddr returned bad address: %w", err)
	}

	enc, err := actors.SerializeParams(&miner2.ChangePeerIDParams{NewID: abi.PeerID(peerid)})
	if err != nil {
		return err
	}

	msg := &types.Message{
		To:         addr,
		From:       mi.Worker,
		Method:     miner.Methods.ChangePeerID,
		Params:     enc,
		Value:      types.NewInt(0),
		GasPremium: gasPrice,
	}

	smsg, err := api.MpoolPushMessage(ctx, msg, nil)
	if err != nil {
		return err
	}

	log.Info("Waiting for message: ", smsg.Cid())
	ret, err := api.StateWaitMsg(ctx, smsg.Cid(), build.MessageConfidence)
	if err != nil {
		return err
	}

	if ret.Receipt.ExitCode != 0 {
		return xerrors.Errorf("update peer id message failed with exit code %d", ret.Receipt.ExitCode)
	}

	return nil
}

func createStorageMiner(ctx context.Context, api lapi.FullNode, peerid peer.ID, gasPrice types.BigInt, cctx *cli.Context) (address.Address, error) {
	log.Info("Creating StorageMarket.CreateStorageMiner message")

	var err error
	var owner address.Address
	if cctx.String("owner") != "" {
		owner, err = address.NewFromString(cctx.String("owner"))
	} else {
		owner, err = api.WalletDefaultAddress(ctx)
	}
	if err != nil {
		return address.Undef, err
	}

	ssize, err := units.RAMInBytes(cctx.String("sector-size"))
	if err != nil {
		return address.Undef, fmt.Errorf("failed to parse sector size: %w", err)
	}

	worker := owner
	if cctx.String("worker") != "" {
		worker, err = address.NewFromString(cctx.String("worker"))
	} else if cctx.Bool("create-worker-key") { // TODO: Do we need to force this if owner is Secpk?
		worker, err = api.WalletNew(ctx, types.KTBLS)
	}
	if err != nil {
		return address.Address{}, err
	}

	// make sure the worker account exists on chain
	_, err = api.StateLookupID(ctx, worker, types.EmptyTSK)
	if err != nil {
		signed, err := api.MpoolPushMessage(ctx, &types.Message{
			From:  owner,
			To:    worker,
			Value: types.NewInt(0),
		}, nil)
		if err != nil {
			return address.Undef, xerrors.Errorf("push worker init: %w", err)
		}

		log.Infof("Initializing worker account %s, message: %s", worker, signed.Cid())
		log.Infof("Waiting for confirmation")

		mw, err := api.StateWaitMsg(ctx, signed.Cid(), build.MessageConfidence)
		if err != nil {
			return address.Undef, xerrors.Errorf("waiting for worker init: %w", err)
		}
		if mw.Receipt.ExitCode != 0 {
			return address.Undef, xerrors.Errorf("initializing worker account failed: exit code %d", mw.Receipt.ExitCode)
		}
	}

	spt, err := ffiwrapper.SealProofTypeFromSectorSize(abi.SectorSize(ssize))
	if err != nil {
		return address.Undef, err
	}

	params, err := actors.SerializeParams(&power2.CreateMinerParams{
		Owner:         owner,
		Worker:        worker,
		SealProofType: spt,
		Peer:          abi.PeerID(peerid),
	})
	if err != nil {
		return address.Undef, err
	}

	sender := owner
	if fromstr := cctx.String("from"); fromstr != "" {
		faddr, err := address.NewFromString(fromstr)
		if err != nil {
			return address.Undef, fmt.Errorf("could not parse from address: %w", err)
		}
		sender = faddr
	}

	createStorageMinerMsg := &types.Message{
		To:    power.Address,
		From:  sender,
		Value: big.Zero(),

		Method: power.Methods.CreateMiner,
		Params: params,

		GasLimit:   0,
		GasPremium: gasPrice,
	}

	signed, err := api.MpoolPushMessage(ctx, createStorageMinerMsg, nil)
	if err != nil {
		return address.Undef, xerrors.Errorf("pushing createMiner message: %w", err)
	}

	log.Infof("Pushed CreateMiner message: %s", signed.Cid())
	log.Infof("Waiting for confirmation")

	mw, err := api.StateWaitMsg(ctx, signed.Cid(), build.MessageConfidence)
	if err != nil {
		return address.Undef, xerrors.Errorf("waiting for createMiner message: %w", err)
	}

	if mw.Receipt.ExitCode != 0 {
		return address.Undef, xerrors.Errorf("create miner failed: exit code %d", mw.Receipt.ExitCode)
	}

	var retval power2.CreateMinerReturn
	if err := retval.UnmarshalCBOR(bytes.NewReader(mw.Receipt.Return)); err != nil {
		return address.Undef, err
	}

	log.Infof("New miners address is: %s (%s)", retval.IDAddress, retval.RobustAddress)
	return retval.IDAddress, nil
}
