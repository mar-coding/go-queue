// registration.go
package service

import (
	"fmt"
	myRPC "github.com/mar-coding/go-queue/internal/transport"
	"net/rpc"
)

func RegisterWithLoadBalancer(nodeID, port, rpcPort, loadBalancerAddr string) error {
	client, err := rpc.Dial("tcp", loadBalancerAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to load balancer: %w", err)
	}
	defer client.Close()

	req := &myRPC.RegisterNodeRequest{
		ID:      nodeID,
		Port:    port,
		RPCPort: rpcPort,
	}
	var resp myRPC.RegisterNodeResponse // Changed from RegisterNodeRequest to RegisterNodeResponse

	err = client.Call("NodeService.Register", req, &resp)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	return nil
}
