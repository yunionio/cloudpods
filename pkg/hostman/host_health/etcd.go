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

package host_health

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

func NewEtcdOptions(
	opt *common_options.EtcdOptions, leaseTimeout, dialTimeout, requestTimeout int,
) (*etcd.SEtcdOptions, error) {
	cfg, err := opt.GetEtcdTLSConfig()
	if err != nil {
		return nil, err
	}
	return &etcd.SEtcdOptions{
		EtcdEndpoint:              opt.EtcdEndpoints,
		EtcdLeaseExpireSeconds:    leaseTimeout,
		EtcdTimeoutSeconds:        dialTimeout,
		EtcdRequestTimeoutSeconds: requestTimeout,
		EtcdEnabldSsl:             opt.EtcdUseTLS,
		TLSConfig:                 cfg,
	}, nil
}

type SEtcdClient struct {
	cli *etcd.SEtcdClient

	hostId        string
	onUnhealthy   func()
	timeout       int
	requestExpend int
}

func NewEtcdClient(opt *common_options.EtcdOptions, hostId string) (*SEtcdClient, error) {
	var dialTimeout, requestTimeout = 3, 2
	cfg, err := NewEtcdOptions(opt, options.HostOptions.HostLeaseTimeout, dialTimeout, requestTimeout)
	if err != nil {
		return nil, err
	}
	cli := new(SEtcdClient)
	err = etcd.InitDefaultEtcdClient(cfg, cli.OnKeepaliveFailure)
	if err != nil {
		return nil, errors.Wrap(err, "init default etcd client")
	}
	cli.cli = etcd.Default()
	cli.hostId = hostId
	cli.timeout = options.HostOptions.HostHealthTimeout - options.HostOptions.HostLeaseTimeout
	cli.requestExpend = requestTimeout
	return cli, nil
}

func (c *SEtcdClient) StartHostHealthCheck(ctx context.Context) error {
	return c.cli.PutSession(ctx,
		fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, c.hostId),
		api.HOST_HEALTH_STATUS_RUNNING,
	)
}

func (c *SEtcdClient) SetOnUnhealthy(onUnhealthy func()) {
	c.onUnhealthy = onUnhealthy
}

func (c *SEtcdClient) OnKeepaliveFailure() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(c.timeout))
	defer cancel()
	err := c.cli.RestartSessionWithContext(ctx)
	if err == nil {
		if err := c.cli.PutSession(context.Background(),
			fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, c.hostId),
			api.HOST_HEALTH_STATUS_RUNNING,
		); err != nil {
			log.Errorf("put host key failed %s", err)
		} else {
			log.Infof("etcd client restart session success")
			return
		}
	}
	log.Errorf("keep etcd lease failed: %s", err)
	if c.onUnhealthy != nil {
		c.onUnhealthy()
	}
	go c.Reconnect()
}

func (c *SEtcdClient) Reconnect() {
	if c.cli.SessionLiving() {
		return
	}
	for {
		if err := c.cli.RestartSession(); err != nil && !c.cli.SessionLiving() {
			log.Errorf("restart session failed %s", err)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	if err := c.cli.PutSession(context.Background(),
		fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, c.hostId),
		api.HOST_HEALTH_STATUS_RUNNING,
	); err != nil {
		log.Errorf("put host key failed %s", err)
		go c.Reconnect()
	} else {
		return
	}
}

func (c *SEtcdClient) Stop() error {
	return nil
}
