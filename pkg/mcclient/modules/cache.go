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

package modules

import (
	"fmt"
	"sync"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

const (
	cacheValidPerviod = 1 * time.Minute
)

type tCachedStatus int

const (
	cacheInit     = tCachedStatus(0)
	cacheFetching = tCachedStatus(1)
	cacheComplete = tCachedStatus(2)
)

type sCachedResource struct {
	key      string
	cachedAt time.Time
	lock     *sync.Cond
	object   jsonutils.JSONObject
	status   tCachedStatus
}

type sCachedResourceManager struct {
	lock          *sync.Mutex
	resourceCache map[string]sCachedResource
}

var cachedResourceManager *sCachedResourceManager

func init() {
	cachedResourceManager = &sCachedResourceManager{
		lock:          &sync.Mutex{},
		resourceCache: make(map[string]sCachedResource),
	}
}

func cacheKey(manager modulebase.Manager, idstr string) string {
	return fmt.Sprintf("%s-%s", manager.KeyString(), idstr)
}

func (cm *sCachedResourceManager) getLocked(key string) *sCachedResource {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	return cm.getUnlocked(key, true)
}

func (cm *sCachedResourceManager) getUnlocked(key string, alloc bool) *sCachedResource {
	_, ok := cm.resourceCache[key]
	if !ok {
		if !alloc {
			return nil
		}
		cm.resourceCache[key] = sCachedResource{
			key:    key,
			lock:   sync.NewCond(&sync.Mutex{}),
			status: cacheInit,
		}
	}
	obj := cm.resourceCache[key]
	return &obj
}

func (cm *sCachedResourceManager) getById(manager modulebase.Manager, session *mcclient.ClientSession, idstr string) (jsonutils.JSONObject, error) {
	key := cacheKey(manager, idstr)
	cacheObj := cm.getLocked(key)

	obj := cacheObj.tryGet()
	if obj != nil {
		return obj, nil
	}

	obj, err := manager.GetById(session, idstr, nil)
	if err != nil {
		cacheObj.notifyFail()
		return nil, err
	}
	cacheObj.notifyComplete(obj)
	return obj, nil
}

func (cr *sCachedResource) isValid() bool {
	now := time.Now()
	return cr.status == cacheComplete && now.Sub(cr.cachedAt) <= cacheValidPerviod && cr.object != nil
}

func (cr *sCachedResource) tryGet() jsonutils.JSONObject {
	cr.lock.L.Lock()
	defer cr.lock.L.Unlock()

	if cr.isValid() {
		return cr.object
	}

	for cr.status == cacheFetching {
		cr.lock.Wait()
	}

	if cr.status == cacheComplete {
		return cr.object
	}

	cr.status = cacheFetching

	return nil
}

func (cr *sCachedResource) notifyFail() {
	cr.lock.L.Lock()
	defer cr.lock.L.Unlock()

	cr.status = cacheInit
	cr.object = nil

	cr.lock.Signal()
}

func (cr *sCachedResource) notifyComplete(obj jsonutils.JSONObject) {
	cr.lock.L.Lock()
	defer cr.lock.L.Unlock()

	cr.status = cacheComplete
	cr.object = obj
	cr.cachedAt = time.Now()

	time.AfterFunc(cacheValidPerviod, func() {
		cachedResourceManager.lock.Lock()
		defer cachedResourceManager.lock.Unlock()

		cacheObj := cachedResourceManager.getUnlocked(cr.key, false)
		if cacheObj == nil {
			return
		}

		cacheObj.lock.L.Lock()
		defer cacheObj.lock.L.Unlock()

		if !cacheObj.isValid() {
			delete(cachedResourceManager.resourceCache, cr.key)
		}

	})

	cr.lock.Signal()
}
