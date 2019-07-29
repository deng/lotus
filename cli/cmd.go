package cli

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	logging "github.com/ipfs/go-log"
	manet "github.com/multiformats/go-multiaddr-net"
	"golang.org/x/xerrors"
	"gopkg.in/urfave/cli.v2"

	"github.com/filecoin-project/go-lotus/api"
	"github.com/filecoin-project/go-lotus/api/client"
	"github.com/filecoin-project/go-lotus/node/repo"
)

var log = logging.Logger("cli")

const (
	metadataTraceConetxt = "traceContext"
	metadataContext      = "context"
)

// ApiConnector returns API instance
type ApiConnector func() api.FullNode

func getAPI(ctx *cli.Context, repoFlag string) (string, http.Header, error) {
	r, err := repo.NewFS(ctx.String(repoFlag))
	if err != nil {
		return "", nil, err
	}

	ma, err := r.APIEndpoint()
	if err != nil {
		return "", nil, xerrors.Errorf("failed to get api endpoint: %w", err)
	}
	_, addr, err := manet.DialArgs(ma)
	if err != nil {
		return "", nil, err
	}
	var headers http.Header
	token, err := r.APIToken()
	if err != nil {
		log.Warnf("Couldn't load CLI token, capabilities may be limited: %w", err)
	} else {
		headers = http.Header{}
		headers.Add("Authorization", "Bearer "+string(token))
	}

	return "ws://" + addr + "/rpc/v0", headers, nil
}

func GetAPI(ctx *cli.Context) (api.FullNode, error) {
	addr, headers, err := getAPI(ctx, "repo")
	if err != nil {
		return nil, err
	}

	return client.NewFullNodeRPC(addr, headers)
}

func GetStorageMinerAPI(ctx *cli.Context) (api.StorageMiner, error) {
	addr, headers, err := getAPI(ctx, "storagerepo")
	if err != nil {
		return nil, err
	}

	return client.NewStorageMinerRPC(addr, headers)
}

// ReqContext returns context for cli execution. Calling it for the first time
// installs SIGTERM handler that will close returned context.
// Not safe for concurrent execution.
func ReqContext(cctx *cli.Context) context.Context {
	if uctx, ok := cctx.App.Metadata[metadataContext]; ok {
		// unchecked cast as if something else is in there
		// it is crash worthy either way
		return uctx.(context.Context)
	}
	var tCtx context.Context

	if mtCtx, ok := cctx.App.Metadata[metadataTraceConetxt]; ok {
		tCtx = mtCtx.(context.Context)
	} else {
		tCtx = context.Background()
	}

	ctx, done := context.WithCancel(tCtx)
	sigChan := make(chan os.Signal, 2)
	go func() {
		<-sigChan
		done()
	}()
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	return ctx
}

var Commands = []*cli.Command{
	chainCmd,
	clientCmd,
	minerCmd,
	mpoolCmd,
	netCmd,
	versionCmd,
	walletCmd,
	createMinerCmd,
}
