package ttlpool

import (
	"fmt"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/cache"
)

type Item interface {
	Index() (string, error)
}

type item struct {
	key string
}

func (i *item) Index() (string, error) {
	return i.key, nil
}

type TTLPool struct {
	pool cache.Store
}

func NewTTLPool(defaultTTL time.Duration) *TTLPool {
	p := &TTLPool{}
	p.pool = cache.NewTTLStore(p.storeKeyFunc, defaultTTL)
	return p
}

func (p *TTLPool) Add(obj interface{}) error {
	return p.pool.Add(obj)
}

func (p *TTLPool) Delete(obj interface{}) error {
	return p.pool.Delete(obj)
}

func (p *TTLPool) DeleteByKey(key string) error {
	return p.pool.Delete(&item{key})
}

func (p *TTLPool) Get(obj interface{}) (interface{}, bool, error) {
	return p.pool.Get(obj)
}

func (p *TTLPool) GetByKey(key string) (interface{}, bool, error) {
	return p.pool.GetByKey(key)
}

func (p *TTLPool) Has(obj interface{}) (bool, error) {
	_, exists, err := p.Get(obj)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (p *TTLPool) HasByKey(key string) (bool, error) {
	_, exists, err := p.GetByKey(key)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (p *TTLPool) List() []interface{} {
	return p.pool.List()
}

func (p *TTLPool) storeKeyFunc(obj interface{}) (string, error) {
	return obj.(Item).Index()
}

type CountPool struct {
	ttlPool  *TTLPool
	countMap *sync.Map
}

func NewCountPool() *CountPool {
	ttlPool := NewTTLPool(0)
	return &CountPool{
		ttlPool:  ttlPool,
		countMap: new(sync.Map),
	}
}

func (p *CountPool) storeKeyFunc(obj interface{}) (string, error) {
	return p.ttlPool.storeKeyFunc(obj)
}

func (p *CountPool) Add(obj interface{}, count uint64) error {
	log.V(10).Debugf("CountPool add obj: %v, count: %d", obj, count)
	if count <= 0 {
		return fmt.Errorf("count must be postive number.")
	}
	key, err := p.storeKeyFunc(obj)
	if err != nil {
		return err
	}
	p.countMap.Store(key, count)
	return p.ttlPool.Add(obj)
}

func (p *CountPool) DeleteByKey(key string) (err error) {
	count, ok := p.countMap.Load(key)
	if !ok {
		err = p.ttlPool.DeleteByKey(key)
		return
	}
	minusCount := count.(uint64) - 1
	p.countMap.Store(key, minusCount)
	log.V(10).Debugf("Delete %q, count => %d", key, minusCount)
	if minusCount <= 0 {
		p.countMap.Delete(key)
		return p.ttlPool.DeleteByKey(key)
	}
	return nil
}

func (p *CountPool) Delete(obj interface{}) (err error) {
	key, err := p.storeKeyFunc(obj)
	if err != nil {
		return
	}
	return p.DeleteByKey(key)
}

func (p *CountPool) GetCount(obj interface{}) (count uint64, ok bool, err error) {
	key, err := p.storeKeyFunc(obj)
	if err != nil {
		return
	}
	count, ok = p.GetCountByKey(key)
	return
}

func (p *CountPool) GetCountByKey(key string) (count uint64, ok bool) {
	countI, ok := p.countMap.Load(key)
	if !ok {
		return
	}
	count = countI.(uint64)
	return
}

func (p *CountPool) GetByKey(key string) (interface{}, bool, error) {
	count, ok := p.GetCountByKey(key)
	if !ok {
		return nil, false, nil
	}
	if count <= 0 {
		return nil, false, nil
	}

	return p.ttlPool.GetByKey(key)
}

func (p *CountPool) Get(obj interface{}) (interface{}, bool, error) {
	key, err := p.storeKeyFunc(obj)
	if err != nil {
		return nil, false, err
	}
	return p.GetByKey(key)
}

func (p *CountPool) Has(obj interface{}) (bool, error) {
	return p.ttlPool.Has(obj)
}

func (p *CountPool) HasByKey(key string) (bool, error) {
	return p.ttlPool.HasByKey(key)
}
