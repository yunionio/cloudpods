package deployclient

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"

	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

func InitDeployClient(socketPath string) (deployapi.DeployAgentClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, socketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		return nil, err
	}

	client := deployapi.NewDeployAgentClient(conn)
	deployClient = &client
	return &client, nil
}

var deployClient deployapi.DeployAgentClient

func GetDeployClient() deployapi.DeployAgentClient {
	return deployClient
}
