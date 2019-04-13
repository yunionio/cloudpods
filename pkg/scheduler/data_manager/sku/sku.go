package sku

import (
	"fmt"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/wait"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/compute/models"
)

var (
	skuManager *SSkuManager
)

func Start(refreshInterval time.Duration) {
	skuManager = &SSkuManager{
		skuMap:          newSkuMap(),
		refreshInterval: refreshInterval,
	}
	skuManager.sync()
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

func GetByZone(instanceType, zoneId string) *ServerSku {
	return skuManager.GetByZone(instanceType, zoneId)
}

type skuMap struct {
	*sync.Map
}

type ServerSku struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	ZoneId string `json:"zone_id"`
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
	return fmt.Sprintf("%s", jsonutils.Marshal(l).String())
}

func (l skuList) GetByZone(zoneId string) *ServerSku {
	for _, s := range l {
		if s.ZoneId == zoneId {
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
	cache.Store(instanceType, skus)
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
	q := models.ServerSkuManager.Query("id", "name", "zone_id")
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(q.Field("prepaid_status"), models.SkuStatusAvailable),
			sqlchemy.Equals(q.Field("postpaid_status"), models.SkuStatusAvailable)))
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

func (m *SSkuManager) GetByZone(instanceType, zoneId string) *ServerSku {
	l := m.skuMap.Get(instanceType)
	if l == nil {
		return nil
	}
	return l.GetByZone(zoneId)
}
