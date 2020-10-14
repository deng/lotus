package client

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"net/http"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
)

// NewCommonRPC creates a new http jsonrpc client.
func NewCommonRPC(ctx context.Context, addr string, requestHeader http.Header) (api.Common, jsonrpc.ClientCloser, error) {
	var res apistruct.CommonStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
	)

	return &res, closer, err
}

// NewFullNodeRPC creates a new http jsonrpc client.
func NewFullNodeRPC(ctx context.Context, addr string, requestHeader http.Header) (api.FullNode, jsonrpc.ClientCloser, error) {
	var res apistruct.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.CommonStruct.Internal,
			&res.Internal,
		}, requestHeader)

	return &res, closer, err
}

// NewStorageMinerRPC creates a new http jsonrpc client for miner
func NewStorageMinerRPC(ctx context.Context, addr string, requestHeader http.Header, opts ...jsonrpc.Option) (api.StorageMiner, jsonrpc.ClientCloser, error) {
	var res apistruct.StorageMinerStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.CommonStruct.Internal,
			&res.Internal,
		},
		requestHeader,
		opts...,
	)

	return &res, closer, err
}

func NewWorkerRPC(ctx context.Context, addr string, requestHeader http.Header, opts ...jsonrpc.Option) (api.WorkerAPI, jsonrpc.ClientCloser, error) {
	var res apistruct.WorkerStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
		opts...,
	)

	return &res, closer, err
}

// NewStorageSealerRPC creates a new http jsonrpc client for sealer
func NewStorageSealerRPC(ctx context.Context, addr string, requestHeader http.Header, opts ...jsonrpc.Option) (api.StorageSealer, jsonrpc.ClientCloser, error) {
	var res apistruct.StorageSealerStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.CommonStruct.Internal,
			&res.Internal,
		},
		requestHeader,
		opts...,
	)

	return &res, closer, err
}

// NewStorageDealerRPC creates a new http jsonrpc client for dealer
func NewStorageDealerRPC(ctx context.Context, addr string, requestHeader http.Header, opts ...jsonrpc.Option) (api.StorageDealer, jsonrpc.ClientCloser, error) {
	var res apistruct.StorageDealerStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.StorageSealerStruct.Internal,
			&res.CommonStruct.Internal,
			&res.Internal,
		},
		requestHeader,
		opts...,
	)

	return &res, closer, err
}

// NewGatewayRPC creates a new http jsonrpc client for a gateway node.
func NewGatewayRPC(ctx context.Context, addr string, requestHeader http.Header, opts ...jsonrpc.Option) (api.GatewayAPI, jsonrpc.ClientCloser, error) {
	var res apistruct.GatewayStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
		opts...,
	)

	return &res, closer, err
}

func NewWalletRPC(ctx context.Context, addr string, requestHeader http.Header) (api.WalletAPI, jsonrpc.ClientCloser, error) {
	var res apistruct.WalletStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin",
		[]interface{}{
			&res.Internal,
		},
		requestHeader,
	)

	return &res, closer, err
}
