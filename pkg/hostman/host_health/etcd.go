package host_health

import (
	"context"
	"fmt"

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
	var timeout = c.timeout
	for timeout > 0 {
		timeout -= c.requestExpend
		if err := c.cli.RestartSession(); err != nil {
			log.Errorf("restart session failed %s", err)
		} else {
			break
		}
	}
	if timeout > 0 {
		if err := c.cli.PutSession(context.Background(),
			fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, c.hostId),
			api.HOST_HEALTH_STATUS_RUNNING,
		); err != nil {
			log.Errorf("put host key failed %s", err)
		} else {
			return
		}
	}
	log.Errorln("keep etcd lease failed")
	if c.onUnhealthy != nil {
		c.onUnhealthy()
	}
}

func (c *SEtcdClient) Stop() error {
	return nil
}
