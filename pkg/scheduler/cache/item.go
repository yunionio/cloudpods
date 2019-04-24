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
	"time"

	expirationcache "yunion.io/x/pkg/util/cache"
)

type UpdateFunc func([]string) ([]interface{}, error)
type LoadFunc func() ([]interface{}, error)
type GetUpdateFunc func(d []interface{}) ([]string, error)

// Item implement CachedItem interface
type Item struct {
	// Name of this cache item
	name string
	// Time to live duration
	ttl time.Duration
	// Load all objects period
	period time.Duration
	// Function to get index like id or name of this cache item
	keyFunc expirationcache.KeyFunc
	// Function to update range of cache item by their key
	update UpdateFunc
	// Function to load all cache items
	load LoadFunc
	// Function to get item must be updated
	getUpdate GetUpdateFunc
}

// NewCacheItem new a Item implement CachedItem interface
func NewCacheItem(name string, ttl, period time.Duration,
	keyf expirationcache.KeyFunc,
	update UpdateFunc, load LoadFunc,
	getUpdate GetUpdateFunc,
) CachedItem {
	return &Item{
		name:      name,
		ttl:       ttl,
		period:    period,
		keyFunc:   keyf,
		update:    update,
		load:      load,
		getUpdate: getUpdate,
	}
}

func (h *Item) TTL() time.Duration {
	return h.ttl
}

func (h *Item) Key(obj interface{}) (string, error) {
	return h.keyFunc(obj)
}

func (h *Item) Name() string {
	return h.name
}

func (h *Item) Period() time.Duration {
	return h.period
}

func (h *Item) Update(ids []string) ([]interface{}, error) {
	return h.update(ids)
}

func (h *Item) Load() ([]interface{}, error) {
	return h.load()
}

func (h *Item) GetUpdate(d []interface{}) ([]string, error) {
	return h.getUpdate(d)
}
