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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/errors"
	gp "yunion.io/x/pkg/util/goroutine_pool"
	utiltrace "yunion.io/x/pkg/util/trace"
	"yunion.io/x/pkg/util/workqueue"

	"yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
)

const (
	NoResourceAvailableMsg = "No resource are avaliable that match all of the following predicates:"
)

// goroutine pool is to solve the problem of go expansion in
// the stack, check goroutine pool status every minute.
var (
	pool = gp.New(60 * time.Second)
)

type FailedPredicateMap map[string][]PredicateFailureReason

type FitError struct {
	Unit               *Unit
	FailedCandidateMap map[string]*FailedCandidates
}

// Error returns detailed information of why the guest failed to fit on each host
func (fe *FitError) Error() string {
	ss := []string{}
	for stage, fcs := range fe.FailedCandidateMap {
		ss = append(ss, fmt.Sprintf("%v(-%v)", stage, len(fcs.Candidates)))
	}
	reasonMsg := fmt.Sprintf("%s filter by %v, session_id=%q", NoResourceAvailableMsg,
		strings.Join(ss, ", "), fe.Unit.SessionID())
	return reasonMsg
}

type NoResourceError struct {
	info      string
	sessionID string
}

func (e *NoResourceError) Error() string {
	return fmt.Sprintf("No resource avaliable to schedule, session_id: %q, info: %q", e.sessionID, e.info)
}

type Scheduler interface {
	BeforePredicate() error
	Predicates() (map[string]FitPredicate, error)
	PriorityConfigs() ([]PriorityConfig, error)

	// mark already selected candidates dirty that
	// can't be use again until cleanup them
	//DirtySelectedCandidates([]*SelectedCandidate)
}

type GenericScheduler struct {
	Scheduler
	predicates map[string]FitPredicate
	priorities []PriorityConfig
}

func NewGenericScheduler(s Scheduler) (*GenericScheduler, error) {
	g := &GenericScheduler{}
	predicates, err := s.Predicates()
	if err != nil {
		return nil, err
	}
	priorities, err := s.PriorityConfigs()
	if err != nil {
		return nil, err
	}
	g.Scheduler = s
	g.predicates = predicates
	g.priorities = priorities
	return g, nil
}

func (g *GenericScheduler) Schedule(unit *Unit, candidates []Candidater) (*SchedResultItemList, error) {
	startTime := time.Now()
	defer func() {
		log.V(4).Infof("Schedule cost time: %v", time.Since(startTime))
	}()

	// get schedule context and information
	schedInfo := unit.SchedInfo
	isSuggestion := schedInfo.IsSuggestion

	// new trace follow all steps
	trace := utiltrace.New(fmt.Sprintf("SessionID: %s, schedule info: %s",
		schedInfo.SessionId, unit.Info()))

	defer trace.LogIfLong(1 * time.Second)
	if len(candidates) == 0 {
		return nil, &NoResourceError{
			sessionID: schedInfo.SessionId,
			info:      unit.Info(),
		}
	}

	// setup something before run predicates, but now there is no actions
	err := g.BeforePredicate()
	if err != nil {
		return nil, err
	}
	trace.Step("Computing predicates")

	// load all predicates and find the candidate can statisfy schedule condition
	filteredCandidates, err := findCandidatesThatFit(unit, candidates, g.predicates)
	if err != nil {
		return nil, err
	}

	// if there is no candidate and not from scheduler/test api will return
	if len(filteredCandidates) == 0 && !isSuggestion {
		return nil, &FitError{
			Unit:               unit,
			FailedCandidateMap: unit.FailedCandidateMap,
		}
	}

	var selectedCandidates []*SelectedCandidate
	if len(filteredCandidates) > 0 {
		trace.Step("Prioritizing")
		// load all priorities and calculate the candidate's score
		priorityList, err := PrioritizeCandidates(unit, filteredCandidates, g.priorities)
		if err != nil {
			return nil, err
		}

		trace.Step("Selecting hosts")
		// select target candate hosts
		selectedCandidates, err = SelectHosts(unit, priorityList)
		if err != nil {
			return nil, err
		}
	} else {
		selectedCandidates = []*SelectedCandidate{}
	}

	resultItems, err := generateScheduleResult(unit, selectedCandidates, candidates)
	if err != nil {
		return nil, err
	}

	return &SchedResultItemList{Unit: unit, Data: resultItems}, nil
}

func newSchedResultByCtx(u *Unit, count int64, c Candidater) *SchedResultItem {
	showDetails := u.SchedInfo.ShowSuggestionDetails
	id := c.IndexKey()
	r := &SchedResultItem{
		ID:                id,
		Count:             count,
		Capacity:          u.GetCapacity(id),
		Name:              c.Getter().Name(),
		Score:             u.GetScore(id).String(),
		Data:              u.GetFiltedData(id, count),
		Candidater:        c,
		AllocatedResource: u.GetAllocatedResource(id),
		SchedData:         u.SchedData(),
	}

	if showDetails {
		r.CapacityDetails = GetCapacities(u, id)
		r.ScoreDetails = u.GetScoreDetails(id)
	}
	return r
}

func generateScheduleResult(u *Unit, scs []*SelectedCandidate, cs []Candidater) (SchedResultItems, error) {
	results := make(SchedResultItems, 0)
	itemMap := make(map[string]int)

	for _, it := range scs {
		cid := it.Candidate.IndexKey()
		r := newSchedResultByCtx(u, it.Count, it.Candidate)
		results = append(results, r)
		itemMap[cid] = 1
	}

	suggestionLimit := u.SchedInfo.SuggestionLimit
	for _, c := range cs {
		if suggestionLimit <= int64(len(results)) {
			break
		}
		id := c.IndexKey()
		if _, ok := itemMap[id]; !ok && u.GetCapacity(id) > 0 {
			itemMap[id] = 1
			r := newSchedResultByCtx(u, 0, c)
			results = append(results, r)
		}
	}

	suggestionAll := u.SchedInfo.SuggestionAll
	if suggestionAll || len(u.SchedData().PreferCandidates) > 0 {
		for _, c := range cs {
			if suggestionLimit <= int64(len(results)) {
				break
			}
			id := c.IndexKey()
			if _, ok := itemMap[id]; !ok {
				itemMap[id] = 0
				r := newSchedResultByCtx(u, 0, c)
				results = append(results, r)
			}
		}
	}

	return results, nil
}

type SchedResultItem struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Count    int64                  `json:"count"`
	Data     map[string]interface{} `json:"data"`
	Capacity int64                  `json:"capacity"`
	Score    string                 `json:"score"`

	CapacityDetails map[string]int64 `json:"capacity_details"`
	ScoreDetails    string           `json:"score_details"`

	Candidater Candidater `json:"-"`

	*AllocatedResource

	SchedData *api.SchedInfo
}

type StorageUsed struct {
	used map[string]int64
}

func NewStorageUsed() *StorageUsed {
	return &StorageUsed{
		used: make(map[string]int64),
	}
}

func (s *StorageUsed) Get(storageId string) int64 {
	if used, ok := s.used[storageId]; ok {
		return used
	}
	return 0
}

func (s *StorageUsed) Add(storageId string, used int64) {
	if s.used == nil {
		s.used = make(map[string]int64)
	}
	oUsed, ok := s.used[storageId]
	if ok {
		s.used[storageId] = oUsed + used
	} else {
		s.used[storageId] = used
	}
}

func (item *SchedResultItem) ToCandidateResource(storageUsed *StorageUsed) *schedapi.CandidateResource {
	return &schedapi.CandidateResource{
		HostId: item.ID,
		Name:   item.Name,
		Disks:  item.getDisks(storageUsed),
		Nets:   item.Nets,
	}
}

func (item *SchedResultItem) getDisks(used *StorageUsed) []*schedapi.CandidateDisk {
	inputs := item.SchedData.Disks
	ret := make([]*schedapi.CandidateDisk, 0)
	for idx, disk := range item.Disks {
		ret = append(ret, &schedapi.CandidateDisk{
			Index:      idx,
			StorageIds: item.getSortStorageIds(used, inputs[idx], disk.Storages),
		})
	}
	return ret
}

type sortStorage struct {
	Id      string
	FeeSize int64
}

type sortStorages []sortStorage

func (s sortStorages) Len() int {
	return len(s)
}

func (s sortStorages) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortStorages) Less(i, j int) bool {
	s1 := s[i]
	s2 := s[j]
	return s1.FeeSize > s2.FeeSize
}

func (s sortStorages) getIds() []string {
	ret := make([]string, 0)
	for _, obj := range s {
		ret = append(ret, obj.Id)
	}
	return ret
}

func (item *SchedResultItem) getSortStorageIds(
	used *StorageUsed,
	disk *compute.DiskConfig,
	storages []*schedapi.CandidateStorage) []string {
	reqSize := disk.SizeMb
	ss := make([]sortStorage, 0)
	for _, s := range storages {
		ss = append(ss, sortStorage{
			Id:      s.Id,
			FeeSize: s.FreeCapacity - used.Get(s.Id),
		})
	}
	toSort := sortStorages(ss)
	sort.Sort(toSort)
	sortedStorages := toSort.getIds()
	ret := make([]string, 0)
	for idx, id := range sortedStorages {
		if idx == 0 {
			used.Add(id, int64(reqSize))
		}
		ret = append(ret, id)
	}
	return ret
}

func GetCapacities(u *Unit, id string) (res map[string]int64) {
	res = make(map[string]int64)
	capacities := u.GetCapacities(id)
	if len(capacities) > 0 {
		for name, capacity := range capacities {
			res[name] = capacity.GetCount()
		}
	}
	return
}

type SchedResultItems []*SchedResultItem

func (its SchedResultItems) Len() int {
	return len(its)
}

func (its SchedResultItems) Swap(i, j int) {
	its[i], its[j] = its[j], its[i]
}

func (its SchedResultItems) Less(i, j int) bool {
	it1, it2 := its[i], its[j]
	return it1.Capacity < it2.Capacity
	/*
		ctx := its.Unit

		m := func(c int64) int64 {
			if c > 0 {
				return 1
			}
			return 0
		}

		v := func(count, capacity, score int64) int64 {
			return (m(count) << 42) | (m(capacity) << 21) | score
		}

		count1, count2 := it1.Count, it2.Count
		capacity1, capacity2 := ctx.GetCapacity(it1.ID), ctx.GetCapacity(it2.ID)
		score1, score2 := int64(ctx.GetScore(it1.ID)), int64(ctx.GetScore(it2.ID))

		return v(count1, capacity1, score1) < v(count2, capacity2, score2)
	*/
}

type SchedResultItemList struct {
	Unit *Unit
	Data SchedResultItems
}

func (its SchedResultItemList) String() string {
	bytes, _ := json.Marshal(its.Data)
	return string(bytes)
}

type SelectedCandidate struct {
	Count     int64
	Candidate Candidater
}

func (s SelectedCandidate) Index() (string, error) {
	return s.Candidate.IndexKey(), nil
}

func (s SelectedCandidate) GetCount() uint64 {
	return uint64(s.Count)
}

// SelectHosts takes a prioritized list of candidates and then picks
// a group of hosts
func SelectHosts(unit *Unit, priorityList HostPriorityList) ([]*SelectedCandidate, error) {
	if len(priorityList) == 0 {
		return nil, fmt.Errorf("SelectHosts get empty priorityList.")
	}

	selectedMap := make(map[string]*SelectedCandidate)
	schedData := unit.SchedData()
	count := schedData.Count
	isSuggestion := unit.SchedInfo.IsSuggestion
	bestEffort := unit.SchedInfo.BestEffort
	selectedCandidates := []*SelectedCandidate{}

	plugins := unit.AllSelectPlugins()

	sort.Sort(sort.Reverse(priorityList))

completed:
	for len(priorityList) > 0 {
		log.V(10).Debugf("PriorityList: %#v", priorityList)
		currentPriority := unit.GetMaxSelectPriority()
		priorityList0 := HostPriorityList{}
		for _, it := range priorityList {
			if count <= 0 {
				break completed
			}
			hostID := it.Host
			if !currentPriority.IsEmpty() && unit.GetSelectPriority(hostID).Less(currentPriority) {
				priorityList0 = append(priorityList0, it)
				continue
			}
			var (
				selectedItem *SelectedCandidate
				ok           bool
			)
			if selectedItem, ok = selectedMap[hostID]; !ok {
				selectedItem = &SelectedCandidate{
					Count:     0,
					Candidate: it.Candidate,
				}
				selectedMap[hostID] = selectedItem
			}
			selectedItem.Count++
			count--
			// if capacity of the host large than selected count, this host can be added to priorityList.
			if unit.GetCapacity(hostID) > selectedItem.Count {
				priorityList0 = append(priorityList0, it)
			}
		}
		if !currentPriority.IsEmpty() {
			unit.UpdateSelectPriority()
		}
		// sort by score
		priorityList = priorityList0
		//sort.Sort(sort.Reverse(priorityList))
	}

	for _, sc := range selectedMap {
		for _, plugin := range plugins {
			plugin.OnSelectEnd(unit, sc.Candidate, sc.Count)
		}
		selectedCandidates = append(selectedCandidates, sc)
	}

	if !isSuggestion && !bestEffort {
		if count > 0 {
			return nil, fmt.Errorf("No enough resource, request/capacity: %d/%d", schedData.Count, schedData.Count-count)
		}
	}

	return selectedCandidates, nil
}

func findCandidatesThatFit(unit *Unit, candidates []Candidater, predicates map[string]FitPredicate) ([]Candidater, error) {
	var filtered []Candidater

	newPredicates, err := preExecPredicate(unit, candidates, predicates)
	if err != nil {
		return nil, err
	}

	// sort predicates by their name
	predicateNames := make([]string, 0, len(newPredicates))
	for name := range newPredicates {
		predicateNames = append(predicateNames, name)
	}
	sort.Strings(predicateNames)
	predicateArray := make([]FitPredicate, 0, len(predicateNames))
	for _, name := range predicateNames {
		predicateArray = append(predicateArray, newPredicates[name])
	}

	// do predicate filter
	if len(predicateArray) == 0 {
		filtered = candidates
	} else {
		// Create predicate list with enough space to avoid growing it
		// and allow assigning.
		filtered = make([]Candidater, len(candidates))
		errsChannel := make(chan error, len(candidates))
		var filteredLen int32
		checkUnit := func(i int) {
			fits, fcs, err := unitFitsOnCandidate(
				unit, candidates[i], predicateArray)
			if err != nil {
				errsChannel <- err
				return
			}
			if fits {
				filtered[atomic.AddInt32(&filteredLen, 1)-1] = candidates[i]
			} else {
				unit.AppendFailedCandidates(fcs)
			}
		}
		workqueue.Parallelize(o.GetOptions().PredicateParallelizeSize, len(candidates), checkUnit)
		filtered = filtered[:filteredLen]
		if len(errsChannel) > 0 {
			errs := make([]error, 0)
			length := len(errsChannel)
			for ; length > 0; length-- {
				errs = append(errs, <-errsChannel)
			}
			return []Candidater{}, errors.NewAggregate(errs)
		}
	}
	return filtered, nil
}

func preExecPredicate(unit *Unit, candidates []Candidater, predicates map[string]FitPredicate) (map[string]FitPredicate, error) {
	newPredicateFuncs := map[string]FitPredicate{}
	for name, predicate := range predicates {
		// generate new FitPredicates because of race condition?
		newPredicate := predicate.Clone()
		ok, err := newPredicate.PreExecute(unit, candidates)
		if err != nil {
			return nil, err
		}
		if ok {
			newPredicateFuncs[name] = newPredicate
		}
	}
	return newPredicateFuncs, nil
}

type WaitGroupWrapper struct {
	sync.WaitGroup
}

func (w *WaitGroupWrapper) Wrap(cb func()) {
	w.Add(1)
	pool.Go(func() {
		cb()
		w.Done()
	})
}

func unitFitsOnCandidate(
	unit *Unit,
	candidate Candidater,
	predicates []FitPredicate,
) (bool, []FailedCandidate, error) {
	var (
		fit     bool
		reasons []PredicateFailureReason
		err     error
		fcs     []FailedCandidate
		logs    []SchedLog
	)

	isFit := true
	defer func() {
		if len(logs) > 0 {
			unit.LogManager.Appends(logs)
		}
	}()

	toLog := func(fit bool, reasons []PredicateFailureReason,
		err error, stage string) SchedLog {
		var (
			//sFit     string
			messages = make([]*LogMessage, 0)
		)
		/*if fit {
			sFit = "Success."
		} else {
			sFit = "Failed:"
		}*/
		if err != nil {
			messages = append(messages, &LogMessage{Type: "error", Info: fmt.Sprintf("%v", err)})
		} else {
			if len(reasons) != 0 {
				for _, reason := range reasons {
					messages = append(messages, &LogMessage{Type: reason.GetType(), Info: reason.GetReason()})
				}
			}
		}

		candidateLogIndex := fmt.Sprintf("%v:%s", candidate.Getter().Name(), candidate.IndexKey())

		return NewSchedLog(candidateLogIndex, stage, messages, !fit)
	}

	for _, predicate := range predicates {
		fit, reasons, err = predicate.Execute(unit, candidate)
		logs = append(logs, toLog(fit, reasons, err, predicate.Name()))
		if err != nil {
			return false, nil, err
		}
		if !fit {
			fcs = append(fcs, FailedCandidate{
				Stage:     predicate.Name(),
				Candidate: candidate,
				Reasons:   reasons,
			})
			isFit = false
			// When AlwaysCheckAllPredicates is set to true, scheduler checks all
			// the configured predicates even after one or more of them fails.
			// When the flag is set to false, scheduler skips checking the rest
			// of the predicates after it finds one predicate that failed.
			if !o.GetOptions().AlwaysCheckAllPredicates {
				break
			}
		}
	}
	return isFit, fcs, nil
}

// PrioritizeCandidates by running the individual priority functions in parallel.
// Each priority function is expected to set a score of 0-10
// 0 is the lowest priority score (least preffered node) and 10 is the highest
/// Each priority function can also have its own weight
// The resource scores returned by priority function are multiplied by the weights to get weighted scores
// All scores are finally combined (added) to get the total weighted scores of all resources
func PrioritizeCandidates(
	unit *Unit,
	candidates []Candidater,
	priorities []PriorityConfig,
) (HostPriorityList, error) {
	// If no priority configs are provided, then the EqualPriority function is applied
	// This is required to generate the priority list in the required format
	if len(priorities) == 0 {
		result := make(HostPriorityList, 0, len(candidates))
		for _, candidate := range candidates {
			hostPriority, err := EqualPriority(unit, candidate)
			if err != nil {
				return nil, err
			}
			result = append(result, hostPriority)
		}
		return result, nil
	}

	wg := sync.WaitGroup{}
	results := make([]HostPriorityList, len(priorities))
	newPriorities, err := preExecPriorities(priorities, unit, candidates)
	if err != nil {
		return nil, err
	}
	// Max : 3 * len(newPriorities)
	errsChannel := make(chan error, 3*len(newPriorities))
	for i := range newPriorities {
		results[i] = make(HostPriorityList, len(candidates))
	}

	processCandidate := func(index int) {
		var err error
		candidate := candidates[index]
		for i := range newPriorities {
			results[i][index], err = newPriorities[i].Map(unit, candidate)
			if err != nil {
				errsChannel <- err
				return
			}
		}
	}
	workqueue.Parallelize(o.GetOptions().PriorityParallelizeSize, len(candidates), processCandidate)

	for i, p := range newPriorities {
		wg.Add(1)
		go func(index int, priority PriorityConfig) {
			defer wg.Done()
			if err := priority.Reduce(unit, candidates, results[index]); err != nil {
				errsChannel <- err
			}
		}(i, p)
	}
	// Wait for all computations to be finished.
	wg.Wait()
	if len(errsChannel) != 0 {
		errs := make([]error, 0)
		length := len(errsChannel)
		for ; length > 0; length-- {
			errs = append(errs, <-errsChannel)
		}
		return HostPriorityList{}, errors.NewAggregate(errs)
	}

	// Summarize all scores
	result := make(HostPriorityList, 0, len(candidates))
	// TODO: Consider parallelizing it

	// Do plugin priorities step
	for _, candidate := range candidates {
		for _, plugin := range unit.AllSelectPlugins() {
			plugin.OnPriorityEnd(unit, candidate)
		}
	}

	for i, candidate := range candidates {
		result = append(result, HostPriority{Host: candidates[i].IndexKey(), Score: *newScore(), Candidate: candidates[i]})
		//for j := range newPriorities {
		//result[i].Score += results[j][i].Score * newPriorities[j].Weight
		//}
		result[i].Score = unit.GetScore(candidate.IndexKey())
	}
	if log.V(10) {
		for i := range result {
			log.Infof("Host %s => Score %s", result[i].Host, result[i].Score.String())
		}
	}
	return result, nil
}

func preExecPriorities(priorities []PriorityConfig, unit *Unit, candidates []Candidater) ([]PriorityConfig, error) {
	newPriorities := []PriorityConfig{}
	for _, p := range priorities {
		ok, _, err := p.Pre(unit, candidates)
		if err != nil {
			return nil, err
		}
		if ok {
			newPriorities = append(newPriorities, p)
		}
	}
	return newPriorities, nil
}

// EqualPriority is a prioritizer function that gives an equal weight of one to all candidates
func EqualPriority(_ *Unit, candidate Candidater) (HostPriority, error) {
	indexKey := candidate.IndexKey()
	if indexKey == "" {
		return HostPriority{}, fmt.Errorf("Candidate indexKey is empty")
	}
	return HostPriority{
		Host:      indexKey,
		Score:     newZeroScore(),
		Candidate: candidate,
	}, nil
}
