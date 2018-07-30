package candidate

import (
	"github.com/yunionio/onecloud/pkg/scheduler/cache"
)

type groupCacher interface {
	Get(string) (cache.Cache, error)
}

type DBGroupCacher interface {
	groupCacher
}

type SyncGroupCacher interface {
	groupCacher
}

type descer interface {
	UUID() string
}

type BuildActor interface {
	Clone() BuildActor
	Type() string
	AllIDs() ([]string, error)
	Do(ids []string, db DBGroupCacher, sync SyncGroupCacher) ([]interface{}, error)
}
