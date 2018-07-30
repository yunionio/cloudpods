package cache

import (
	"time"
)

type CacheGroup interface {
	Run()
	Get(string) (Cache, error)
}

type Cache interface {
	Add(obj interface{}) error
	Update(obj interface{}) error
	Delete(obj interface{}) error
	List() []interface{}
	Get(string) (item interface{}, err error)
	Start(<-chan struct{})

	Reload(keys []string) (items []interface{}, err error)
	ReloadAll() (items []interface{}, err error)
	WaitForReady()
}

type CachedItem interface {
	TTL() time.Duration
	Name() string
	Period() time.Duration
	Update(keys []string) ([]interface{}, error)
	Load() ([]interface{}, error)
	Key(obj interface{}) (string, error)
	GetUpdate(d []interface{}) ([]string, error)
}
