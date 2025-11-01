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

package sku

import (
	"context"
	"fmt"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/wait"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

var (
	skuManager *SSkuManager
)

func Start(ctx context.Context, refreshInterval time.Duration) {
	skuManager = &SSkuManager{
		skuMap:          newSkuMap(),
		refreshInterval: refreshInterval,
	}
	skuManager.startWatch(ctx)
	skuManager.sync()
}

func (m *SSkuManager) startWatch(ctx context.Context) {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	informer.NewWatchManagerBySessionBg(s, func(man *informer.SWatchManager) error {
		if err := man.For(compute.ServerSkus).AddEventHandler(ctx, newEventHandler(compute.ServerSkus, m)); err != nil {
			return errors.Wrapf(err, "watch resource %s", compute.ServerSkus.KeyString())
		}
		return nil
	})
}

func newEventHandler(res informer.IResourceManager, dataMan *SSkuManager) informer.EventHandler {
	return &eventHandler{
		res:     res,
		dataMan: dataMan,
	}
}

type eventHandler struct {
	res     informer.IResourceManager
	dataMan *SSkuManager
}

func (e eventHandler) keyword() string {
	return e.res.GetKeyword()
}

func (e eventHandler) newServerSkuFromJson(obj *jsonutils.JSONDict) (*models.SServerSku, error) {
	sku := &models.SServerSku{}
	if err := obj.Unmarshal(sku); err != nil {
		return nil, errors.Wrapf(err, "unmarshal server sku by: %s", obj.String())
	}
	return sku, nil
}

func (e eventHandler) newServerSku(obj *jsonutils.JSONDict) (*models.SServerSku, error) {
	sku, err := e.newServerSkuFromJson(obj)
	if err != nil {
		return nil, errors.Wrap(err, "newServerSkuFromJson")
	}
	if sku.PostpaidStatus == "" {
		obj, err := models.ServerSkuManager.FetchById(sku.Id)
		if err != nil {
			return nil, errors.Wrapf(err, "fetch serversku by id %q", sku.Id)
		}
		sku = obj.(*models.SServerSku)
	}
	return sku, nil
}

func isValidServerSku(sku *models.SServerSku) error {
	if sku.PrepaidStatus != computeapi.SkuStatusAvailable && sku.PostpaidStatus != computeapi.SkuStatusAvailable {
		return errors.Wrapf(errors.ErrInvalidStatus, "sku: %s, prepaid_status: %q, postpaid_status: %q", sku.Name, sku.PrepaidStatus, sku.PostpaidStatus)
	}
	if !sku.Enabled.IsTrue() {
		return errors.Wrapf(errors.ErrInvalidStatus, "sku: %s, enabled: %q", sku.Name, sku.Enabled)
	}
	return nil
}

func newServerSku(sku *models.SServerSku) *ServerSku {
	return &ServerSku{
		Id:       sku.Id,
		Name:     sku.Name,
		RegionId: sku.CloudregionId,
		ZoneId:   sku.ZoneId,
		Provider: sku.Provider,
	}
}

func (e eventHandler) addServerSku(obj *jsonutils.JSONDict) error {
	dbSku, err := e.newServerSku(obj)
	if err != nil {
		return errors.Wrap(err, "new server sku")
	}
	if err := isValidServerSku(dbSku); err != nil {
		return errors.Wrap(err, "invalid server sku")
	}

	sku := newServerSku(dbSku)
	log.Infof("add server sku %s", jsonutils.Marshal(sku).String())
	e.dataMan.skuMap.Add(sku.Name, sku)
	return nil
}

func (e eventHandler) OnAdd(obj *jsonutils.JSONDict) {
	log.Infof("%s [CREATED]: \n%s", e.keyword(), obj.String())
	if err := e.addServerSku(obj); err != nil {
		log.Errorf("add server sku error: %v", err)
		return
	}
}

func (e eventHandler) deleteServerSku(sku *ServerSku) error {
	e.dataMan.skuMap.Delete(sku.Name, sku)
	return nil
}

func (e eventHandler) updateServerSku(newObj *jsonutils.JSONDict) error {
	newSku, err := e.newServerSku(newObj)
	if err != nil {
		return errors.Wrap(err, "new old server sku")
	}
	shouldDelete := false
	err = isValidServerSku(newSku)
	if err != nil {
		shouldDelete = true
	}
	sku := newServerSku(newSku)
	if shouldDelete {
		log.Infof("delete server sku %s when updating, cause of %v", sku.Name, err)
		if err := e.deleteServerSku(sku); err != nil {
			return errors.Wrap(err, "delete server sku")
		}
	} else {
		if err := e.addServerSku(newObj); err != nil {
			return errors.Wrap(err, "add server sku")
		}
	}
	return nil

}

func (e eventHandler) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	log.Infof("%s [UPDATED]: \n[NEW]: %s\n[OLD]: %s", e.keyword(), newObj.String(), oldObj.String())
	if err := e.updateServerSku(newObj); err != nil {
		log.Errorf("update server sku error: %v", err)
	}
}

func (e eventHandler) OnDelete(obj *jsonutils.JSONDict) {
	log.Infof("%s [DELETED]: \n%s", e.keyword(), obj.String())
	sku, err := e.newServerSkuFromJson(obj)
	if err != nil {
		log.Errorf("new server sku error: %v", err)
		return
	}
	if err := e.deleteServerSku(newServerSku(sku)); err != nil {
		log.Errorf("delete server sku error: %v", err)
	}
}

func SyncOnce(wait bool) error {
	if skuManager == nil {
		return fmt.Errorf("sku manager not init")
	}
	if wait {
		skuManager.syncOnce()
	} else {
		go skuManager.syncOnce()
	}
	return nil
}

func GetByZone(instanceType, regionId, zoneId string) *ServerSku {
	return skuManager.GetByZone(instanceType, regionId, zoneId)
}

func GetByRegion(instanceType, regionId string) []*ServerSku {
	return skuManager.GetByRegion(instanceType, regionId)
}

type skuMap struct {
	*sync.Map
}

type ServerSku struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	RegionId string `json:"cloudregion_id"`
	ZoneId   string `json:"zone_id"`
	Provider string `json:"provider"`
}

type skuList []*ServerSku

func (l skuList) Has(newSku *ServerSku) (int, bool) {
	for i, oldSku := range l {
		if oldSku.Id == newSku.Id {
			return i, true
		}
	}
	return -1, false
}

func (l skuList) DebugString() string {
	return jsonutils.Marshal(l).String()
}

func (l skuList) GetByRegion(regionId string) []*ServerSku {
	ret := make([]*ServerSku, 0)
	for idx := range l {
		sku := l[idx]
		if sku.RegionId == regionId {
			ret = append(ret, sku)
		}
	}
	return ret
}

func (l skuList) GetByZone(regionId, zoneId string) *ServerSku {
	for _, s := range l {
		if s.ZoneId == zoneId || (len(s.ZoneId) == 0 && s.RegionId == regionId) {
			return s
		}
	}
	return nil
}

func newSkuMap() *skuMap {
	return &skuMap{
		Map: new(sync.Map),
	}
}

func (cache *skuMap) Get(instanceType string) skuList {
	value, ok := cache.Load(instanceType)
	if ok {
		return value.(skuList)
	}
	return nil
}

func (cache *skuMap) Add(instanceType string, sku *ServerSku) {
	skus := cache.Get(instanceType)
	if skus == nil {
		skus = make([]*ServerSku, 0)
	}
	skus = append(skus, sku)
	cache.Store(instanceType, skuList(skus))
}

func (cache *skuMap) Delete(instanceType string, sku *ServerSku) {
	skus := cache.Get(instanceType)
	if len(skus) == 0 {
		return
	}
	newSkus := make([]*ServerSku, 0)
	for i := range skus {
		curSku := skus[i]
		if curSku.Id == sku.Id {
			log.Infof("delete sku from cache: %s", jsonutils.Marshal(sku))
			continue
		}
		newSkus = append(newSkus, curSku)
	}
	cache.Store(instanceType, skuList(newSkus))
}

type SSkuManager struct {
	// skus cache all server skus in database, key is InstanceType, value is []models.SServerSku
	skuMap          *skuMap
	refreshInterval time.Duration
}

func (m *SSkuManager) syncOnce() {
	log.Infof("SkuManager start sync")
	startTime := time.Now()

	skus := make([]ServerSku, 0)
	q := models.ServerSkuManager.Query("id", "name", "cloudregion_id", "zone_id", "provider").IsTrue("enabled")
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("prepaid_status"), computeapi.SkuStatusAvailable),
			sqlchemy.Equals(q.Field("postpaid_status"), computeapi.SkuStatusAvailable)))
	if err := q.All(&skus); err != nil {
		log.Errorf("SkuManager query all available skus error: %v", err)
		return
	}
	m.skuMap = newSkuMap()
	for _, sku := range skus {
		tmp := sku
		m.skuMap.Add(sku.Name, &tmp)
	}
	log.Infof("SkuManager end sync, consume %s", time.Since(startTime))
}

func (m *SSkuManager) sync() {
	wait.Forever(m.syncOnce, m.refreshInterval)
}

func (m *SSkuManager) GetByZone(instanceType, regionId, zoneId string) *ServerSku {
	l := m.skuMap.Get(instanceType)
	if l == nil {
		return nil
	}
	return l.GetByZone(regionId, zoneId)
}

func (m *SSkuManager) GetByRegion(instanceType, regionId string) []*ServerSku {
	l := m.skuMap.Get(instanceType)
	if l == nil {
		return nil
	}
	return l.GetByRegion(regionId)
}
