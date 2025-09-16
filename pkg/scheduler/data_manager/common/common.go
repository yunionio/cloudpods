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

package common

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/wait"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
)

type IResourceManager[O lockman.ILockedObject] interface {
	GetKeyword() string

	GetRefreshInterval() time.Duration
	GetStore() IResourceStore[O]
	GetResource(id string) (O, bool)

	SyncOnce() error

	Start(ctx context.Context)
}

type IResourceStore[O lockman.ILockedObject] interface {
	GetInformerResourceManager() informer.IResourceManager

	Init() error
	Get(id string) (O, bool)
	GetAll() []O
	GetByPrefix(prefixId string) []O
	Add(obj *jsonutils.JSONDict)
	Update(oldObj, newObj *jsonutils.JSONDict)
	Delete(obj *jsonutils.JSONDict)
}

type CommonResourceManager[O lockman.ILockedObject] struct {
	keyword         string
	refreshInterval time.Duration
	store           IResourceStore[O]
}

func NewCommonResourceManager[O lockman.ILockedObject](
	keyword string,
	refreshInterval time.Duration,
	store IResourceStore[O],
) *CommonResourceManager[O] {
	return &CommonResourceManager[O]{
		keyword:         keyword,
		refreshInterval: refreshInterval,
		store:           store,
	}
}

func (m *CommonResourceManager[O]) Start(ctx context.Context) {
	go func() {
		Start[O](ctx, m)
	}()
}

func (m *CommonResourceManager[O]) GetKeyword() string {
	return m.keyword
}

func (m CommonResourceManager[O]) GetStore() IResourceStore[O] {
	return m.store
}

func (m CommonResourceManager[O]) GetResource(id string) (O, bool) {
	return m.store.Get(id)
}

func (m CommonResourceManager[O]) GetAll() []O {
	return m.store.GetAll()
}

func (m *CommonResourceManager[O]) GetRefreshInterval() time.Duration {
	return m.refreshInterval
}

func (m *CommonResourceManager[O]) SyncOnce() error {
	return m.GetStore().Init()
}

type FGetDBObject func(man db.IModelManager, id string, obj *jsonutils.JSONDict) (db.IModel, error)

type ResourceStore[O lockman.ILockedObject] struct {
	dataMap     *sync.Map
	modelMan    db.IModelManager
	res         informer.IResourceManager
	getId       func(O) string
	getWatchId  func(*jsonutils.JSONDict) string
	getDBObject FGetDBObject
	onAdd       func(obj db.IModel)
	onUpdate    func(oldObj *jsonutils.JSONDict, newObj db.IModel)
	onDelete    func(obj *jsonutils.JSONDict)
}

func NewResourceStore[O lockman.ILockedObject](
	modelMan db.IModelManager,
	res informer.IResourceManager,
) *ResourceStore[O] {
	return newResourceStore[O](modelMan, res, nil, nil, nil)
}

func NewJointResourceStore[O lockman.ILockedObject](
	modelMan db.IModelManager,
	res informer.IResourceManager,
	getId func(O) string,
	getWatchId func(*jsonutils.JSONDict) string,
	getDBObject FGetDBObject,
) *ResourceStore[O] {
	return newResourceStore(modelMan, res, getId, getWatchId, getDBObject)
}

func newResourceStore[O lockman.ILockedObject](
	modelMan db.IModelManager,
	res informer.IResourceManager,
	getId func(O) string,
	getWatchId func(*jsonutils.JSONDict) string,
	getDBObject FGetDBObject,
) *ResourceStore[O] {
	if getId == nil {
		getId = func(o O) string {
			return o.GetId()
		}
	}
	if getWatchId == nil {
		getWatchId = func(o *jsonutils.JSONDict) string {
			id, _ := o.GetString("id")
			return id
		}
	}
	if getDBObject == nil {
		getDBObject = func(man db.IModelManager, id string, o *jsonutils.JSONDict) (db.IModel, error) {
			return man.FetchById(id)
		}
	}
	return &ResourceStore[O]{
		dataMap:     new(sync.Map),
		modelMan:    modelMan,
		res:         res,
		getId:       getId,
		getWatchId:  getWatchId,
		getDBObject: getDBObject,
		onAdd:       nil,
		onUpdate:    nil,
		onDelete:    nil,
	}
}

func (s *ResourceStore[O]) WithOnAdd(onAdd func(db.IModel)) *ResourceStore[O] {
	s.onAdd = onAdd
	return s
}

func (s *ResourceStore[O]) WithOnUpdate(onUpdate func(old *jsonutils.JSONDict, newObj db.IModel)) *ResourceStore[O] {
	s.onUpdate = onUpdate
	return s
}

func (s *ResourceStore[O]) WithOnDelete(onDelete func(*jsonutils.JSONDict)) *ResourceStore[O] {
	s.onDelete = onDelete
	return s
}

func (s *ResourceStore[O]) GetInformerResourceManager() informer.IResourceManager {
	return s.res
}

func (s *ResourceStore[O]) Init() error {
	objs := make([]O, 0)
	q := s.modelMan.Query()
	if err := db.FetchModelObjects(s.modelMan, q, &objs); err != nil {
		return err
	}
	for _, obj := range objs {
		s.dataMap.Store(s.getId(obj), obj)
	}
	return nil
}

func (s *ResourceStore[O]) Get(id string) (O, bool) {
	obj, ok := s.dataMap.Load(id)
	if !ok {
		var ret O
		return ret, false
	}
	return obj.(O), true
}

func (s *ResourceStore[O]) GetByPrefix(prefixId string) []O {
	ret := make([]O, 0)
	s.dataMap.Range(func(key, value any) bool {
		if strings.HasPrefix(key.(string), prefixId) {
			ret = append(ret, value.(O))
		}
		return true
	})
	return ret
}

func (s *ResourceStore[O]) GetAll() []O {
	ret := make([]O, 0)
	s.dataMap.Range(func(key, value any) bool {
		ret = append(ret, value.(O))
		return true
	})
	return ret
}

func (s *ResourceStore[O]) Add(obj *jsonutils.JSONDict) {
	id := s.getWatchId(obj)
	if id != "" {
		dbObj, err := s.getDBObject(s.modelMan, id, obj)
		if err == nil {
			v := reflect.ValueOf(dbObj)
			tmpObj := v.Elem().Interface()
			s.dataMap.Store(id, tmpObj)
			log.Infof("Add %s %s", s.modelMan.Keyword(), obj.String())
			if s.onAdd != nil {
				s.onAdd(dbObj)
			}
		} else {
			log.Errorf("Fetch %s by id %s error when created: %v", s.modelMan.Keyword(), id, err)
		}
	}
}

func (s *ResourceStore[O]) removeIgnoreKeys(obj *jsonutils.JSONDict) *jsonutils.JSONDict {
	// ignore keys updated by cloudaccount
	for _, key := range []string{
		"probe_at",
		"update_version",
		"updated_at",
	} {
		obj.Remove(key)
	}
	return obj
}

func (s *ResourceStore[O]) Update(oldObj, newObj *jsonutils.JSONDict) {
	id := s.getWatchId(newObj)
	oldObj = s.removeIgnoreKeys(oldObj)
	newObj = s.removeIgnoreKeys(newObj)
	isEq := oldObj.String() == newObj.String()
	if id != "" && !isEq {
		dbObj, err := s.modelMan.FetchById(id)
		if err == nil {
			v := reflect.ValueOf(dbObj)
			tmpObj := v.Elem().Interface()
			s.dataMap.Store(id, tmpObj)
			log.Infof("Update %s %s", s.modelMan.Keyword(), newObj.String())
			if s.onUpdate != nil {
				s.onUpdate(oldObj, dbObj)
			}
		} else {
			log.Errorf("Fetch %s by id %s error when updated: %v", s.modelMan.Keyword(), id, err)
		}
	}
}

func (s *ResourceStore[O]) Delete(obj *jsonutils.JSONDict) {
	id := s.getWatchId(obj)
	if id != "" {
		s.dataMap.Delete(id)
		log.Infof("Delete %s %s", s.modelMan.Keyword(), obj.String())
		if s.onDelete != nil {
			s.onDelete(obj)
		}
	}
}

func Start[O lockman.ILockedObject](ctx context.Context, resMan IResourceManager[O]) {
	startWatch(ctx, resMan)
	startSync(resMan)
}

func startWatch[O lockman.ILockedObject](ctx context.Context, resMan IResourceManager[O]) {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	informer.NewWatchManagerBySessionBg(s, func(man *informer.SWatchManager) error {
		res := resMan.GetStore().GetInformerResourceManager()
		if err := man.For(res).AddEventHandler(ctx, newEventHandler(res, resMan)); err != nil {
			return errors.Wrapf(err, "watch resource %s", res.KeyString())
		}
		return nil
	})
}

func startSync[O lockman.ILockedObject](resMan IResourceManager[O]) {
	wait.Forever(func() {
		log.Infof("%s data start sync", resMan.GetKeyword())
		startTime := time.Now()
		if err := syncOnce(resMan); err != nil {
			log.Errorf("%s sync data error: %v", resMan.GetKeyword(), err)
			return
		}
		log.Infof("%s finish sync, elapsed %s", resMan.GetKeyword(), time.Since(startTime))
	}, resMan.GetRefreshInterval())
}

func syncOnce[O lockman.ILockedObject](resMan IResourceManager[O]) error {
	if err := resMan.SyncOnce(); err != nil {
		return errors.Wrapf(err, "sync once of %s", resMan.GetKeyword())
	}
	return nil
}

type eventHandler[O lockman.ILockedObject] struct {
	resMan  informer.IResourceManager
	dataMan IResourceManager[O]
}

func newEventHandler[O lockman.ILockedObject](resMan informer.IResourceManager, dataMan IResourceManager[O]) informer.EventHandler {
	return &eventHandler[O]{
		resMan:  resMan,
		dataMan: dataMan,
	}
}

func (e eventHandler[O]) keyword() string {
	return e.resMan.GetKeyword()
}

func (e eventHandler[O]) store() IResourceStore[O] {
	return e.dataMan.GetStore()
}

func (e eventHandler[O]) OnAdd(obj *jsonutils.JSONDict) {
	log.Debugf("%s [CREATED]: \n%s", e.keyword(), obj.String())
	e.store().Add(obj)
}

func (e eventHandler[O]) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	log.Debugf("%s [UPDATED]: \n[NEW]: %s\n[OLD]: %s", e.keyword(), newObj.String(), oldObj.String())
	e.store().Update(oldObj, newObj)
}

func (e eventHandler[O]) OnDelete(obj *jsonutils.JSONDict) {
	log.Debugf("%s [DELETED]: \n%s", e.keyword(), obj.String())
	e.store().Delete(obj)
}
