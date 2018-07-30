package manager

import (
	"container/list"
	"sync"
	"time"

	o "github.com/yunionio/onecloud/cmd/scheduler/options"
	"github.com/yunionio/pkg/util/wait"
	u "github.com/yunionio/pkg/utils"
)

type HistoryItem struct {
	Task *Task
	Time time.Time
}

func NewHistoryItem(task *Task) *HistoryItem {
	return &HistoryItem{
		Task: task,
		Time: time.Now(),
	}
}

func (h *HistoryItem) ToMap() map[string]string {
	ret := make(map[string]string)
	ret["SessionID"] = h.Task.GetSessionID()
	return ret
}

func (h *HistoryItem) IsSuggestion() bool {
	return h.Task.SchedInfo.IsSuggestion
}

type HistoryManager struct {
	capacity          int
	historyMap        map[string]*HistoryItem
	historyList       *list.List
	normalHistoryList *list.List // exclude scheduler-test
	lock              sync.Mutex
	stopCh            <-chan struct{}
}

func NewHistoryManager(stopCh <-chan struct{}) *HistoryManager {
	return &HistoryManager{
		capacity:          o.GetOptions().SchedulerHistoryLimit,
		historyMap:        make(map[string]*HistoryItem),
		historyList:       list.New(),
		normalHistoryList: list.New(),
		lock:              sync.Mutex{},
		stopCh:            stopCh,
	}
}

func (m *HistoryManager) NewHistoryItem(task *Task) *HistoryItem {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, ls := range []*list.List{m.historyList, m.normalHistoryList} {
		for ls.Len() > m.capacity {
			h := ls.Back()
			ls.Remove(h)
		}
	}

	historyItem := NewHistoryItem(task)
	m.historyList.PushFront(historyItem)
	if !historyItem.IsSuggestion() {
		m.normalHistoryList.PushFront(historyItem)
	}
	m.historyMap[task.GetSessionID()] = historyItem

	return historyItem
}

func (m *HistoryManager) cleanHistoryMap() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.historyMap) <= m.capacity {
		return
	}
	oldHistoryMap := m.historyMap
	newHistoryMap := make(map[string]*HistoryItem)
	for _, ls := range []*list.List{m.historyList, m.normalHistoryList} {
		for element := ls.Front(); element != nil; element = element.Next() {
			sessionId := (element.Value.(*HistoryItem)).Task.GetSessionID()
			if h, ok := oldHistoryMap[sessionId]; ok {
				newHistoryMap[sessionId] = h
			}
		}
	}
	oldHistoryMap = nil

	m.historyMap = newHistoryMap
}

func (m *HistoryManager) Run() {
	go wait.Until(m.cleanHistoryMap, u.ToDuration(o.GetOptions().SchedulerHistoryCleanPeriod), m.stopCh)
}

func (m *HistoryManager) GetHistoryList(offset int64, limit int64, all bool) ([]*HistoryItem, int64) {
	m.lock.Lock()
	defer m.lock.Unlock()

	var hList *list.List
	if all {
		hList = m.historyList
	} else {
		hList = m.normalHistoryList
	}

	total := int64(hList.Len())
	historyItems := []*HistoryItem{}
	element := hList.Front()
	for index := int64(0); index < offset; index++ {
		if element != nil {
			element = element.Next()
		} else {
			return historyItems, total
		}
	}

	for index := int64(0); index < limit; index++ {
		if element != nil {
			historyItems = append(historyItems, element.Value.(*HistoryItem))
			element = element.Next()
		} else {
			break
		}
	}

	return historyItems, total
}

func (m *HistoryManager) GetHistory(sessionId string) *HistoryItem {
	m.lock.Lock()
	defer m.lock.Unlock()

	if historyItem, ok := m.historyMap[sessionId]; ok {
		return historyItem
	}

	return nil
}
