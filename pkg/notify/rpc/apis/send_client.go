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

package apis

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

type SendNotificationClient struct {
	sendAgentClient
	Conn        *grpc.ClientConn
	CallTimeout time.Duration
}

func NewSendNotificationClient(cc *grpc.ClientConn) *SendNotificationClient {
	return &SendNotificationClient{
		sendAgentClient: sendAgentClient{cc},
		Conn:            cc,
		CallTimeout:     30 * time.Second,
	}
}

func (c *SendNotificationClient) Send(ctx context.Context, in *SendParams, opts ...grpc.CallOption) (*Empty, error) {
	ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
	defer cancel()
	return c.sendAgentClient.Send(ctx, in, opts...)
}

func (c *SendNotificationClient) AddConfig(ctx context.Context, in *AddConfigInput, opts ...grpc.CallOption) (*Empty, error) {
    ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
    defer cancel()
    return c.sendAgentClient.AddConfig(ctx, in, opts...)
}

func (c *SendNotificationClient) DeleteConfig(ctx context.Context, in *DeleteConfigInput, opts ...grpc.CallOption) (*Empty, error) {
    ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
    defer cancel()
    return c.sendAgentClient.DeleteConfig(ctx, in, opts...)
}

func (c *SendNotificationClient) UpdateConfig(ctx context.Context, in *UpdateConfigInput, opts ...grpc.CallOption) (*Empty, error) {
	ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
	defer cancel()
	return c.sendAgentClient.UpdateConfig(ctx, in, opts...)
}

func (c *SendNotificationClient) CompleteConfig(ctx context.Context, in *CompleteConfigInput, opts ...grpc.CallOption) (*Empty, error) {
    ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
    defer cancel()
    return c.sendAgentClient.CompleteConfig(ctx, in, opts...)
}

func (c *SendNotificationClient) ValidateConfig(ctx context.Context, in *ValidateConfigInput, opts ...grpc.CallOption) (*ValidateConfigReply, error) {
	ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
	defer cancel()
	return c.sendAgentClient.ValidateConfig(ctx, in, opts...)
}

func (c *SendNotificationClient) UseridByMobile(ctx context.Context, in *UseridByMobileParams, opts ...grpc.CallOption) (*UseridByMobileReply, error) {
	ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
	defer cancel()
	return c.sendAgentClient.UseridByMobile(ctx, in, opts...)
}

func (c *SendNotificationClient) Ready(ctx context.Context, in *ReadyInput, opts ...grpc.CallOption) (*ReadyOutput, error) {
    ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
    defer cancel()
    return c.sendAgentClient.Ready(ctx, in, opts...)
}

func (c *SendNotificationClient) BatchSend(ctx context.Context, in *BatchSendParams, opts ...grpc.CallOption) (*BatchSendReply, error) {
	ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
	defer cancel()
	return c.sendAgentClient.BatchSend(ctx, in, opts...)
}
