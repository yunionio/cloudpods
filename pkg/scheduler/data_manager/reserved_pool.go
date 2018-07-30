package data_manager

import (
	"sync"
	"time"

	"github.com/yunionio/pkg/util/wait"
)

const (
	SessionExpiredTime = 30 * 60 // Seconds
)

type value_t interface{}

type KeyValue interface {
	Get(key string) interface{}
}

// ReservedItem
type ReservedItem struct {
	CandidateId string
	data        map[string]value_t
	sync.RWMutex
}

func NewReservedItem(candidateID string) *ReservedItem {
	return &ReservedItem{
		CandidateId: candidateID,
		data:        make(map[string]value_t),
	}
}

func (item *ReservedItem) Get(key string, def_value value_t) value_t {
	item.RLock()
	defer item.RUnlock()
	value, ok := item.data[key]
	if ok {
		return value
	}
	return def_value
}

func (item *ReservedItem) Set(key string, value value_t) {
	item.set(key, value)
}

func (item *ReservedItem) set(key string, value value_t) {
	item.Lock()
	item.Unlock()

	item.data[key] = value
}

func (item *ReservedItem) SetAll(values map[string]interface{}) {
	for key, value := range values {
		if value != nil {
			item.set(key, value)
		}
	}
}

func (item *ReservedItem) GetAll() (values map[string]interface{}) {
	item.RLock()
	defer item.RUnlock()
	values = make(map[string]interface{})
	for key, value := range item.data {
		values[key] = value
	}
	return values
}

func (item *ReservedItem) ToDict() map[string]interface{} {
	item.RLock()
	defer item.RUnlock()

	dict := make(map[string]interface{})

	for id, value := range item.data {
		dict[id] = value
	}

	return dict
}

type SessionItem struct {
	Time time.Time
	data map[string]*ReservedItem // candidateID -> ReservedItem
	sync.RWMutex
}

func (si *SessionItem) get(candidateID string) *ReservedItem {
	si.RLock()
	defer si.RUnlock()
	reservedItem := si.data[candidateID] // *ReservedItem
	return reservedItem
}

func (si *SessionItem) set(candidateID string, reservedItem *ReservedItem) {
	si.Lock()
	defer si.Unlock()
	si.data[candidateID] = reservedItem
}

func (si *SessionItem) AllCandidateIDs() []string {
	si.RLock()
	defer si.RUnlock()
	candidateIds := []string{}
	for candidateID := range si.data {
		candidateIds = append(candidateIds, candidateID)
	}
	return candidateIds
}

func NewSessionItem() *SessionItem {
	return &SessionItem{
		Time: time.Now(),
		data: make(map[string]*ReservedItem),
	}
}

func (si *SessionItem) ToDict() map[string]interface{} {
	si.RLock()
	defer si.RUnlock()

	dict := make(map[string]interface{})

	dict["time"] = si.Time.Local().Format("2006-01-02 15:04:05")
	for key, rItem := range si.data {
		dict[key] = rItem.ToDict()
	}

	return dict
}

type CandidateItem struct {
	candidateID string
	data        map[string]*ReservedItem // sessionID -> ReservedItem
	result      *ReservedItem
	dirty       bool
	sync.RWMutex
}

func (ci *CandidateItem) get(sessionID string) *ReservedItem {
	ci.RLock()
	defer ci.RUnlock()
	reservedItem := ci.data[sessionID]
	return reservedItem
}

func (ci *CandidateItem) set(sessionID string, reservedItem *ReservedItem) {
	ci.Lock()
	defer ci.Unlock()
	ci.data[sessionID] = reservedItem
	ci.dirty = true
}

func NewCandidateItem(candidateID string) *CandidateItem {
	return &CandidateItem{
		candidateID: candidateID,
		data:        make(map[string]*ReservedItem),
		dirty:       true,
	}
}

func (ci *CandidateItem) caculate() *ReservedItem {
	ci.RLock()
	defer ci.RUnlock()
	data := make(map[string][]value_t)
	for _, reservedItem := range ci.data {
		for key, value := range reservedItem.data {
			values, _ := data[key]
			data[key] = append(values, value)
		}
	}
	reservedItem := NewReservedItem(ci.candidateID)
	for key, values := range data {
		reservedItem.Set(key, ReservedSum(key, values))
	}
	return reservedItem
}

type ReservedPool struct {
	Name          string
	sessionDict   map[string]*SessionItem
	candidateDict map[string]*CandidateItem
	data          map[string]value_t

	stopCh <-chan struct{}
	sync.RWMutex
}

func NewReservedPool(name string, stopCh <-chan struct{}) *ReservedPool {
	return &ReservedPool{
		Name:          name,
		sessionDict:   make(map[string]*SessionItem),
		candidateDict: make(map[string]*CandidateItem),
		data:          make(map[string]value_t),
		stopCh:        stopCh,
	}
}

func (pool *ReservedPool) Start() {
	go wait.Until(pool.checkSessionExpires, time.Duration(10)*time.Second, pool.stopCh)
}

func (pool *ReservedPool) checkSessionExpires() {
	pool.Lock()
	defer pool.Unlock()
	now := time.Now()
	for sessionID, sessionItem := range pool.sessionDict {
		if now.Sub(sessionItem.Time).Seconds() > SessionExpiredTime {
			pool.removeSession(sessionID)
		}
	}
}

func (pool *ReservedPool) Add(sessionID string, candidateID string,
	reservedItem *ReservedItem) {
	pool.Lock()
	defer pool.Unlock()
	session_item, ok := pool.sessionDict[sessionID]
	if !ok {
		session_item = NewSessionItem()
		pool.sessionDict[sessionID] = session_item
	}
	session_item.set(candidateID, reservedItem)
	candidateItem, ok := pool.candidateDict[candidateID]
	if !ok {
		candidateItem = NewCandidateItem(candidateID)
		pool.candidateDict[candidateID] = candidateItem
	}
	candidateItem.set(sessionID, reservedItem)
}

func (pool *ReservedPool) GetReservedItem(candidateID string) *ReservedItem {
	pool.RLock()
	defer pool.RUnlock()
	candidateItem, ok := pool.candidateDict[candidateID]
	if !ok {
		return nil
	}

	if candidateItem.dirty {
		candidateItem.result = candidateItem.caculate()
		candidateItem.dirty = false
	}
	return candidateItem.result
}

func (pool *ReservedPool) GetSessionItem(sessionID string) *SessionItem {
	pool.RLock()
	defer pool.RUnlock()
	if sessionItem, ok := pool.sessionDict[sessionID]; ok {
		return sessionItem
	}
	return nil
}

func (pool *ReservedPool) RemoveSession(sessionID string) bool {
	pool.Lock()
	defer pool.Unlock()

	return pool.removeSession(sessionID)
}

func (pool *ReservedPool) removeSession(sessionID string) bool {
	if sessionItem, ok := pool.sessionDict[sessionID]; ok {
		delete(pool.sessionDict, sessionID)
		if len(pool.sessionDict) == 0 {
			pool.candidateDict = make(map[string]*CandidateItem)
		} else {
			for _, candidateId := range sessionItem.AllCandidateIDs() {
				if candidateItem, ok := pool.candidateDict[candidateId]; ok {
					if _, ok := candidateItem.data[sessionID]; ok {
						delete(candidateItem.data, sessionID)
					}
				}
			}
		}
		return true
	}
	return false
}

func (pool *ReservedPool) InSession(candidateId string) bool {
	pool.RLock()
	defer pool.RUnlock()
	if candidateItem, ok := pool.candidateDict[candidateId]; ok {
		return len(candidateItem.data) > 0
	}
	return false
}

func (pool *ReservedPool) ToDict() interface{} {
	pool.RLock()
	defer pool.RUnlock()

	data := make(map[string]interface{})

	for sessionId, sessionItem := range pool.sessionDict {
		data[sessionId] = sessionItem.ToDict()
	}

	return data
}
