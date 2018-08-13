package candidate

import (
	"yunion.io/x/onecloud/pkg/scheduler/cache"
)

type candidateItem struct {
	cache.CachedItem
}

func NewCandidateManager(db DBGroupCacher, sync SyncGroupCacher, stopCh <-chan struct{}) *cache.GroupManager {
	items := defaultCadidateItems(db, sync)
	return cache.NewGroupManager(CacheKind, items, stopCh)
}
