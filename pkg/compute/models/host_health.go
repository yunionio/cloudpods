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

package models

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var hostHealthChecker *SHostHealthChecker

type SHostHealthChecker struct {
	// etcd client
	cli *etcd.SEtcdClient
	// time of wait host reconnect
	timeout time.Duration
	// hosts chan
	hc map[string]chan struct{}
}

func hostKey(hostname string) string {
	return fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, hostname)
}

func InitHostHealthChecker(cli *etcd.SEtcdClient, timeout int) *SHostHealthChecker {
	if hostHealthChecker != nil {
		return hostHealthChecker
	}
	hostHealthChecker = &SHostHealthChecker{
		cli:     cli,
		timeout: time.Duration(timeout) * time.Second,
		hc:      make(map[string]chan struct{}),
	}
	return hostHealthChecker
}

func (h *SHostHealthChecker) StartHostsHealthCheck(ctx context.Context) error {
	log.Infof("Start host health check......")
	return h.startHealthCheck(ctx)
}

func (h *SHostHealthChecker) startHealthCheck(ctx context.Context) error {
	q := HostManager.Query().IsTrue("enabled").IsTrue("enable_health_check").Equals("host_type", api.HOST_TYPE_HYPERVISOR)
	rows, err := q.Rows()
	if err != nil {
		log.Errorf("HostHealth check Query hosts %s", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		host := new(SHost)
		err = q.Row2Struct(rows, host)
		if err != nil {
			return errors.Wrap(err, "q.Row2Struct")
		}
		host.SetModelManager(HostManager, host)
		err = h.startWatcher(ctx, host.GetHostnameByName())
		if err != nil {
			return errors.Wrap(err, "startWatcher")
		}
	}
	return nil
}

func (h *SHostHealthChecker) startWatcher(ctx context.Context, hostname string) error {
	log.Infof("Start watch host %s", hostname)
	var key = hostKey(hostname)

	if _, ok := h.hc[hostname]; !ok {
		h.hc[hostname] = make(chan struct{})
	}
	if err := h.cli.Watch(
		ctx, key,
		h.onHostOnline(ctx, hostname),
		h.onHostOffline(ctx, hostname),
		h.onHostOfflineDeleted(ctx, hostname),
	); err != nil {
		return err
	}

	// watched key not found, wait 60s(default) and do onHostUnhealthy
	_, err := h.cli.Get(ctx, key)
	if err == etcd.ErrNoSuchKey {
		log.Warningf("No such key %s", hostname)
		go func() {
			select {
			case <-time.NewTimer(h.timeout).C:
				h.onHostUnhealthy(ctx, hostname)
			case <-h.hc[hostname]:
				if _err := h.startWatcher(ctx, hostname); _err != nil {
					log.Errorf("failed start watcher %s", _err)
				}
			case <-ctx.Done():
				log.Infof("exit watch host %s", hostname)
			}
		}()
		return nil
	}
	return err
}

func (h *SHostHealthChecker) onHostUnhealthy(ctx context.Context, hostname string) {
	lockman.LockRawObject(ctx, api.HOST_HEALTH_LOCK_PREFIX, hostname)
	defer lockman.ReleaseRawObject(ctx, api.HOST_HEALTH_LOCK_PREFIX, hostname)
	host := HostManager.FetchHostByHostname(hostname)
	if host != nil && !utils.IsInStringArray(host.RemoteHealthStatus(ctx),
		[]string{api.HOST_HEALTH_STATUS_RECONNECTING, api.HOST_HEALTH_STATUS_RUNNING},
	) {
		// in case hostagent health manager in status reconnecting
		host.OnHostDown(ctx, auth.AdminCredential())
	}
}

func (h *SHostHealthChecker) onHostOnline(ctx context.Context, hostname string) etcd.TEtcdCreateEventFunc {
	return func(ctx context.Context, key, value []byte) {
		log.Infof("Got host online %s", hostname)
		if h.hc[hostname] != nil {
			h.hc[hostname] <- struct{}{}
		}
	}
}

func (h *SHostHealthChecker) processHostOffline(ctx context.Context, hostname string) {
	log.Warningf("host %s disconnect with etcd", hostname)
	go func() {
		select {
		case <-time.NewTimer(h.timeout).C:
			h.onHostUnhealthy(ctx, hostname)
		case <-h.hc[hostname]:
			if err := h.startWatcher(ctx, hostname); err != nil {
				log.Errorf("failed start watcher %s", err)
			}
		}
	}()
}

func (h *SHostHealthChecker) onHostOffline(ctx context.Context, hostname string) etcd.TEtcdModifyEventFunc {
	return func(ctx context.Context, key, oldvalue, value []byte) {
		log.Errorf("watch host key modified %s %s %s", key, oldvalue, value)
		h.processHostOffline(ctx, hostname)
	}
}

func (h *SHostHealthChecker) onHostOfflineDeleted(ctx context.Context, hostname string) etcd.TEtcdDeleteEventFunc {
	return func(ctx context.Context, key []byte) {
		log.Errorf("watch host key deleled %s", key)
		h.processHostOffline(ctx, hostname)
	}
}

func (h *SHostHealthChecker) WatchHost(ctx context.Context, hostname string) error {
	h.cli.Unwatch(hostKey(hostname))
	return h.startWatcher(ctx, hostname)
}

func (h *SHostHealthChecker) UnwatchHost(ctx context.Context, hostname string) {
	log.Infof("Unwatch host %s", hostname)
	h.cli.Unwatch(hostKey(hostname))
	delete(h.hc, hostname)
}
