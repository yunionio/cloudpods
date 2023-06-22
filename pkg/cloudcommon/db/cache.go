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

package db

import (
	"database/sql"
	"sync"

	"yunion.io/x/pkg/errors"
)

type ICacheable interface {
	GetId() string
}

type SCacheManager[T ICacheable] struct {
	cache   *sync.Map
	manager IStandaloneModelManager
}

func NewCacheManager[T ICacheable](manager IStandaloneModelManager) *SCacheManager[T] {
	return &SCacheManager[T]{
		cache:   &sync.Map{},
		manager: manager,
	}
}

func (cm *SCacheManager[T]) Invalidate() {
	cm.cache = nil
}

func (cm *SCacheManager[T]) FetchById(id string) (*T, error) {
	if cm.cache == nil {
		cm.fetchCacheFromDB()
	}
	m, ok := cm.cache.Load(id)
	if ok {
		return m.(*T), nil
	} else {
		return nil, errors.Wrapf(sql.ErrNoRows, "no such id %s", id)
	}
}

func (cm *SCacheManager[T]) fetchCacheFromDB() error {
	q := cm.manager.Query()
	ret := make([]T, 0)
	err := FetchModelObjects(cm.manager, q, &ret)
	if err != nil {
		return errors.Wrap(err, "FetchModelObjects")
	}
	cache := &sync.Map{}
	for i := range ret {
		cache.Store(ret[i].GetId(), &ret[i])
	}
	cm.cache = cache
	return nil
}

func (cm *SCacheManager[T]) Update(obj *T) {
	cm.cache.Store((*obj).GetId(), obj)
}

func (cm *SCacheManager[T]) Delete(obj *T) {
	cm.cache.Delete((*obj).GetId())
}

func (cm *SCacheManager[T]) Range(proc func(key interface{}, value interface{}) bool) {
	if cm.cache == nil {
		cm.fetchCacheFromDB()
	}
	cm.cache.Range(proc)
}
