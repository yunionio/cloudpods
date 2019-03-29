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
