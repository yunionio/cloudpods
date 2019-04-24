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

	"yunion.io/x/log"
)

type GroupManager struct {
	Group  map[string]Cache
	stopCh <-chan struct{}
}

func NewGroupManager(kind string, items []CachedItem, ch <-chan struct{}) *GroupManager {
	man := new(GroupManager)
	man.Group = make(map[string]Cache)
	man.stopCh = ch
	for _, item := range items {
		man.Group[item.Name()] = NewCache(kind, item)
	}
	return man
}

func (m *GroupManager) Run() {
	go func() {
		m.run()
		select {}
	}()
}

func (m *GroupManager) run() {
	for name, c := range m.Group {
		log.V(3).Infof("Start cache: %v", name)
		c.Start(m.stopCh)
	}
}

func (m *GroupManager) Get(name string) (Cache, error) {
	entity, ok := m.Group[name]
	if !ok {
		return nil, fmt.Errorf("DB cache item: %s not found", name)
	}
	return entity, nil
}
