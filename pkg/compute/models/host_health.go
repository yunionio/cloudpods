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
}

func hostKey(hostId string) string {
	return fmt.Sprintf("%s/%s", api.HOST_HEALTH_PREFIX, hostId)
}

func InitHostHealthChecker(cli *etcd.SEtcdClient, timeout int) *SHostHealthChecker {
	if hostHealthChecker != nil {
		return hostHealthChecker
	}
	hostHealthChecker = &SHostHealthChecker{cli, time.Duration(timeout) * time.Second}
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
	log.Debugf("Start watch host %s", hostId)
	var (
		ch  chan struct{}
		key = hostKey(hostId)
	)
	_, err := h.cli.Get(ctx, key)
	if err == etcd.ErrNoSuchKey {
		log.Errorf("No such key %s", hostId)
		ch = make(chan struct{})
		go func() {
			select {
			case <-time.NewTimer(h.timeout).C:
				h.onHostUnhealthy(ctx, hostId)
			case <-ch:
				h.startWatcher(ctx, hostId)
			case <-ctx.Done():
				log.Infof("exit watch host %s", hostId)
			}
		}()
	}
	h.cli.Watch(ctx, key, h.onHostOnline(hostId, ch), h.onHostOffline(hostId))
}

func (h *SHostHealthChecker) onHostUnhealthy(ctx context.Context, hostId string) {
	lockman.LockRawObject(ctx, api.HOST_HEALTH_LOCK_PREFIX, hostId)
	defer lockman.ReleaseRawObject(ctx, api.HOST_HEALTH_LOCK_PREFIX, hostId)
	host := HostManager.FetchHostById(hostId)
	if host.EnableHealthCheck == true {
		host.OnHostDown(ctx, auth.AdminCredential())
	}
}

func (h *SHostHealthChecker) onHostOnline(hostId string, ch chan struct{}) etcd.TEtcdCreateEventFunc {
	return func(key, value []byte) {
		log.Debugf("Got host online %s", hostId)
		if ch != nil {
			close(ch)
		}
	}
}

func (h *SHostHealthChecker) onHostOffline(hostId string) etcd.TEtcdModifyEventFunc {
	return func(key, oldvalue, value []byte) {
		log.Warningf("host %s disconnect with etcd", hostId)
		host := HostManager.FetchHostById(hostId)
		if host.EnableHealthCheck == true {
			h.startWatcher(context.Background(), hostId)
		}
	}
}

func (h *SHostHealthChecker) WatchHost(ctx context.Context, hostId string) {
	h.cli.Unwatch(hostKey(hostId))
	h.startWatcher(ctx, hostId)
}

func (h *SHostHealthChecker) UnwatchHost(ctx context.Context, hostId string) {
	log.Debugf("Unwatch host %s", hostId)
	h.cli.Unwatch(hostKey(hostId))
}
