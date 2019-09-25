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

package rpc

import (
	"sync"

	"yunion.io/x/onecloud/pkg/notify/rpc/apis"
)

// ServiceMap has a map of string and apis.SendNotification's pointer, and a RWMutex lock protect map.
type ServiceMap struct {
	serviceMap map[string]*apis.SendNotificationClient
	lock       sync.RWMutex
}

func NewServiceMap() *ServiceMap {
	return &ServiceMap{serviceMap: make(map[string]*apis.SendNotificationClient)}
}

func (sm *ServiceMap) Get(serviceName string) (*apis.SendNotificationClient, bool) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	client, ok := sm.serviceMap[serviceName]
	return client, ok
}

func (sm *ServiceMap) Set(service *apis.SendNotificationClient, serviceName string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.serviceMap[serviceName] = service
}

func (sm *ServiceMap) Remove(serviceName string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	service, ok := sm.serviceMap[serviceName]
	if !ok {
		return
	}
	service.Conn.Close()
	delete(sm.serviceMap, serviceName)
}

func (sm *ServiceMap) BatchRemove(serviceNames []string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	for _, serviceName := range serviceNames {
		service, ok := sm.serviceMap[serviceName]
		if !ok {
			continue
		}
		service.Conn.Close()
		delete(sm.serviceMap, serviceName)
	}
}

func (sm *ServiceMap) ServiceNames() []string {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	serviceNames := make([]string, 0, len(sm.serviceMap))
	for serviceName := range sm.serviceMap {
		serviceNames = append(serviceNames, serviceName)
	}
	return serviceNames
}

func (sm *ServiceMap) IsExist(serviceName string) bool {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	_, ok := sm.serviceMap[serviceName]
	return ok
}

func (sm *ServiceMap) Len() int {
	return len(sm.serviceMap)
}

func (sm *ServiceMap) Map(f func(*apis.SendNotificationClient)) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	for _, service := range sm.serviceMap {
		f(service)
	}
}
