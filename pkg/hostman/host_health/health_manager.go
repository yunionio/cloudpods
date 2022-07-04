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
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/hostman/guestman/types"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SHostHealthManager struct {
	cli *etcd.SEtcdClient

	timeout       int
	requestExpend int

	hostId     string
	status     string
	onHostDown string
}

var (
	manager         *SHostHealthManager
	HostDownActions = []string{hostconsts.SHUTDOWN_SERVERS}
)

func InitHostHealthManager(hostId, onHostDown string) (*SHostHealthManager, error) {
	if manager != nil {
		return manager, nil
	}

	var m = SHostHealthManager{}
	var dialTimeout, requestTimeout = 3, 2

	cfg, err := NewEtcdOptions(
		&options.HostOptions.EtcdOptions,
		options.HostOptions.HostLeaseTimeout,
		dialTimeout, requestTimeout,
	)
	if err != nil {
		return nil, err
	}
	err = etcd.InitDefaultEtcdClient(cfg, m.OnKeepaliveFailure)
	if err != nil {
		return nil, errors.Wrap(err, "init default etcd client")
	}

	m.cli = etcd.Default()
	m.onHostDown = onHostDown
	m.hostId = hostId
	m.requestExpend = requestTimeout
	m.timeout = options.HostOptions.HostHealthTimeout - options.HostOptions.HostLeaseTimeout

	if err := m.StartHealthCheck(); err != nil {
		return nil, err
	}
	m.status = api.HOST_HEALTH_STATUS_RUNNING
	manager = &m
	return manager, nil
}

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

func (m *SHostHealthManager) StartHealthCheck() error {
	return m.cli.PutSession(context.Background(),
		fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, m.hostId),
		api.HOST_HEALTH_STATUS_RUNNING,
	)
}

func (m *SHostHealthManager) OnKeepaliveFailure() {
	m.status = api.HOST_HEALTH_STATUS_RECONNECTING
	nicRecord := m.recordNic()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(m.timeout))
	defer cancel()
	err := m.cli.RestartSessionWithContext(ctx)
	if err == nil {
		if err := m.cli.PutSession(context.Background(),
			fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, m.hostId),
			api.HOST_HEALTH_STATUS_RUNNING,
		); err != nil {
			log.Errorf("put host key failed %s", err)
		} else {
			m.status = api.HOST_HEALTH_STATUS_RUNNING
			log.Infof("etcd client restart session success")
			return
		}
	}
	log.Errorf("keep etcd lease failed: %s", err)

	if m.networkAvailable(nicRecord) {
		log.Infof("network is available, try reconnect")
		// may be etcd not work
		m.Reconnect()
	} else {
		log.Errorf("netwrok is unavailable, going to shutdown servers")
		m.status = api.HOST_HEALTH_STATUS_UNKNOWN
		m.OnUnhealth()
	}
}

func (m *SHostHealthManager) recordNic() map[string]int {
	nicRecord := make(map[string]int)
	for _, n := range options.HostOptions.Networks {
		data := strings.Split(n, "/")
		interf := data[0]
		rx, err := fileutils2.FileGetContents(
			fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", interf),
		)
		if err != nil {
			log.Errorf("failed get nic rx %s  statistics %s", interf, err)
			continue
		}
		tx, err := fileutils2.FileGetContents(
			fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", interf),
		)
		if err != nil {
			log.Errorf("failed get nic tx %s  statistics %s", interf, err)
			continue
		}
		irx, err := strconv.Atoi(strings.TrimSpace(rx))
		if err != nil {
			log.Errorf("failed convert rx %s %s", rx, err)
		}
		itx, err := strconv.Atoi(strings.TrimSpace(tx))
		if err != nil {
			log.Errorf("failed convert tx %s %s", tx, err)
		}
		nicRecord[interf] = irx + itx
	}
	return nicRecord
}

func (m *SHostHealthManager) networkAvailable(oldRecord map[string]int) bool {
	newRecord := m.recordNic()
	for _, n := range options.HostOptions.Networks {
		data := strings.Split(n, "/")
		interf := data[0]

		oldR, ok := oldRecord[interf]
		if !ok {
			continue
		}
		newR, ok := newRecord[interf]
		if !ok {
			log.Errorf("nic %s record not found", n)
			continue
		}

		if newR != oldR {
			return true
		}
	}
	return false
}

func (m *SHostHealthManager) OnUnhealth() {
	if m.onHostDown == hostconsts.SHUTDOWN_SERVERS {
		log.Errorf("Host unhealthy, going to shotdown servers")
		m.shutdownServers()
	}
	// reconnect wait for network available
	m.Reconnect()
	utils.DumpAllGoroutineStack(log.Logger().Out)
	os.Exit(1)
}

func (m *SHostHealthManager) Reconnect() {
	if m.cli.SessionLiving() {
		return
	}
	for {
		if err := m.cli.RestartSession(); err != nil && !m.cli.SessionLiving() {
			log.Errorf("restart session failed %s", err)
			time.Sleep(1 * time.Second)
		} else {
			log.Infof("restart ression success")
			break
		}
	}
	if err := m.cli.PutSession(context.Background(),
		fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, m.hostId),
		api.HOST_HEALTH_STATUS_RUNNING,
	); err != nil {
		log.Errorf("put host key failed %s", err)
		go m.Reconnect()
	} else {
		m.status = api.HOST_HEALTH_STATUS_RUNNING
		log.Infof("put key %s/%s success", api.HOST_HEALTH_PREFIX, m.hostId)
		return
	}
}

func (m *SHostHealthManager) SetOnHostDown(onHostDown string) {
	m.onHostDown = onHostDown
}

// shutdown servers used shared storage
func (m *SHostHealthManager) shutdownServers() {
	types.HealthCheckReactor.ShutdownServers()
}

func SetOnHostDown(onHostDown string) error {
	if manager != nil {
		manager.SetOnHostDown(onHostDown)
		return nil
	}
	return fmt.Errorf("host health manager not init")
}

func GetHealthStatus() string {
	if manager == nil {
		return ""
	}
	return manager.status
}
