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

func hostKey(hostId string) string {
	return fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, hostId)
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

func (h *SHostHealthChecker) StartHostsHealthCheck(ctx context.Context) {
	log.Infof("Start host health check......")
	h.startHealthCheck(ctx)
}

func (h *SHostHealthChecker) startHealthCheck(ctx context.Context) {
	q := HostManager.Query().IsTrue("enabled").IsTrue("enable_health_check").Equals("host_type", api.HOST_TYPE_HYPERVISOR)
	rows, err := q.Rows()
	if err != nil {
		log.Errorf("HostHealth check Query hosts %s", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		host := new(SHost)
		q.Row2Struct(rows, host)
		host.SetModelManager(HostManager, host)
		h.startWatcher(ctx, host.Id)
	}
}

func (h *SHostHealthChecker) startWatcher(ctx context.Context, hostId string) {
	log.Infof("Start watch host %s", hostId)
	var (
		ch  chan struct{}
		key = hostKey(hostId)
	)

	_, err := h.cli.Get(ctx, key)
	if err == etcd.ErrNoSuchKey {
		log.Warningf("No such key %s", hostId)
		ch = make(chan struct{})
		go func() {
			select {
			case <-time.NewTimer(h.timeout).C:
				h.onHostUnhealthy(ctx, hostId)
			case <-h.hc[hostId]:
				h.startWatcher(ctx, hostId)
			case <-ctx.Done():
				log.Infof("exit watch host %s", hostId)
			}
		}()
	}
	if _, ok := h.hc[hostId]; !ok {
		h.hc[hostId] = ch
	}
	h.cli.Watch(
		ctx, key,
		h.onHostOnline(ctx, hostId),
		h.onHostOffline(ctx, hostId),
		h.onHostOfflineDeleted(ctx, hostId),
	)
}

func (h *SHostHealthChecker) onHostUnhealthy(ctx context.Context, hostId string) {
	lockman.LockRawObject(ctx, api.HOST_HEALTH_LOCK_PREFIX, hostId)
	defer lockman.ReleaseRawObject(ctx, api.HOST_HEALTH_LOCK_PREFIX, hostId)
	host := HostManager.FetchHostById(hostId)
	if host != nil && host.EnableHealthCheck == true {
		host.OnHostDown(ctx, auth.AdminCredential())
	}
}

func (h *SHostHealthChecker) onHostOnline(ctx context.Context, hostId string) etcd.TEtcdCreateEventFunc {
	return func(ctx context.Context, key, value []byte) {
		log.Infof("Got host online %s", hostId)
		if h.hc[hostId] != nil {
			h.hc[hostId] <- struct{}{}
		}
	}
}

func (h *SHostHealthChecker) processHostOffline(ctx context.Context, hostId string) {
	log.Warningf("host %s disconnect with etcd", hostId)
	go func() {
		select {
		case <-time.NewTimer(h.timeout).C:
			h.onHostUnhealthy(ctx, hostId)
		case <-h.hc[hostId]:
			h.startWatcher(ctx, hostId)
		}
	}()
}

func (h *SHostHealthChecker) onHostOffline(ctx context.Context, hostId string) etcd.TEtcdModifyEventFunc {
	return func(ctx context.Context, key, oldvalue, value []byte) {
		log.Errorf("watch host key modified %s %s %s", key, oldvalue, value)
		h.processHostOffline(ctx, hostId)
	}
}

func (h *SHostHealthChecker) onHostOfflineDeleted(ctx context.Context, hostId string) etcd.TEtcdDeleteEventFunc {
	return func(ctx context.Context, key []byte) {
		log.Errorf("watch host key deleled %s", key)
		h.processHostOffline(ctx, hostId)
	}
}

func (h *SHostHealthChecker) WatchHost(ctx context.Context, hostId string) {
	h.cli.Unwatch(hostKey(hostId))
	h.startWatcher(ctx, hostId)
}

func (h *SHostHealthChecker) UnwatchHost(ctx context.Context, hostId string) {
	log.Infof("Unwatch host %s", hostId)
	h.cli.Unwatch(hostKey(hostId))
	delete(h.hc, hostId)
}
