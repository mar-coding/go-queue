package rpc

import (
	"context"
	myType "github.com/mar-coding/go-queue/loadbalancer/types"
)

// Client defines the interface for RPC operations
type Client interface {
	Ping(ctx context.Context, address string, msg string) error
	CreateQueue(ctx context.Context, address string, req *myType.CreateQueueRequest) (*myType.CreateQueueResponse, error)
	AppendData(ctx context.Context, address string, req *myType.AppendDataRequest) (*myType.AppendDataResponse, error)
	ReadData(ctx context.Context, address string, req *myType.ReadDataRequest) (*myType.ReadDataResponse, error)
}

// Ensure ClientRPC implements the Client interface
var _ Client = (*ClientRPC)(nil)
