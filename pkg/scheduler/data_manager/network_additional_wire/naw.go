package network_additional_wire

import (
	"context"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/wait"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
)

var (
	manager *networkAdditionalWireManager
)

func Start(ctx context.Context, refreshInterval time.Duration) {
	manager = &networkAdditionalWireManager{
		dataMap:         newNawMap(),
		refreshInterval: refreshInterval,
	}
	manager.sync()
}

type nawList []*models.SNetworkAdditionalWire

type nawMap struct {
	*sync.Map
}

func newNawMap() *nawMap {
	return &nawMap{
		Map: new(sync.Map),
	}
}

func (m *nawMap) GetByNetworkId(id string) nawList {
	value, ok := m.Load(id)
	if ok {
		return value.(nawList)
	}
	return nil
}

func (m *nawMap) Add(item *models.SNetworkAdditionalWire) {
	items := m.GetByNetworkId(item.NetworkId)
	if items == nil {
		items = make([]*models.SNetworkAdditionalWire, 0)
	}
	items = append(items, item)
	m.Store(item.NetworkId, items)
}

type networkAdditionalWireManager struct {
	dataMap         *nawMap
	refreshInterval time.Duration
}

func (m *networkAdditionalWireManager) syncOnce() {
	log.Infof("NetworkAdditionalWireManager start sync")
	startTime := time.Now()
	q := models.NetworkAdditionalWireManager.Query()
	ret := make([]models.SNetworkAdditionalWire, 0)
	err := db.FetchModelObjects(models.NetworkAdditionalWireManager, q, &ret)
	if err != nil {
		log.Errorf("NetworkAdditionalWireManager fetch err: %v", err)
		return
	}
	m.dataMap = newNawMap()
	for i := range ret {
		obj := &ret[i]
		m.dataMap.Add(obj)
	}
	log.Infof("NetworkAdditionalWireManager end sync, consume %s", time.Since(startTime))
}

func (m *networkAdditionalWireManager) sync() {
	wait.Forever(m.syncOnce, m.refreshInterval)
}

func (m *networkAdditionalWireManager) GetByNetworkId(networkId string) []*models.SNetworkAdditionalWire {
	list := m.dataMap.GetByNetworkId(networkId)
	return list
}

func FetchNetworkAdditionalWireIds(networkId string) []string {
	naws := manager.GetByNetworkId(networkId)
	wires := make([]string, len(naws))
	for i := range naws {
		wires[i] = naws[i].WireId
	}
	return wires
}
