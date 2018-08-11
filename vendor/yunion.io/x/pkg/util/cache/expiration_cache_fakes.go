package cache

import (
	"yunion.io/x/pkg/util/clock"
	"yunion.io/x/pkg/util/sets"
)

type fakeThreadSafeMap struct {
	ThreadSafeStore
	deletedKeys chan<- string
}

func (c *fakeThreadSafeMap) Delete(key string) {
	if c.deletedKeys != nil {
		c.ThreadSafeStore.Delete(key)
		c.deletedKeys <- key
	}
}

type FakeExpirationPolicy struct {
	NeverExpire     sets.String
	RetrieveKeyFunc KeyFunc
}

func (p *FakeExpirationPolicy) IsExpired(obj *timestampedEntry) bool {
	key, _ := p.RetrieveKeyFunc(obj)
	return !p.NeverExpire.Has(key)
}

func NewFakeExpirationStore(keyFunc KeyFunc, deletedKeys chan<- string, expirationPolicy ExpirationPolicy, cacheClock clock.Clock) Store {
	cacheStorage := NewThreadSafeStore(Indexers{}, Indices{})
	return &ExpirationCache{
		cacheStorage:     &fakeThreadSafeMap{cacheStorage, deletedKeys},
		keyFunc:          keyFunc,
		clock:            cacheClock,
		expirationPolicy: expirationPolicy,
	}
}
