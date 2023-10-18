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

package schedtag

import (
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type iCache interface {
	get(key string, newFunc func() (interface{}, error)) (interface{}, error)
	rawGet(key string) (interface{}, bool)
	set(key string, obj interface{})
	delete(key string)
}

type cache struct {
	sync.Map

	mutex sync.Mutex
}

func newCache() iCache {
	return &cache{
		Map:   sync.Map{},
		mutex: sync.Mutex{},
	}
}

func (c *cache) get(key string, newFunc func() (interface{}, error)) (interface{}, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	start := time.Now()
	defer func() {
		log.Errorf("+++get key %q elpased: %s", key, time.Since(start))
	}()
	obj, ok := c.Load(key)
	if ok {
		return obj, nil
	}

	obj, err := newFunc()
	if err != nil {
		return nil, errors.Wrapf(err, "new object by key %q", key)
	}
	c.Store(key, obj)
	return obj, nil
}

func (c *cache) rawGet(key string) (interface{}, bool) {
	return c.Map.Load(key)
}

func (c *cache) set(key string, obj interface{}) {
	c.Store(key, obj)
}

func (c *cache) delete(key string) {
	c.Delete(key)
}
