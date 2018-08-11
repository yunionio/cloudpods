package sync

import (
	"yunion.io/x/onecloud/pkg/scheduler/cache"
)

func NewSyncManager(stopCh <-chan struct{}) *cache.GroupManager {
	items := defaultSyncItems()
	return cache.NewGroupManager(CacheKind, items, stopCh)
}

type syncItem struct {
	cache.CachedItem
}
