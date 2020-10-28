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
	"os"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

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
	onHostDown   string
}

const SHUTDOWN_SERVERS = "shutdown-servers"

var (
	manager         *SHostHealthManager
	HostDownActions = []string{SHUTDOWN_SERVERS}
)

func InitHostHealthManager(hostId, onHostDown string) (*SHostHealthManager, error) {
	if manager != nil {
		return manager, nil
	}
	var m *SHostHealthManager
	switch options.HostOptions.HealthDriver {
	case "etcd":
		cli, err := NewEtcdClient(&options.HostOptions.EtcdOptions, hostId)
		if err != nil {
			return nil, errors.Wrap(err, "new etcd client")
		}
		m = new(SHostHealthManager)
		m.cli = cli
	default:
		return nil, fmt.Errorf("not support health driver %s", options.HostOptions.HealthDriver)
	}
	m.guestManager = guestman.GetGuestManager()
	m.onHostDown = onHostDown
	m.cli.SetOnUnhealthy(m.OnUnhealth)
	if err := m.StartHealthCheck(); err != nil {
		return nil, err
	}
	manager = m
	return manager, nil
}

func (m *SHostHealthManager) StartHealthCheck() error {
	return m.cli.StartHostHealthCheck(context.Background())
}

func (m *SHostHealthManager) OnUnhealth() {
	log.Debugf("Host unhealthy, going to shotdown servers")
	m.status = UNHEALTHY
	if m.onHostDown == SHUTDOWN_SERVERS {
		m.shutdownServers()
	}
	utils.DumpAllGoroutineStack(log.Logger().Out)
	os.Exit(1)
}

func (m *SHostHealthManager) SetOnHostDown(onHostDown string) {
	m.onHostDown = onHostDown
}

// shutdown servers used shared storage
func (m *SHostHealthManager) shutdownServers() {
	m.guestManager.ShutdownSharedStorageServers()
}

func (m *SHostHealthManager) Stop() error {
	return m.cli.Stop()
}

func SetOnHostDown(onHostDown string) error {
	if manager != nil {
		manager.SetOnHostDown(onHostDown)
		return nil
	}
	return fmt.Errorf("host health manager not init")
}

type Client interface {
	StartHostHealthCheck(context.Context) error
	SetOnUnhealthy(func())
	Stop() error
}
