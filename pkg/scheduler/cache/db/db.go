package db

import (
	"yunion.io/x/onecloud/pkg/scheduler/cache"
)

type dbItem struct {
	cache.CachedItem
	fields []string
}

func NewCacheManager(stopCh <-chan struct{}) *cache.GroupManager {
	return cache.NewGroupManager(CacheKind, DefaultCachedItems(), stopCh)
}
