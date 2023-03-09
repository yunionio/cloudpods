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
	"io/ioutil"
	"path"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudmon/misc"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SHostHealthManager struct {
	cli *etcd.SEtcdClient

	timeout       int
	requestExpend int

	hostId string
	status string

	masterNodesIps []string
}

var (
	manager *SHostHealthManager
)

func InitHostHealthManager(hostId string) (*SHostHealthManager, error) {
	if manager != nil {
		return manager, nil
	}

	var m = SHostHealthManager{}
	masterNodesIps, err := m.masterNodesInternalIps()
	if err != nil {
		return nil, err
	} else if len(masterNodesIps) == 0 {
		return nil, errors.Errorf("failed get k8s master nodes")
	}
	m.masterNodesIps = masterNodesIps

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
	m.hostId = hostId
	m.requestExpend = requestTimeout
	m.timeout = options.HostOptions.HostHealthTimeout - options.HostOptions.HostLeaseTimeout

	if err := m.StartHealthCheck(); err != nil {
		return nil, err
	}
	log.Infof("put key %s success", m.GetKey())
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
		m.GetKey(), api.HOST_HEALTH_STATUS_RUNNING,
	)
}

func (m *SHostHealthManager) GetKey() string {
	return fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, m.hostId)
}

func (m *SHostHealthManager) OnKeepaliveFailure() {
	m.status = api.HOST_HEALTH_STATUS_RECONNECTING
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(m.timeout))
	defer cancel()
	err := m.cli.RestartSessionWithContext(ctx)
	if err == nil {
		if err := m.cli.PutSession(context.Background(),
			m.GetKey(), api.HOST_HEALTH_STATUS_RUNNING,
		); err != nil {
			log.Errorf("put host key failed %s", err)
		} else {
			m.status = api.HOST_HEALTH_STATUS_RUNNING
			log.Infof("etcd client restart session put %s success", m.GetKey())
			return
		}
	}
	log.Errorf("keep etcd lease failed: %s", err)

	if m.networkAvailable() {
		log.Infof("network is available, try reconnect")
		// may be etcd not work
		m.Reconnect()
	} else {
		log.Errorf("netwrok is unavailable, going to shutdown servers")
		m.status = api.HOST_HEALTH_STATUS_UNKNOWN
		m.OnUnhealth()
	}
}

func (m *SHostHealthManager) networkAvailable() bool {
	res, err := misc.Ping(m.masterNodesIps, 3, 10, true)
	if err != nil {
		log.Errorf("failed ping master nodes %s", res)
		return true
	}
	for _, v := range res {
		if v.Loss() < 100 {
			return true
		}
	}
	return false
}

func (m *SHostHealthManager) masterNodesInternalIps() ([]string, error) {
	result, err := modules.Hosts.Get(hostutils.GetComputeSession(context.Background()), "k8s-master-node-ips", nil)
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0)
	err = result.Unmarshal(&ips, "ips")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal master node ips")
	}
	return ips, nil
}

func (m *SHostHealthManager) OnUnhealth() {
	p := path.Join(options.HostOptions.ServersPath, hostconsts.HOST_HEALTH_FILENAME)
	if fileutils2.Exists(p) {
		if act, err := fileutils2.FileGetContents(p); err != nil {
			log.Errorf(" failed read file %s: %s", p, err)
		} else if act == hostconsts.SHUTDOWN_SERVERS {
			log.Errorf("Host unhealthy, going to shutdown servers")
			m.shutdownServers()
		}
	}
	// reconnect wait for network available
	m.Reconnect()
}

func (m *SHostHealthManager) Reconnect() {
	if m.cli.SessionLiving() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := m.cli.RestartSessionWithContext(ctx); err != nil && !m.cli.SessionLiving() {
		log.Errorf("restart session failed %s", err)
		go m.Reconnect()
		return
	}
	log.Infof("restart ression success")

	if err := m.cli.PutSession(
		context.Background(), m.GetKey(), api.HOST_HEALTH_STATUS_RUNNING,
	); err != nil {
		log.Errorf("put host key failed %s", err)
		go m.Reconnect()
		return
	}
	log.Infof("put key %s success", m.GetKey())
	m.status = api.HOST_HEALTH_STATUS_RUNNING
}

func (m *SHostHealthManager) shutdownServers() {
	files, err := ioutil.ReadDir(options.HostOptions.ServersPath)
	if err != nil {
		log.Errorf("failed walk dir %s: %s", options.HostOptions.ServersPath, err)
		return
	}
	for i := range files {
		if hostutils.IsGuestDir(files[i], options.HostOptions.ServersPath) {
			stopvm := path.Join(options.HostOptions.ServersPath, files[i].Name(), "stopvm")
			if fileutils2.Exists(stopvm) {
				log.Infof("start exec stopvm script for guest %s", files[i].Name())
				out, err := procutils.NewRemoteCommandAsFarAsPossible("bash", stopvm, "--force").Output()
				if err != nil {
					log.Errorf("failed exec stopvm script for guest %s: %s %s", files[i].Name(), out, err)
				}
			}
		}
	}
}

func GetHealthStatus() string {
	if manager == nil {
		return ""
	}
	return manager.status
}
