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
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
)

const (
	EmptyCapacity int64 = -1
	MaxCapacity   int64 = 0x7FFFFFFFFFFFFFFF
)

var (
	EmptyCapacities          = make(map[string]Counter)
	EmptySelectPriorityValue = SSelectPriorityValue(0)
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
	s.SetScore(score.NewZeroScore(), tristate.None)
	return *s
}

type SchedContextDataItem struct {
	Networks *sync.Map
	Data     map[string]interface{}
}

type LogMessage struct {
	Type string
	Info string
}

type LogMessages []*LogMessage

func (ms LogMessages) String() string {
	ss := make([]string, 0)
	for _, s := range ms {
		ss = append(ss, fmt.Sprintf("%s: %s", s.Type, s.Info))
	}
	return strings.Join(ss, ",")
}

type SchedLog struct {
	Candidate string
	Action    string
	Messages  LogMessages
	IsFailed  bool
}

func NewSchedLog(candidate, action string, messages LogMessages, isFailed bool) SchedLog {
	return SchedLog{candidate, action, messages, isFailed}
}

func (log *SchedLog) String() string {
	prefix := "Success"
	if log.IsFailed {
		prefix = "Failed"
	}
	return fmt.Sprintf("%s: %v [%v] %v", prefix, log.Candidate, log.Action, log.Messages.String())
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

	r = strings.Compare(logList[i].Messages.String(), logList[j].Messages.String())
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

/*func (m *SchedLogManager) Append(candidate, action, message string, isFailed bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.Logs = append(m.Logs, NewSchedLog(candidate, action, message, isFailed))
}*/

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

		newLog := NewSchedLog(log.Candidate, strings.Join(actions, ","), log.Messages, isFailed)
		return newLog.String()
	}

	startIndex := -1
	for index, len := 0, len(m.Logs); index < len; index++ {
		if startIndex < 0 {
			startIndex = index
		} else {
			log0, log := m.Logs[startIndex], m.Logs[index]
			if log0.Candidate != log.Candidate || log0.Messages.String() != log.Messages.String() {
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

	SelectPriorityMap        map[string]SSelectPriority
	SelectPriorityUpdaterMap map[string]SSelectPriorityUpdater
	SelectPriorityLock       sync.Mutex
}

func NewScheduleUnit(info *api.SchedInfo, schedManager interface{}) *Unit {
	cmap := make(map[string]*Capacity) // candidate_id, Capacity
	smap := make(map[string]Score)     // candidate_id, Score
	spmap := make(map[string]SSelectPriority)
	spumap := make(map[string]SSelectPriorityUpdater)
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

		SelectPriorityMap:        spmap,
		SelectPriorityUpdaterMap: spumap,
	}
	return unit
}

func (u *Unit) Info() string {
	return u.SchedInfo.JSON(u.SchedInfo).String()
}

func (u *Unit) SessionID() string {
	return u.SchedInfo.SessionId
}

func (u *Unit) SchedData() *api.SchedInfo {
	return u.SchedInfo
}

func (u *Unit) ShouldExecuteSchedtagFilter(hostId string) bool {
	return !utils.IsInStringArray(
		hostId, []string{u.SchedInfo.PreferHost, u.SchedInfo.PreferBackupHost},
	)
}

func (u *Unit) GetHypervisor() string {
	hypervisor := compute.HOSTTYPE_HYPERVISOR[u.SchedData().Hypervisor]
	if hypervisor == "" {
		hypervisor = u.SchedData().Hypervisor
	}
	return hypervisor
}

func (u *Unit) GetHypervisorDriver() models.IGuestDriver {
	return models.GetDriver(u.GetHypervisor())
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

	// Capacity must >= -1
	if !validateCapacityInput(capacity) {
		err := fmt.Errorf("capacity counter %#v invalid %d", capacity, capacity.GetCount())
		log.Errorf("SetCapacity error: %v", err)
		return err
	}

	log.Debugf("%q setCapacity id: %s, capacity: %d", name, id, capacity.GetCount())

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

func (u *Unit) GetSelectPriority(id string) SSelectPriorityValue {
	if sp, ok := u.SelectPriorityMap[id]; ok {
		return sp.Value()
	}
	return EmptySelectPriorityValue
}

func (u *Unit) SetSelectPriorityWithLock(id string, name string, spv SSelectPriorityValue) {
	u.SelectPriorityLock.Lock()
	defer u.SelectPriorityLock.Unlock()

	sp, ok := u.SelectPriorityMap[id]
	if !ok {
		sp = NewSSelctPriority()
		u.SelectPriorityMap[id] = sp
	}

	sp[name] = spv
}

func (u *Unit) UpdateSelectPriority() {
	for hostID, sp := range u.SelectPriorityMap {
		for name, spv := range sp {
			sp[name] = u.SelectPriorityUpdaterMap[name](u, spv, hostID)
		}
	}
}

func (u *Unit) GetMaxSelectPriority() (max SSelectPriorityValue) {
	max = EmptySelectPriorityValue
	for _, sp := range u.SelectPriorityMap {
		val := sp.Value()
		if max.Less(val) {
			max = val
		}
	}
	return
}

func (u *Unit) RegisterSelectPriorityUpdater(name string, f SSelectPriorityUpdater) {
	u.SelectPriorityLock.Lock()
	defer u.SelectPriorityLock.Unlock()

	u.SelectPriorityUpdaterMap[name] = f
}

func validateCapacityInput(c Counter) bool {
	if c != nil && c.GetCount() >= -1 {
		return true
	}
	return false
}

type ScoreValue struct {
	value score.TScore
}

func (u *Unit) setScore(id string, val score.SScore, prefer tristate.TriState) {
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

	scoreObj.ScoreBucket.SetScore(val, prefer)

	log.V(10).Infof("SetScore: %q -> %s", id, val.String())
}

func (u *Unit) SetScore(id string, val score.SScore) {
	u.setScore(id, val, tristate.None)
}

func (u *Unit) SetPreferScore(id string, val score.SScore) {
	u.setScore(id, val, tristate.True)
}

func (u *Unit) SetAvoidScore(id string, val score.SScore) {
	u.setScore(id, val, tristate.False)
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

type SSelectPriority map[string]SSelectPriorityValue

func (s SSelectPriority) Value() (val SSelectPriorityValue) {
	val = EmptySelectPriorityValue
	for _, v := range s {
		if v > val {
			val = v
		}
	}
	return
}

func NewSSelctPriority() SSelectPriority {
	return make(map[string]SSelectPriorityValue)
}

// SSelectPriorityUpdater will call to update the specified host after each round of selection
type SSelectPriorityUpdater func(u *Unit, origin SSelectPriorityValue, hostID string) SSelectPriorityValue

type SSelectPriorityValue int

func (s SSelectPriorityValue) Less(sp SSelectPriorityValue) bool {
	return s < sp
}

func (s SSelectPriorityValue) Sub(sp SSelectPriorityValue) (ret SSelectPriorityValue) {
	ret = s - sp
	if ret.Less(EmptySelectPriorityValue) {
		ret = EmptySelectPriorityValue
	}
	return
}

func (s SSelectPriorityValue) SubOne() SSelectPriorityValue {
	return s.Sub(SSelectPriorityValue(1))
}

func (s SSelectPriorityValue) IsEmpty() bool {
	return s == EmptySelectPriorityValue
}
