package host_health

import (
	"context"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

type Status int

const (
	UNKNOWN Status = iota
	HEALTHY
	UNHEALTHY
)

type SHostHealthManager struct {
	cli          Client
	status       Status
	guestManager *guestman.SGuestManager
}

var manager *SHostHealthManager

func InitHostHealthManager(hostId string) (*SHostHealthManager, error) {
	if manager != nil {
		return manager, nil
	}
	switch options.HostOptions.HealthDriver {
	case "etcd":
		cli, err := NewEtcdClient(&options.HostOptions.EtcdOptions, hostId)
		if err != nil {
			return nil, errors.Wrap(err, "new etcd client")
		}
		manager = new(SHostHealthManager)
		manager.cli = cli
	default:
		return nil, fmt.Errorf("not support health driver %s", options.HostOptions.HealthDriver)
	}
	manager.guestManager = guestman.GetGuestManager()
	manager.cli.SetOnUnhealthy(manager.OnUnhealth)
	go manager.StartHealthCheck()
	return manager, nil
}

func (m *SHostHealthManager) StartHealthCheck() error {
	return m.cli.StartHostHealthCheck(context.Background())
}

func (m *SHostHealthManager) OnUnhealth() {
	log.Debugf("Host unhealthy, going to shotdown servers")
	m.status = UNHEALTHY
	if options.HostOptions.HealthShutdownServers {
		m.shutdownServers()
	}
}

// shutdown servers used shared storage
func (m *SHostHealthManager) shutdownServers() {
	m.guestManager.ShutdownSharedStorageServers()
}

func (m *SHostHealthManager) Stop() error {
	return m.cli.Stop()
}

type Client interface {
	StartHostHealthCheck(context.Context) error
	SetOnUnhealthy(func())
	Stop() error
}
