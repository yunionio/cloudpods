// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deployclient

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"

	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

var (
	deployClient *DeployClient
	_            deployapi.DeployAgentClient = deployClient
)

func Init(socketPath string) {
	deployClient = NewDeployClient(socketPath)
}

type DeployClient struct {
	socketPath string
}

func NewDeployClient(socketPath string) *DeployClient {
	return &DeployClient{socketPath}
}

func GetDeployClient() *DeployClient {
	return deployClient
}

func grcpDialWithUnixSocket(ctx context.Context, socketPath string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, socketPath, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second*3),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
}

func (c *DeployClient) DeployGuestFs(ctx context.Context, in *deployapi.DeployParams, opts ...grpc.CallOption) (*deployapi.DeployGuestFsResponse, error) {
	conn, err := grcpDialWithUnixSocket(ctx, c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := deployapi.NewDeployAgentClient(conn)
	ret, err := client.DeployGuestFs(ctx, in, opts...)
	return ret, err
}

func (c *DeployClient) ResizeFs(ctx context.Context, in *deployapi.ResizeFsParams, opts ...grpc.CallOption) (*deployapi.Empty, error) {
	conn, err := grcpDialWithUnixSocket(ctx, c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := deployapi.NewDeployAgentClient(conn)
	return client.ResizeFs(ctx, in, opts...)
}

func (c *DeployClient) FormatFs(ctx context.Context, in *deployapi.FormatFsParams, opts ...grpc.CallOption) (*deployapi.Empty, error) {
	conn, err := grcpDialWithUnixSocket(ctx, c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := deployapi.NewDeployAgentClient(conn)
	return client.FormatFs(ctx, in, opts...)
}

func (c *DeployClient) SaveToGlance(ctx context.Context, in *deployapi.SaveToGlanceParams, opts ...grpc.CallOption) (*deployapi.SaveToGlanceResponse, error) {
	conn, err := grcpDialWithUnixSocket(ctx, c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := deployapi.NewDeployAgentClient(conn)
	return client.SaveToGlance(ctx, in, opts...)
}

func (c *DeployClient) ProbeImageInfo(ctx context.Context, in *deployapi.ProbeImageInfoPramas, opts ...grpc.CallOption) (*deployapi.ImageInfo, error) {
	conn, err := grcpDialWithUnixSocket(ctx, c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := deployapi.NewDeployAgentClient(conn)
	return client.ProbeImageInfo(ctx, in, opts...)
}

func (c *DeployClient) ConnectEsxiDisks(
	ctx context.Context, in *deployapi.ConnectEsxiDisksParams, opts ...grpc.CallOption,
) (*deployapi.EsxiDisksConnectionInfo, error) {
	conn, err := grcpDialWithUnixSocket(ctx, c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := deployapi.NewDeployAgentClient(conn)
	return client.ConnectEsxiDisks(ctx, in, opts...)
}

func (c *DeployClient) DisconnectEsxiDisks(
	ctx context.Context, in *deployapi.EsxiDisksConnectionInfo, opts ...grpc.CallOption,
) (*deployapi.Empty, error) {
	conn, err := grcpDialWithUnixSocket(ctx, c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := deployapi.NewDeployAgentClient(conn)
	return client.DisconnectEsxiDisks(ctx, in, opts...)
}
