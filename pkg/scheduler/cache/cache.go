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

package cache

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"yunion.io/x/log"
	expirationcache "yunion.io/x/pkg/util/cache"
	"yunion.io/x/pkg/util/wait"
)

var (
	normalError = fmt.Errorf("%s", "no need update all")
)

var (
	// Full update every 10 minutes(30s * 20), but The first implementation subtracts initialization
	fullUpdateHostsCounter      = 0
	fullUpdateBaremetalsCounter = 0
)

func NewCache(kind string, item CachedItem) Cache {
	cache := newSchedulerCache(kind, item)
	return cache
}

type schedulerCache struct {
	kind           string
	item           CachedItem
	cache          expirationcache.Store
	readyCh        chan struct{}
	cacheCandidate sync.Map
}

func newSchedulerCache(
	kind string,
	item CachedItem,
) *schedulerCache {
	return &schedulerCache{
		kind:    kind,
		item:    item,
		cache:   expirationcache.NewTTLStore(item.Key, item.TTL()),
		readyCh: make(chan struct{}),
	}
}

func (c *schedulerCache) Name() string {
	return fmt.Sprintf("%s - %s", c.kind, c.item.Name())
}

func (c *schedulerCache) Get(key string) (interface{}, error) {
	value, ok, err := c.cache.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !ok {
		log.Infof("Update %s, id: %s", c.Name(), key)
		objs, err := c.item.Update([]string{key})
		if err != nil {
			return nil, err
		}

		if len(objs) < 1 {
			return nil, fmt.Errorf("object %v not found", key)
		}

		obj := objs[0]
		err = c.cache.Add(obj)
		if err != nil {
			return nil, err
		}

		return obj, nil
	}

	return value, nil
}

func (c *schedulerCache) Add(obj interface{}) error {
	return c.cache.Add(obj)
}

func (c *schedulerCache) Update(obj interface{}) error {
	return c.Add(obj)
}

func (c *schedulerCache) Delete(obj interface{}) error {
	return c.cache.Delete(obj)
}

func (c *schedulerCache) List() []interface{} {
	return c.cache.List()
}

func (c *schedulerCache) Start(stop <-chan struct{}) {
	f := c.updateAllObjects
	p := c.item.Period()

	go wait.Until(f, p, stop)
}

func (c *schedulerCache) Reload(keys []string) ([]interface{}, error) {
	return c.loadObjects(keys)
}

func (c *schedulerCache) ReloadAll() ([]interface{}, error) {
	return c.loadObjects(nil)
}

func (c *schedulerCache) WaitForReady() {
	readyCh := c.readyCh
	if readyCh != nil {
		<-c.readyCh
	}
}

func (c *schedulerCache) updateAllObjects() {
	defer func() {
		if c.readyCh != nil {
			close(c.readyCh)
			c.readyCh = nil
		}
	}()
	// Get the data you need to update.
	ids, err := c.item.GetUpdate(c.List())
	// if ids is nil and err is nil,than update all.
	if len(ids) == 0 && err == nil {
		c.loadObjects(nil)
	} else if len(ids) == 0 && reflect.DeepEqual(err, normalError) {
		// if ids is nil and err is normalError then return.
		return
	} else if len(ids) > 0 {
		log.V(10).Debugf("Update host/baremetal status list: %v", ids)
		c.loadObjects(ids)
	}
}

func (c *schedulerCache) loadObjects(ids []string) ([]interface{}, error) {
	log.Infof("Start load %s, period: %v, ttl: %v", c.Name(), c.item.Period(), c.item.TTL())
	startTime := time.Now()

	defer func() {
		log.Infof("End load %s, elapsed %s", c.Name(), time.Since(startTime))
	}()

	var (
		objects    []interface{}
		needUpdate map[string]bool
		err        error
	)

	if ids == nil {
		needUpdate = make(map[string]bool, 0)
		c.cacheCandidate.Range(func(key, _ interface{}) bool {
			if key != nil && key.(string) != "" {
				needUpdate[key.(string)] = true
			}

			return true
		})
		objects, err = c.item.Load()
	} else {
		needUpdate = make(map[string]bool, len(ids))
		for _, id := range ids {
			if id != "" {
				needUpdate[id] = true
			}
		}
		objects, err = c.item.Update(ids)
	}
	if err != nil {
		log.Errorf("Load %s: %v", c.Name(), err)
		return nil, err
	}

	log.V(4).Infof("%v objects loaded", len(objects))

	for _, obj := range objects {
		// Add the load new data into cache.
		err := c.Add(obj)
		if err != nil {
			log.Errorf("Add %v object to %s cache: %v", obj, c.Name(), err)
			continue
		}

		if id, err := c.item.Key(obj); err == nil {
			// If exist the id then the id is valid and we set it to false.
			if _, ok := needUpdate[id]; ok {
				needUpdate[id] = false
			}
			// Add or update new data into global cache.
			c.cacheCandidate.Store(id, obj)
		}
	}

	// If status is true,then the host must have been deleted.
	for id, status := range needUpdate {
		if status {
			// Load the need delete object and will delete it from chache and scheduler'cache.
			object, ok := c.cacheCandidate.Load(id)
			if ok {
				c.cacheCandidate.Delete(id)
				c.Delete(object)
			}
		}
	}

	return objects, err
}
