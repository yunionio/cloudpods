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

package candidate

import (
	"fmt"
	"reflect"
	gosync "sync"
	"time"

	"yunion.io/x/log"
	u "yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/scheduler/cache"
	"yunion.io/x/onecloud/pkg/scheduler/options"
)

const (
	CacheKind = "CandidateCache"

	HostCandidateCache      = "Hosts"
	BaremetalCandidateCache = "Baremetals"

	HostDescBuilder      = HostCandidateCache
	BaremetalDescBuilder = BaremetalCandidateCache
)

func defaultCadidateItems() []cache.CachedItem {
	return []cache.CachedItem{
		newHostCache(),
		newBaremetalCache(),
	}
}

func uuidKey(obj interface{}) (string, error) {
	return obj.(descer).GetId(), nil
}

func generalUpdateFunc(act BuildActor, mutex *gosync.Mutex) cache.UpdateFunc {
	return func(ids []string) ([]interface{}, error) {
		mutex.Lock()
		defer mutex.Unlock()
		newAct := act.Clone()
		builder := NewDescBuilder(newAct)
		descs, err := builder.Build(ids)
		if err != nil {
			return nil, err
		}

		return descs, nil
	}
}

func generalLoadFunc(act BuildActor, mutex *gosync.Mutex) cache.LoadFunc {
	return func() ([]interface{}, error) {
		mutex.Lock()
		defer mutex.Unlock()
		newAct := act.Clone()
		builder := NewDescBuilder(newAct)

		ids, err := act.AllIDs()
		if err != nil {
			return nil, err
		}
		descs, err := builder.Build(ids)
		if err != nil {
			return nil, err
		}
		return descs, nil
	}
}

// generalGetUpdateFunc provides the ability to generate regularly updated data.
func generalGetUpdateFunc(isBaremetal bool) cache.GetUpdateFunc {
	// The purpose of the counter is to update the data in full at regular intervals.
	fullUpdateCounter := 0
	return func(d []interface{}) ([]string, error) {
		// Full update every 10 minutes(30s * 20)
		if isBaremetal && fullUpdateCounter >= options.Options.BaremetalCandidateCacheReloadCount {
			fullUpdateCounter = 1
			log.Infof("FullUpdateCounter: %d, update all baremetals.", fullUpdateCounter)
			return nil, nil
		}

		if !isBaremetal && fullUpdateCounter >= options.Options.HostCandidateCacheReloadCount {
			fullUpdateCounter = 1
			log.Infof("FullUpdateCounter: %d, update all hosts.", fullUpdateCounter)
			return nil, nil
		}

		allStatus := make(map[string]time.Time, len(d))
		// This will reflect the key `ID` and `UpdatedAt`,maybe one day can optimize this part.
		for _, item := range d {
			r := reflect.ValueOf(item)
			f := reflect.Indirect(r)
			key := f.FieldByName("Id")
			if !key.IsValid() {
				key = f.FieldByName("ID")
			}
			value := f.FieldByName("UpdatedAt")
			if key.IsValid() && value.IsValid() {
				allStatus[key.String()] = value.Interface().(time.Time)
			} else {
				log.Errorf("get `ID` and `UpdatedAt` errror in host:%v\n", item)
			}
		}

		fullUpdateCounter++
		modified, err := FetchHostsUpdateStatus(isBaremetal)
		if err != nil {
			return nil, err
		}
		modifiedIds := make([]string, 0, len(modified))
		// Aggregate the updated hosts
		for _, status := range modified {
			// If host does not exist[ok=false] or has updated will be in update list.
			if t, ok := allStatus[status.Id]; !ok || !t.Equal(status.UpdatedAt) {
				modifiedIds = append(modifiedIds, status.Id)
			}
		}
		if len(modifiedIds) == 0 {
			return nil, fmt.Errorf("%s", "no need update all")
		}
		return modifiedIds, nil
	}
}

func newHostCache() cache.CachedItem {
	mutex := new(gosync.Mutex)
	update := generalUpdateFunc(newHostBuilder(), mutex)
	load := generalLoadFunc(newHostBuilder(), mutex)
	getUpdate := generalGetUpdateFunc(false)
	item := new(candidateItem)

	item.CachedItem = cache.NewCacheItem(
		HostCandidateCache,
		u.ToDuration(options.Options.HostCandidateCacheTTL),
		u.ToDuration(options.Options.HostCandidateCachePeriod),
		uuidKey,
		update,
		load,
		getUpdate,
	)
	return item
}

func newBaremetalCache() cache.CachedItem {
	// The mutex solves the possible dirty data asked lead to over-commit.
	mutex := new(gosync.Mutex)
	update := generalUpdateFunc(newBaremetalBuilder(), mutex)
	load := generalLoadFunc(newBaremetalBuilder(), mutex)
	getUpdate := generalGetUpdateFunc(true)
	item := new(candidateItem)

	item.CachedItem = cache.NewCacheItem(
		BaremetalCandidateCache,
		u.ToDuration(options.Options.BaremetalCandidateCacheTTL),
		u.ToDuration(options.Options.BaremetalCandidateCachePeriod),
		uuidKey,
		update,
		load,
		getUpdate,
	)
	return item
}
