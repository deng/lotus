package node_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"io/ioutil"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	sectorbuilder "github.com/filecoin-project/go-sectorbuilder"

	"github.com/filecoin-project/lotus/build"
	genesis "github.com/filecoin-project/lotus/genesis"

	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	saminer "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	badger "github.com/ipfs/go-ds-badger2"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/test"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/cmd/lotus-seed/seed"
	"github.com/filecoin-project/lotus/lib/jsonrpc"
	"github.com/filecoin-project/lotus/miner"
	"github.com/filecoin-project/lotus/node"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/lotus/node/modules"
	modtest "github.com/filecoin-project/lotus/node/modules/testing"
	"github.com/filecoin-project/lotus/node/repo"
	"github.com/filecoin-project/lotus/storage/sbmock"
)

func init() {
	_ = logging.SetLogLevel("*", "INFO")

	build.SectorSizes = []abi.SectorSize{1024}
	build.MinimumMinerPower = 1024
}

func testStorageNode(ctx context.Context, t *testing.T, waddr address.Address, act address.Address, pk crypto.PrivKey, tnd test.TestNode, mn mocknet.Mocknet, opts node.Option) test.TestStorageNode {
	r := repo.NewMemory(nil)

	lr, err := r.Lock(repo.StorageMiner)
	require.NoError(t, err)

	ks, err := lr.KeyStore()
	require.NoError(t, err)

	kbytes, err := pk.Bytes()
	require.NoError(t, err)

	err = ks.Put("libp2p-host", types.KeyInfo{
		Type:       "libp2p-host",
		PrivateKey: kbytes,
	})
	require.NoError(t, err)

	ds, err := lr.Datastore("/metadata")
	require.NoError(t, err)
	err = ds.Put(datastore.NewKey("miner-address"), act.Bytes())
	require.NoError(t, err)

	err = lr.Close()
	require.NoError(t, err)

	peerid, err := peer.IDFromPrivateKey(pk)
	require.NoError(t, err)

	enc, err := actors.SerializeParams(&saminer.ChangePeerIDParams{NewID: peerid})
	require.NoError(t, err)

	msg := &types.Message{
		To:       act,
		From:     waddr,
		Method:   builtin.MethodsMiner.ChangePeerID,
		Params:   enc,
		Value:    types.NewInt(0),
		GasPrice: types.NewInt(0),
		GasLimit: types.NewInt(1000000),
	}

	_, err = tnd.MpoolPushMessage(ctx, msg)
	require.NoError(t, err)

	// start node
	var minerapi api.StorageMiner

	mineBlock := make(chan struct{})
	// TODO: use stop
	_, err = node.New(ctx,
		node.StorageMiner(&minerapi),
		node.Online(),
		node.Repo(r),
		node.Test(),

		node.MockHost(mn),

		node.Override(new(api.FullNode), tnd),
		node.Override(new(*miner.Miner), miner.NewTestMiner(mineBlock, act)),

		opts,
	)
	if err != nil {
		t.Fatalf("failed to construct node: %v", err)
	}

	/*// Bootstrap with full node
	remoteAddrs, err := tnd.NetAddrsListen(ctx)
	require.NoError(t, err)

	err = minerapi.NetConnect(ctx, remoteAddrs)
	require.NoError(t, err)*/
	mineOne := func(ctx context.Context) error {
		select {
		case mineBlock <- struct{}{}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return test.TestStorageNode{StorageMiner: minerapi, MineOne: mineOne}
}

func builder(t *testing.T, nFull int, storage []int) ([]test.TestNode, []test.TestStorageNode) {
	ctx := context.Background()
	mn := mocknet.New(ctx)

	fulls := make([]test.TestNode, nFull)
	storers := make([]test.TestStorageNode, len(storage))

	pk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	minerPid, err := peer.IDFromPrivateKey(pk)
	require.NoError(t, err)

	_ = minerPid // TODO: what do we do with this now?

	var genbuf bytes.Buffer

	if len(storage) > 1 {
		panic("need more peer IDs")
	}
	// PRESEAL SECTION, TRY TO REPLACE WITH BETTER IN THE FUTURE
	// TODO: would be great if there was a better way to fake the preseals

	var genms []genesis.Miner
	var maddrs []address.Address

	var presealDirs []string
	for i := 0; i < len(storage); i++ {
		maddr, err := address.NewIDAddress(300 + uint64(i))
		if err != nil {
			t.Fatal(err)
		}
		tdir, err := ioutil.TempDir("", "preseal-memgen")
		if err != nil {
			t.Fatal(err)
		}
		genm, _, err := seed.PreSeal(maddr, 1024, 0, 1, tdir, []byte("make genesis mem random"), nil)
		if err != nil {
			t.Fatal(err)
		}

		presealDirs = append(presealDirs, tdir)
		maddrs = append(maddrs, maddr)
		genms = append(genms, *genm)
	}

	templ := &genesis.Template{
		Miners: genms,
	}

	// END PRESEAL SECTION

	for i := 0; i < nFull; i++ {
		var genesis node.Option
		if i == 0 {
			genesis = node.Override(new(modules.Genesis), modtest.MakeGenesisMem(&genbuf, *templ))
		} else {
			genesis = node.Override(new(modules.Genesis), modules.LoadGenesis(genbuf.Bytes()))
		}

		var err error
		// TODO: Don't ignore stop
		_, err = node.New(ctx,
			node.FullAPI(&fulls[i].FullNode),
			node.Online(),
			node.Repo(repo.NewMemory(nil)),
			node.MockHost(mn),
			node.Test(),

			genesis,
		)
		if err != nil {
			t.Fatal(err)
		}

	}

	for i, full := range storage {
		// TODO: support non-bootstrap miners
		if i != 0 {
			t.Fatal("only one storage node supported")
		}
		if full != 0 {
			t.Fatal("storage nodes only supported on the first full node")
		}

		f := fulls[full]

		genMiner := maddrs[i]
		wa := genms[i].Worker

		storers[i] = testStorageNode(ctx, t, wa, genMiner, pk, f, mn, node.Options())

		sma := storers[i].StorageMiner.(*impl.StorageMinerAPI)

		psd := presealDirs[i]
		mds, err := badger.NewDatastore(filepath.Join(psd, "badger"), nil)
		if err != nil {
			t.Fatal(err)
		}

		osb, err := sectorbuilder.New(&sectorbuilder.Config{
			SectorSize:    1024,
			WorkerThreads: 2,
			Miner:         genMiner,
			Paths:         sectorbuilder.SimplePath(psd),
		}, namespace.Wrap(mds, datastore.NewKey("/sectorbuilder")))
		if err != nil {
			t.Fatal(err)
		}

		if err := sma.SectorBuilder.(*sectorbuilder.SectorBuilder).ImportFrom(osb, false); err != nil {
			t.Fatal(err)
		}

	}

	if err := mn.LinkAll(); err != nil {
		t.Fatal(err)
	}

	return fulls, storers
}

func mockSbBuilder(t *testing.T, nFull int, storage []int) ([]test.TestNode, []test.TestStorageNode) {
	ctx := context.Background()
	mn := mocknet.New(ctx)

	fulls := make([]test.TestNode, nFull)
	storers := make([]test.TestStorageNode, len(storage))

	pk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	minerPid, err := peer.IDFromPrivateKey(pk)
	require.NoError(t, err)

	_ = minerPid // TODO

	var genbuf bytes.Buffer

	if len(storage) > 1 {
		panic("need more peer IDs")
	}
	// PRESEAL SECTION, TRY TO REPLACE WITH BETTER IN THE FUTURE
	// TODO: would be great if there was a better way to fake the preseals

	var genms []genesis.Miner
	var maddrs []address.Address
	var presealDirs []string
	for i := 0; i < len(storage); i++ {
		maddr, err := address.NewIDAddress(300 + uint64(i))
		if err != nil {
			t.Fatal(err)
		}
		tdir, err := ioutil.TempDir("", "preseal-memgen")
		if err != nil {
			t.Fatal(err)
		}
		genm, err := sbmock.PreSeal(1024, maddr, 1)
		if err != nil {
			t.Fatal(err)
		}

		presealDirs = append(presealDirs, tdir)
		maddrs = append(maddrs, maddr)
		genms = append(genms, *genm)
	}
	templ := &genesis.Template{
		Miners: genms,
	}

	// END PRESEAL SECTION

	for i := 0; i < nFull; i++ {
		var genesis node.Option
		if i == 0 {
			genesis = node.Override(new(modules.Genesis), modtest.MakeGenesisMem(&genbuf, *templ))
		} else {
			genesis = node.Override(new(modules.Genesis), modules.LoadGenesis(genbuf.Bytes()))
		}

		var err error
		// TODO: Don't ignore stop
		_, err = node.New(ctx,
			node.FullAPI(&fulls[i].FullNode),
			node.Online(),
			node.Repo(repo.NewMemory(nil)),
			node.MockHost(mn),
			node.Test(),

			node.Override(new(sectorbuilder.Verifier), sbmock.MockVerifier),

			genesis,
		)
		if err != nil {
			t.Fatal(err)
		}

	}

	for i, full := range storage {
		// TODO: support non-bootstrap miners
		if i != 0 {
			t.Fatal("only one storage node supported")
		}
		if full != 0 {
			t.Fatal("storage nodes only supported on the first full node")
		}

		f := fulls[full]

		genMiner := maddrs[i]
		wa := genms[i].Worker

		storers[i] = testStorageNode(ctx, t, wa, genMiner, pk, f, mn, node.Options(
			node.Override(new(sectorbuilder.Interface), sbmock.NewMockSectorBuilder(5, build.SectorSizes[0])),
		))
	}

	if err := mn.LinkAll(); err != nil {
		t.Fatal(err)
	}

	return fulls, storers
}

func TestAPI(t *testing.T) {
	test.TestApis(t, builder)
}

func rpcBuilder(t *testing.T, nFull int, storage []int) ([]test.TestNode, []test.TestStorageNode) {
	fullApis, storaApis := builder(t, nFull, storage)
	fulls := make([]test.TestNode, nFull)
	storers := make([]test.TestStorageNode, len(storage))

	for i, a := range fullApis {
		rpcServer := jsonrpc.NewServer()
		rpcServer.Register("Filecoin", a)
		testServ := httptest.NewServer(rpcServer) //  todo: close

		var err error
		fulls[i].FullNode, _, err = client.NewFullNodeRPC("ws://"+testServ.Listener.Addr().String(), nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	for i, a := range storaApis {
		rpcServer := jsonrpc.NewServer()
		rpcServer.Register("Filecoin", a)
		testServ := httptest.NewServer(rpcServer) //  todo: close

		var err error
		storers[i].StorageMiner, _, err = client.NewStorageMinerRPC("ws://"+testServ.Listener.Addr().String(), nil)
		if err != nil {
			t.Fatal(err)
		}
		storers[i].MineOne = a.MineOne
	}

	return fulls, storers
}

func TestAPIRPC(t *testing.T) {
	test.TestApis(t, rpcBuilder)
}

func TestAPIDealFlow(t *testing.T) {
	logging.SetLogLevel("miner", "ERROR")
	logging.SetLogLevel("chainstore", "ERROR")
	logging.SetLogLevel("chain", "ERROR")
	logging.SetLogLevel("sub", "ERROR")
	logging.SetLogLevel("storageminer", "ERROR")

	test.TestDealFlow(t, mockSbBuilder, 10*time.Millisecond)
}

func TestAPIDealFlowReal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	test.TestDealFlow(t, builder, time.Second)
}
