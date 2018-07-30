package data_manager

import (
	"fmt"
	"sync"
)

// Reserved pool manager is mainly to cloud resources to do state
// updates, currently divided into host, baemetal there are three
// network resources management.
type ReservedPoolManager struct {
	// store all reserved data
	pools  map[string]*ReservedPool
	stopCh <-chan struct{}
	sync.RWMutex
}

func (pm *ReservedPoolManager) GetPool(name string) (*ReservedPool, error) {
	pm.RLock()
	pool, ok := pm.pools[name]
	if !ok {
		return nil, fmt.Errorf("reserved pool %v not found", name)
	}
	pm.RUnlock()
	return pool, nil
}

func (pm *ReservedPoolManager) addPool(pool *ReservedPool) {
	pm.Lock()
	// add or update
	pm.pools[pool.Name] = pool
	pm.Unlock()

	pool.Start()
}

func NewReservedPoolManager(stopCh <-chan struct{}) *ReservedPoolManager {
	pm := &ReservedPoolManager{
		pools:  make(map[string]*ReservedPool),
		stopCh: stopCh,
	}
	pm.addPool(NewReservedPool("host", stopCh))
	pm.addPool(NewReservedPool("baremetal", stopCh))
	pm.addPool(NewReservedPool("networks", stopCh))
	return pm
}

func (pm *ReservedPoolManager) SearchReservedPoolBySessionID(sessionId string) (
	*ReservedPool, error) {
	for _, pool := range pm.pools {
		if pool.GetSessionItem(sessionId) != nil {
			return pool, nil
		}
	}
	return nil, fmt.Errorf("session id: %v not found", sessionId)
}

func (pm *ReservedPoolManager) InSession(resType string, candidateId string) bool {
	if pool, err := pm.GetPool(resType); err == nil {
		return pool.InSession(candidateId)
	}
	return false
}

func (pm *ReservedPoolManager) RemoveSession(sessionId string) bool {
	for _, pool := range pm.pools {
		if pool.RemoveSession(sessionId) {
			return true
		}
	}
	return false
}

func ReservedSubtract(key string, value value_t, reserved value_t) value_t {
	var al ResAlgorithm = GetResAlgorithm(key)
	if al != nil {
		return al.Subtract(value, reserved)
	}
	return value
}

func ReservedSum(key string, values []value_t) value_t {
	var al ResAlgorithm = GetResAlgorithm(key)
	if al != nil {
		return al.Sum(values)
	}
	return nil
}
