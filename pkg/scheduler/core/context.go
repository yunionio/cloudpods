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

package core

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
)

const (
	EmptyCapacity int64 = -1
	MaxCapacity   int64 = 0x7FFFFFFFFFFFFFFF
)

var (
	EmptyCapacities = make(map[string]Counter)
)

type SharedResourceManager struct {
	resourceMap map[string]Counter
	lock        sync.Mutex
}

func NewSharedResourceManager() *SharedResourceManager {
	return &SharedResourceManager{
		lock:        sync.Mutex{},
		resourceMap: make(map[string]Counter),
	}
}

func (m *SharedResourceManager) Add(resourceKey string, capacity Counter) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.resourceMap[resourceKey] = capacity
}

type CounterManager struct {
	Counters map[string]Counter
	lock     sync.Mutex
}

func NewCounterManager() *CounterManager {
	return &CounterManager{
		Counters: make(map[string]Counter),
		lock:     sync.Mutex{},
	}
}

func (m *CounterManager) Get(key string) Counter {
	m.lock.Lock()
	defer m.lock.Unlock()

	if counter, ok := m.Counters[key]; ok {
		return counter
	}

	return nil
}

func (m *CounterManager) GetOrCreate(key string, creator func() Counter) Counter {
	m.lock.Lock()
	defer m.lock.Unlock()

	if counter, ok := m.Counters[key]; ok {
		return counter
	}

	counter := creator()
	if counter == nil {
		return nil
	}

	m.Counters[key] = counter
	return counter
}

type Counter interface {
	GetCount() int64
}

type MultiCounter interface {
	Counter
	Add(counter Counter)
}

type NormalCounter struct {
	Value int64
}

func NewNormalCounter(value int64) *NormalCounter {
	return &NormalCounter{
		Value: value,
	}
}

func (c *NormalCounter) GetCount() int64 {
	return c.Value
}

type Counters struct {
	counters []Counter
	lock     sync.Mutex
	sum      int64
}

func NewCounters() *Counters {
	return &Counters{
		sum:  EmptyCapacity,
		lock: sync.Mutex{},
	}
}

func (c *Counters) Add(cnt Counter) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.counters = append(c.counters, cnt)
	c.sum = EmptyCapacity
}

func (c *Counters) GetCount() int64 {
	if c.sum == EmptyCapacity {
		c.sum = c.calculateCount()
	}
	return c.sum
}

func (c *Counters) calculateCount() int64 {
	if len(c.counters) == 0 {
		return 0
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	value := int64(0)
	for _, c := range c.counters {
		count := c.GetCount()
		if count != EmptyCapacity {
			value += c.GetCount()
		}
	}
	return value
}

type MinCounters struct {
	counters []Counter
}

func NewMinCounters() *MinCounters {
	return &MinCounters{}
}

func (c *MinCounters) Add(counter Counter) {
	c.counters = append(c.counters, counter)
}

func (c *MinCounters) GetCount() int64 {
	if len(c.counters) == 0 {
		return EmptyCapacity
	}
	minCount := c.counters[0].GetCount()
	if len(c.counters) == 1 {
		return minCount
	}
	for _, c0 := range c.counters[1:] {
		count := c0.GetCount()
		if count < minCount {
			minCount = count
		}
	}
	return minCount
}

type Capacity struct {
	Values   map[string]Counter
	MinValue int64
}

type Score struct {
	*score.ScoreBucket
}

func newScore() *Score {
	return &Score{
		ScoreBucket: score.NewScoreBuckets(),
	}
}

func newZeroScore() Score {
	s := newScore()
	s.Append(score.NewZeroScore())
	return *s
}

type SchedContextDataItem struct {
	Networks *sync.Map
	Data     map[string]interface{}
}

type SchedLog struct {
	Candidate string
	Action    string
	Message   string
	IsFailed  bool
}

func NewSchedLog(candidate, action, message string, isFailed bool) SchedLog {
	return SchedLog{candidate, action, message, isFailed}
}

func (log *SchedLog) String() string {
	return fmt.Sprintf("%v [%v] %v", log.Candidate, log.Action, log.Message)
}

type SchedLogList []SchedLog

func (logList SchedLogList) Get(index string) *SchedLog {
	for _, l := range logList {
		if l.Candidate == index {
			return &l
		}
	}
	return nil
}

func (logList SchedLogList) Len() int {
	return len(logList)
}

func (logList SchedLogList) Less(i, j int) bool {
	r := strings.Compare(logList[i].Candidate, logList[j].Candidate)
	if r != 0 {
		return r < 0
	}

	r = strings.Compare(logList[i].Message, logList[j].Message)
	if r != 0 {
		return r < 0
	}

	return strings.Compare(logList[i].Action, logList[j].Action) < 0
}

func (logList SchedLogList) Swap(i, j int) {
	logList[i], logList[j] = logList[j], logList[i]
}

type SchedLogManager struct {
	Logs   SchedLogList
	lock   sync.Mutex
	sorted bool
}

func NewSchedLogManager() *SchedLogManager {
	return &SchedLogManager{
		lock: sync.Mutex{},
		Logs: SchedLogList{},
	}
}

func (m *SchedLogManager) Append(candidate, action, message string, isFailed bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.Logs = append(m.Logs, NewSchedLog(candidate, action, message, isFailed))
}

func (m *SchedLogManager) Appends(logs []SchedLog) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.Logs = append(m.Logs, logs...)
}

func (m *SchedLogManager) FailedLogs() SchedLogList {
	var logs SchedLogList
	for _, l := range m.Logs {
		if l.IsFailed {
			logs = append(logs, l)
		}
	}
	return logs
}

func (m *SchedLogManager) Read() []string {
	rets := []string{}

	m.lock.Lock()
	defer m.lock.Unlock()

	if len(m.Logs) == 0 {
		return rets
	}

	if !m.sorted {
		sort.Sort(m.Logs)
		m.sorted = true
	}

	joinLogs := func(startIndex, endIndex int) string {
		if endIndex == startIndex+1 {
			return m.Logs[startIndex].String()
		}

		log := m.Logs[startIndex]
		actions := []string{}
		var isFailed bool
		for ; startIndex < endIndex; startIndex++ {
			actions = append(actions, m.Logs[startIndex].Action)
			if m.Logs[startIndex].IsFailed {
				isFailed = true
			}
		}

		newLog := NewSchedLog(log.Candidate, strings.Join(actions, ","), log.Message, isFailed)
		return newLog.String()
	}

	startIndex := -1
	for index, len := 0, len(m.Logs); index < len; index++ {
		if startIndex < 0 {
			startIndex = index
		} else {
			log0, log := m.Logs[startIndex], m.Logs[index]
			if log0.Candidate != log.Candidate || log0.Message != log.Message {
				rets = append(rets, joinLogs(startIndex, index))
				startIndex = index
			}
		}
	}

	rets = append(rets, joinLogs(startIndex, len(m.Logs)))
	return rets
}

// Unit wraps sched input info and other log and record manager
type Unit struct {
	SchedInfo             *api.SchedInfo
	CapacityMap           map[string]*Capacity
	ScoreMap              map[string]Score
	DataMap               map[string]*SchedContextDataItem
	SharedResourceManager *SharedResourceManager
	CounterManager        *CounterManager

	capacityLock sync.Mutex
	scoreLock    sync.Mutex

	FailedCandidateMap     map[string]*FailedCandidates
	failedCandidateMapLock sync.Mutex

	//ScoreMap   map[string]Score
	//LogManager *LogManager
	//ReservedPool *data_manager.ReservedPool

	SchedulerManager interface{}

	selectPlugins []SelectPlugin

	LogManager *SchedLogManager

	AllocatedResources map[string]*AllocatedResource
}

func NewScheduleUnit(info *api.SchedInfo, schedManager interface{}) *Unit {
	cmap := make(map[string]*Capacity) // candidate_id, Capacity
	smap := make(map[string]Score)     // candidate_id, Score
	unit := &Unit{
		SchedInfo:              info,
		FailedCandidateMap:     make(map[string]*FailedCandidates),
		failedCandidateMapLock: sync.Mutex{},
		CapacityMap:            cmap,
		ScoreMap:               smap,
		capacityLock:           sync.Mutex{},
		scoreLock:              sync.Mutex{},
		DataMap:                make(map[string]*SchedContextDataItem),
		SharedResourceManager:  NewSharedResourceManager(),
		CounterManager:         NewCounterManager(),
		LogManager:             NewSchedLogManager(),
		SchedulerManager:       schedManager,
		AllocatedResources:     make(map[string]*AllocatedResource),
	}
	return unit
}

func (u *Unit) Info() string {
	return fmt.Sprintf("%#v", u.SchedInfo)
}

func (u *Unit) SessionID() string {
	return u.SchedInfo.SessionId
}

func (u *Unit) SchedData() *api.SchedInfo {
	return u.SchedInfo
}

func (u *Unit) ShouldExecuteSchedtagFilter(hostId string) bool {
	schedData := u.SchedData()
	if !schedData.Backup {
		if len(schedData.PreferHost) != 0 {
			return false
		}
		return true
	}
	for _, preferHost := range []string{schedData.PreferHost, schedData.PreferBackupHost} {
		if preferHost == hostId {
			return false
		}
	}
	return true
}

func (u *Unit) IsPublicCloudProvider() bool {
	return u.SchedData().IsPublicCloudProvider()
}

func (u *Unit) SkipDirtyMarkHost() bool {
	return u.SchedData().SkipDirtyMarkHost()
}

func (u *Unit) AppendFailedCandidates(fcs []FailedCandidate) {
	if len(fcs) == 0 {
		return
	}

	u.failedCandidateMapLock.Lock()
	defer u.failedCandidateMapLock.Unlock()

	for _, fc := range fcs {
		fcs, ok := u.FailedCandidateMap[fc.Stage]
		if !ok {
			fcs = &FailedCandidates{}
			u.FailedCandidateMap[fc.Stage] = fcs
		}
		fcs.Candidates = append(fcs.Candidates, fc)
	}
}

func (u *Unit) AppendSelectPlugin(p SelectPlugin) {
	u.selectPlugins = append(u.selectPlugins, p)
}

func (u *Unit) AllSelectPlugins() []SelectPlugin {
	return u.selectPlugins
}

func (u *Unit) GetCapacity(id string) int64 {
	var (
		capacityObj *Capacity
		ok          bool
	)

	u.capacityLock.Lock()
	defer u.capacityLock.Unlock()

	if capacityObj, ok = u.CapacityMap[id]; !ok {
		return 0
	}

	if capacityObj.MinValue == EmptyCapacity {
		capacity := MaxCapacity
		for _, counter := range capacityObj.Values {
			count := counter.GetCount()
			if capacity > count {
				capacity = count
			}
		}

		capacityObj.MinValue = capacity
	}

	return capacityObj.MinValue
}

func (u *Unit) GetCapacityOfName(id string, name string) int64 {
	u.capacityLock.Lock()
	defer u.capacityLock.Unlock()

	if capacityObj, ok := u.CapacityMap[id]; ok {
		if counter, ok0 := capacityObj.Values[name]; ok0 {
			return counter.GetCount()
		}
	}

	return EmptyCapacity
}

func (u *Unit) GetCapacities(id string) map[string]Counter {
	if capacityObj, ok := u.CapacityMap[id]; ok {
		return capacityObj.Values
	}

	return EmptyCapacities
}

func (u *Unit) SetCapacity(id string, name string, capacity Counter) error {
	u.capacityLock.Lock()
	defer u.capacityLock.Unlock()

	// Capacity must >= 0
	if !validateCapacityInput(capacity) {
		err := fmt.Errorf("Capacity counter invalid: %#v, count: %d", capacity, capacity.GetCount())
		log.Errorf("SetCapacity error: %v", err)
		return err
	}

	log.V(10).Debugf("%q setCapacity id: %s, capacity: %d", name, id, capacity.GetCount())

	var (
		capacityObj *Capacity
		ok          bool
	)

	if capacityObj, ok = u.CapacityMap[id]; !ok {
		capacityObj = &Capacity{Values: make(map[string]Counter), MinValue: EmptyCapacity}
		u.CapacityMap[id] = capacityObj
	}

	capacityObj.Values[name] = capacity
	capacityObj.MinValue = EmptyCapacity

	return nil
}

func validateCapacityInput(c Counter) bool {
	if c != nil && c.GetCount() >= 0 {
		return true
	}
	return false
}

type ScoreValue struct {
	value score.TScore
}

func (u *Unit) setScore(id string, val score.SScore, tofront bool) {
	u.scoreLock.Lock()
	defer u.scoreLock.Unlock()

	var (
		scoreObj Score
		ok       bool
	)

	if scoreObj, ok = u.ScoreMap[id]; !ok {
		scoreObj = *newScore()
		u.ScoreMap[id] = scoreObj
	}

	if tofront {
		scoreObj.AddToFirst(val)
	} else {
		scoreObj.SetScore(val)
	}

	log.V(10).Infof("SetScore: %q -> %s", id, val.String())
}

func (u *Unit) SetScore(id string, val score.SScore) {
	u.setScore(id, val, false)
}

func (u *Unit) SetFrontScore(id string, val score.SScore) {
	u.setScore(id, val, true)
}

func (u *Unit) GetScore(id string) Score {
	var (
		scoreObj Score
		ok       bool
	)

	if scoreObj, ok = u.ScoreMap[id]; !ok {
		return *newScore()
	}
	return scoreObj
}

func (u *Unit) GetScoreDetails(id string) string {
	if score, ok := u.ScoreMap[id]; ok {
		return score.String()
	}

	return "EmptyScore"
}
func (u *Unit) SetFiltedData(id string, name string, data interface{}) error {
	u.scoreLock.Lock()
	defer u.scoreLock.Unlock()

	dataItem, ok := u.DataMap[id]
	if !ok {
		dataItem = &SchedContextDataItem{
			Data: make(map[string]interface{}),
		}
		u.DataMap[id] = dataItem
	}

	if name == "network" {
		dataItem.Networks = data.(*sync.Map)
	} else {
		if m, ok := data.(map[string]interface{}); ok {
			for key, value := range m {
				dataItem.Data[key] = value
			}
		}
	}

	return nil
}

func (u *Unit) GetFiltedData(id string, count int64) map[string]interface{} {
	schedContextData := make(map[string]interface{})
	if data, ok := u.DataMap[id]; ok {
		// deal networks
		networks := make(map[string]int64)
		if data.Networks != nil {
			data.Networks.Range(func(networkID, ipNumber interface{}) bool {
				networkIDString := networkID.(string)
				ipNumberInt64 := ipNumber.(int64)
				if count > ipNumberInt64 {
					networks[networkIDString] = ipNumberInt64
					count = count - ipNumberInt64
					return true
				} else if count <= ipNumberInt64 {
					networks[networkIDString] = count
					count = 0
					return false
				}
				return false
			})
		}

		schedContextData["networks"] = networks

		// others
		for key, value := range data.Data {
			schedContextData[key] = value
		}

		return schedContextData
	}

	return nil
}

func (u *Unit) GetAllocatedResource(candidateId string) *AllocatedResource {
	ret, ok := u.AllocatedResources[candidateId]
	if !ok {
		ret = NewAllocatedResource()
		u.AllocatedResources[candidateId] = ret
	}
	return ret
}
