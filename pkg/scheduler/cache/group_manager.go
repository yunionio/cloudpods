package cache

import (
	"fmt"

	"github.com/yunionio/log"
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
