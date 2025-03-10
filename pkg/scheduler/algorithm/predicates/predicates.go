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

package predicates

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/schedtag"
)

// BasePredicate is a default struct for all the predicates that will
// include it and implement it's Name() and PreExecute() methods.
type BasePredicate struct{}

func (b *BasePredicate) Name() string {
	return "base_predicate_should_not_be_called"
}

func (b *BasePredicate) PreExecute(ctx context.Context, unit *core.Unit, candis []core.Candidater) (bool, error) {
	return true, nil
}

func (b *BasePredicate) GetHypervisorDriver(u *core.Unit) models.IGuestDriver {
	hypervisor := u.GetHypervisor()
	driver, _ := models.GetDriver(hypervisor, u.SchedInfo.Provider)
	return driver
}

type PredicateHelper struct {
	predicate      core.FitPredicate
	predicateFails []core.PredicateFailureReason
	capacity       int64
	Unit           *core.Unit
	Candidate      core.Candidater
}

func (h *PredicateHelper) getResult() (bool, []core.PredicateFailureReason, error) {
	if len(h.predicateFails) > 0 {
		return false, h.predicateFails, nil
	}

	if h.capacity == 0 {
		return false, []core.PredicateFailureReason{}, nil
	}

	return true, nil, nil
}

func (h *PredicateHelper) GetResult() (bool, []core.PredicateFailureReason, error) {
	ok, reasons, err := h.getResult()
	if !ok {
		log.Warningf("[Filter Result] candidate: %q, filter: %q, is_ok: %v, reason: %q, error: %v\n",
			h.Candidate.IndexKey(), h.predicate.Name(), ok, getReasonsString(reasons), err)
	}
	return ok, reasons, err
}

func getReasonsString(reasons []core.PredicateFailureReason) string {
	if len(reasons) == 0 {
		return ""
	}

	ss := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		ss = append(ss, reason.GetReason())
	}
	return strings.Join(ss, ", ")
}

func NewPredicateHelper(pre core.FitPredicate, unit *core.Unit, candi core.Candidater) *PredicateHelper {
	h := &PredicateHelper{
		predicate:      pre,
		capacity:       core.EmptyCapacity,
		predicateFails: []core.PredicateFailureReason{},
		Unit:           unit,
		Candidate:      candi,
	}
	return h
}

func (h *PredicateHelper) GetFailedResult(err error) (bool, []core.PredicateFailureReason, error) {
	return false, nil, err
}

func (h *PredicateHelper) AppendPredicateFail(reason core.PredicateFailureReason) {
	h.predicateFails = append(h.predicateFails, reason)
}

type predicateFailure struct {
	err   core.PredicateFailureError
	eType string
}

func (f predicateFailure) GetReason() string {
	return f.err.GetReason()
}

func (f predicateFailure) GetType() string {
	return f.eType
}

func (h *PredicateHelper) AppendPredicateFailMsg(reason string) {
	h.AppendPredicateFailMsgWithType(reason, h.predicate.Name())
}

func (h *PredicateHelper) AppendPredicateFailMsgWithType(reason string, eType string) {
	err := NewUnexceptedResourceError(reason)
	h.AppendPredicateFail(&predicateFailure{err: err, eType: eType})
}

func (h *PredicateHelper) AppendInsufficientResourceError(req, total, free int64) {
	h.AppendPredicateFail(
		&predicateFailure{
			err:   NewInsufficientResourceError(h.Candidate.Getter().Name(), req, total, free),
			eType: h.predicate.Name(),
		})
}

// SetCapacity returns the current resource capacity calculated by a filter.
// And 'capacity' default is -1.
func (h *PredicateHelper) SetCapacity(capacity int64) {
	if capacity < 0 {
		capacity = 0
	}

	h.SetCapacityCounter(core.NewNormalCounter(capacity))
}

func (h *PredicateHelper) SetCapacityCounter(counter core.Counter) {
	capacity := counter.GetCount()
	if capacity < core.EmptyCapacity {
		capacity = core.EmptyCapacity
	}

	h.capacity = capacity
	h.Unit.SetCapacity(h.Candidate.IndexKey(), h.predicate.Name(), counter)
}

func (h *PredicateHelper) SetSelectPriority(sp int) {
	if sp < 0 {
		sp = 0
	}

	h.Unit.SetSelectPriorityWithLock(h.Candidate.IndexKey(), h.predicate.Name(), core.SSelectPriorityValue(sp))
}

func (h *PredicateHelper) Exclude(reason string) {
	h.SetCapacity(0)
	h.AppendPredicateFailMsg(reason)
}

func (h *PredicateHelper) ExcludeByErrors(errs []core.PredicateFailureReason) {
	h.SetCapacity(0)
	for _, err := range errs {
		h.AppendPredicateFail(err)
	}
}

func (h *PredicateHelper) Exclude2(predicateName string, current, expected interface{}) {
	h.Exclude(fmt.Sprintf("%s is '%v', expected '%v'", predicateName, current, expected))
}

// UseReserved check whether the unit can use guest reserved resource
func (h *PredicateHelper) UseReserved() bool {
	usable := false
	data := h.Unit.SchedData()
	isoDevs := data.IsolatedDevices
	if len(isoDevs) > 0 || (data.ChangeConfig && data.HasIsolatedDevice) {
		usable = true
	}
	return usable
}

type PredicatedSchedtagResource struct {
	ISchedtagCandidateResource
	PreferTags []computeapi.SchedtagConfig
	AvoidTags  []computeapi.SchedtagConfig
}

type SchedtagInputResourcesMap map[int][]*PredicatedSchedtagResource

func (m SchedtagInputResourcesMap) getAllTags(isPrefer bool) []computeapi.SchedtagConfig {
	ret := make([]computeapi.SchedtagConfig, 0)
	for _, ss := range m {
		for _, s := range ss {
			var tags []computeapi.SchedtagConfig
			if isPrefer {
				tags = s.PreferTags
			} else {
				tags = s.AvoidTags
			}
			ret = append(ret, tags...)
		}
	}
	return ret
}

func (m SchedtagInputResourcesMap) GetPreferTags() []computeapi.SchedtagConfig {
	return m.getAllTags(true)
}

func (m SchedtagInputResourcesMap) GetAvoidTags() []computeapi.SchedtagConfig {
	return m.getAllTags(false)
}

type CandidateInputResourcesMap struct {
	*sync.Map // map[string]SchedtagInputResourcesMap
}

type ISchedtagCandidateResource interface {
	GetName() string
	GetId() string
	Keyword() string
	// GetSchedtags() []models.SSchedtag
	GetSchedtagJointManager() models.ISchedtagJointManager
	GetDynamicConditionInput() *jsonutils.JSONDict
}

type ISchedtagPredicateInstance interface {
	core.FitPredicate
	OnPriorityEnd(u *core.Unit, c core.Candidater)
	OnSelectEnd(u *core.Unit, c core.Candidater, count int64)

	GetInputs(u *core.Unit) []ISchedtagCustomer
	GetResources(c core.Candidater) []ISchedtagCandidateResource
	IsResourceMatchInput(ctx context.Context, input ISchedtagCustomer, res ISchedtagCandidateResource) bool
	IsResourceFitInput(ctx context.Context, unit *core.Unit, c core.Candidater, res ISchedtagCandidateResource, input ISchedtagCustomer) core.PredicateFailureReason

	DoSelect(c core.Candidater, input ISchedtagCustomer, res []ISchedtagCandidateResource) []ISchedtagCandidateResource
	AddSelectResult(index int, input ISchedtagCustomer, selectRes []ISchedtagCandidateResource, output *core.AllocatedResource)
	GetCandidateResourceSortScore(candidate ISchedtagCandidateResource) int64
}

// Schedtag Description
// require: Must be scheduled to the specified tag resource
// prefer: Priority to the specified resource
// avoid: Try to avoid scheduling to the specified resource
// exclude: Do not allow scheduling on the specified resource
type BaseSchedtagPredicate struct {
	BasePredicate
	plugin.BasePlugin

	CandidateInputResources *CandidateInputResourcesMap

	Hypervisor string
	Provider   string
}

func NewBaseSchedtagPredicate() *BaseSchedtagPredicate {
	return &BaseSchedtagPredicate{
		CandidateInputResources: &CandidateInputResourcesMap{Map: new(sync.Map)}, // make(map[string]SchedtagInputResourcesMap),
	}
}

func (p *PredicatedSchedtagResource) isNoTag() bool {
	return len(p.PreferTags) == 0 && len(p.AvoidTags) == 0
}

func (p *PredicatedSchedtagResource) hasPreferTags() bool {
	return len(p.PreferTags) != 0
}

func (p *PredicatedSchedtagResource) hasAvoidTags() bool {
	return len(p.AvoidTags) != 0
}

type ISchedtagCustomer interface {
	// JSON(interface{}) *jsonutils.JSONDict
	GetDynamicConditionInput() *jsonutils.JSONDict
	Keyword() string
	IsSpecifyResource() bool
	GetSchedtags() []*computeapi.SchedtagConfig
	ResourceKeyword() string
}

type SchedtagResourceW struct {
	candidater ISchedtagCandidateResource
	input      ISchedtagCustomer
}

func (w SchedtagResourceW) GetId() string {
	return w.candidater.GetId()
}

func (w SchedtagResourceW) IndexKey() string {
	return fmt.Sprintf("%s:%s", w.candidater.GetName(), w.candidater.GetId())
}

func (w SchedtagResourceW) ResourceType() string {
	return getSchedtagResourceType(w.candidater)
}

func getSchedtagResourceType(candidater ISchedtagCandidateResource) string {
	return candidater.GetSchedtagJointManager().GetMasterManager().KeywordPlural()
}

func (w SchedtagResourceW) GetSchedtags() []schedtag.ISchedtag {
	return schedtag.GetCandidateSchedtags(w.ResourceType(), w.candidater.GetId())
}

func (w SchedtagResourceW) GetDynamicSchedDesc() *jsonutils.JSONDict {
	ret := jsonutils.NewDict()
	resSchedDesc := w.candidater.GetDynamicConditionInput()
	inputSchedDesc := w.input.GetDynamicConditionInput()
	ret.Add(resSchedDesc, w.candidater.Keyword())
	ret.Add(inputSchedDesc, w.input.Keyword())
	return ret
}

func (p *BaseSchedtagPredicate) GetHypervisorDriver() models.IGuestDriver {
	hypervisor := p.Hypervisor
	if hypervisor == api.HostHypervisorForKvm {
		hypervisor = api.SchedTypeKvm
	}
	driver, _ := models.GetDriver(hypervisor, p.Provider)
	return driver
}

func (p *BaseSchedtagPredicate) check(input ISchedtagCustomer, candidate ISchedtagCandidateResource, u *core.Unit, c core.Candidater, allTags []schedtag.ISchedtag) (*PredicatedSchedtagResource, error) {
	// allTags, err := GetAllSchedtags(getSchedtagResourceType(candidate))
	// sMan, err := schedtag.GetSessionManager(u.SessionID())
	// if err != nil {
	// 	return nil, err
	// }
	tagPredicate := NewSchedtagPredicate(input.GetSchedtags(), allTags)
	res := &PredicatedSchedtagResource{
		ISchedtagCandidateResource: candidate,
	}
	if !input.IsSpecifyResource() {
		if err := tagPredicate.Check(
			SchedtagResourceW{
				candidater: candidate,
				input:      input,
			},
		); err != nil {
			return nil, err
		}
		res.PreferTags = tagPredicate.GetPreferTags()
		res.AvoidTags = tagPredicate.GetAvoidTags()
	}
	return res, nil
}

func (p *BaseSchedtagPredicate) checkResources(input ISchedtagCustomer, ress []ISchedtagCandidateResource, u *core.Unit, c core.Candidater, allTags []schedtag.ISchedtag) ([]*PredicatedSchedtagResource, error) {
	errs := make([]error, len(ress))
	ret := make([]*PredicatedSchedtagResource, len(ress))
	errGrp := errgroup.Group{}
	for i := range ress {
		res := ress[i]
		errGrp.Go(func() error {
			ps, err := p.check(input, res, u, c, allTags)
			if err != nil {
				// append err, resource not suit input customer
				errs[i] = err
			} else {
				ret[i] = ps
			}
			return nil
		})
	}
	if err := errGrp.Wait(); err != nil {
		return nil, fmt.Errorf("errGrp.Wait: %v", err)
	}
	newRet := make([]*PredicatedSchedtagResource, 0)
	newErrs := make([]error, 0)
	for i := range ress {
		if ps := ret[i]; ps != nil {
			newRet = append(newRet, ps)
		} else {
			newErrs = append(newErrs, errs[i])
		}
	}
	if len(newRet) == 0 {
		return nil, errors.NewAggregate(newErrs)
	}
	return newRet, nil
}

func (p *BaseSchedtagPredicate) GetInputResourcesMap(candidateId string) SchedtagInputResourcesMap {
	ret, ok := p.CandidateInputResources.Load(candidateId)
	if !ok {
		ret = make(map[int][]*PredicatedSchedtagResource)
		p.CandidateInputResources.Store(candidateId, ret)
	}
	return ret.(map[int][]*PredicatedSchedtagResource)
}

func (p *BaseSchedtagPredicate) PreExecute(ctx context.Context, sp ISchedtagPredicateInstance, u *core.Unit, cs []core.Candidater) (bool, error) {
	input := sp.GetInputs(u)
	if len(input) == 0 {
		return false, nil
	}

	if u.SchedData().ResetCpuNumaPin {
		return false, nil
	}

	p.Hypervisor = u.GetHypervisor()
	p.Provider = u.SchedInfo.Provider

	// always do select step
	u.AppendSelectPlugin(sp)
	return true, nil
}

func (p *BaseSchedtagPredicate) Execute(
	ctx context.Context,
	sp ISchedtagPredicateInstance,
	u *core.Unit,
	c core.Candidater,
) (bool, []core.PredicateFailureReason, error) {
	//inputTime := time.Now()
	inputs := sp.GetInputs(u)
	resources := sp.GetResources(c)
	//log.Infof("=======%s get input time: %s, inputs: %s", sp.Name(), time.Since(inputTime), jsonutils.Marshal(inputs))

	h := NewPredicateHelper(sp, u, c)

	inputRes := p.GetInputResourcesMap(c.IndexKey())
	filterErrs := make([]core.PredicateFailureReason, 0)
	for idx, input := range inputs {
		fitResources := make([]ISchedtagCandidateResource, 0)
		errs := make([]core.PredicateFailureReason, 0)
		matchedRes := make([]ISchedtagCandidateResource, 0)
		for _, r := range resources {
			if sp.IsResourceMatchInput(ctx, input, r) {
				matchedRes = append(matchedRes, r)
			}
		}
		if len(matchedRes) == 0 {
			errs = append(errs, &FailReason{
				Reason: fmt.Sprintf("Not found matched %s, candidate: %s, %s: %s", input.ResourceKeyword(), c.Getter().Name(), input.Keyword(), input.GetDynamicConditionInput()),
				Type:   fmt.Sprintf("%s_match", input.ResourceKeyword()),
			})
		}
		for _, res := range matchedRes {
			if err := sp.IsResourceFitInput(ctx, u, c, res, input); err == nil {
				fitResources = append(fitResources, res)
			} else {
				errs = append(errs, err)
			}
		}
		if len(fitResources) == 0 {
			h.ExcludeByErrors(errs)
			break
		}
		if len(errs) > 0 {
			filterErrs = append(filterErrs, errs...)
		}

		allTags, err := schedtag.GetAllSchedtags(getSchedtagResourceType(fitResources[0]))
		if err != nil {
			h.Exclude(fmt.Sprintf("get all schedtags"))
			break
		}
		//checkTime := time.Now()
		matchedResources, err := p.checkResources(input, fitResources, u, c, allTags)
		//log.Infof("---%s checkResources time: %s", sp.Name(), time.Since(checkTime))
		if err != nil {
			if len(filterErrs) > 0 {
				h.ExcludeByErrors(filterErrs)
			}
			errMsg := fmt.Sprintf("schedtag: %v", err.Error())
			h.Exclude(errMsg)
		}
		inputRes[idx] = matchedResources
	}

	//log.Infof("=======%s get execute time: %s", sp.Name(), time.Since(inputTime))
	return h.GetResult()
}

func SetCandidateScoreBySchedtag(u *core.Unit, c core.Candidater, aggCountMap map[string]int, prefer bool) {
	stepScore := core.PriorityStep
	doSet := u.SetPreferScore
	if !prefer {
		doSet = u.SetAvoidScore
	}
	for n, count := range aggCountMap {
		doSet(c.IndexKey(), score.NewScore(score.TScore(count*stepScore), n))
	}
}

func (p *BaseSchedtagPredicate) OnPriorityEnd(sp ISchedtagPredicateInstance, u *core.Unit, c core.Candidater) {
	resTags := []schedtag.ISchedtag{}
	// sessionMan, err := schedtag.GetSessionManager(u.SessionID())
	// if err != nil {
	// 	// should not happended
	// 	panic(fmt.Sprintf("GetSessionManager(%q) error: %v", u.SessionID(), err))
	// }
	for _, res := range sp.GetResources(c) {
		resType := getSchedtagResourceType(res)
		tags := schedtag.GetCandidateSchedtags(resType, res.GetId())
		resTags = append(resTags, tags...)
	}

	inputRes := p.GetInputResourcesMap(c.IndexKey())
	avoidTags := inputRes.GetAvoidTags()
	preferTags := inputRes.GetPreferTags()

	avoidCountMap := GetSchedtagCount(avoidTags, resTags, api.AggregateStrategyAvoid)
	preferCountMap := GetSchedtagCount(preferTags, resTags, api.AggregateStrategyPrefer)

	setScore := SetCandidateScoreBySchedtag

	setScore(u, c, preferCountMap, true)
	setScore(u, c, avoidCountMap, false)
}

func (p *BaseSchedtagPredicate) OnSelectEnd(sp ISchedtagPredicateInstance, u *core.Unit, c core.Candidater, count int64) {
	inputRes := p.GetInputResourcesMap(c.IndexKey())
	output := u.GetAllocatedResource(c.IndexKey())
	inputs := sp.GetInputs(u)

	idxKeys := []int{}
	// inputRes is unorder map, sorted it
	for idx := range inputRes {
		idxKeys = append(idxKeys, idx)
	}
	sort.Ints(idxKeys)

	for idx := range idxKeys {
		res := inputRes[idx]
		selRes := p.selectResource(sp, c, inputs[idx], res)
		sortRes := newSortCandidateResource(sp, selRes)
		sort.Sort(sortRes)
		sp.AddSelectResult(idx, inputs[idx], sortRes.res, output)
	}
}

type sortCandidateResource struct {
	predicate ISchedtagPredicateInstance
	res       []ISchedtagCandidateResource
}

func newSortCandidateResource(predicate ISchedtagPredicateInstance, res []ISchedtagCandidateResource) *sortCandidateResource {
	return &sortCandidateResource{
		predicate: predicate,
		res:       res,
	}
}

func (s *sortCandidateResource) Len() int {
	return len(s.res)
}

func (s *sortCandidateResource) DebugString() string {
	var debugStr string
	for _, i := range s.res {
		debugStr = fmt.Sprintf("%s %d", debugStr, s.predicate.GetCandidateResourceSortScore(i))
	}
	return debugStr
}

// desc order
func (s *sortCandidateResource) Less(i, j int) bool {
	res1, res2 := s.res[i], s.res[j]
	v1 := s.predicate.GetCandidateResourceSortScore(res1)
	v2 := s.predicate.GetCandidateResourceSortScore(res2)
	return v1 > v2
}

func (s *sortCandidateResource) Swap(i, j int) {
	s.res[i], s.res[j] = s.res[j], s.res[i]
}

func (p *BaseSchedtagPredicate) selectResource(
	sp ISchedtagPredicateInstance,
	c core.Candidater,
	input ISchedtagCustomer,
	ress []*PredicatedSchedtagResource,
) []ISchedtagCandidateResource {
	preferRes := make([]ISchedtagCandidateResource, 0)
	noTagRes := make([]ISchedtagCandidateResource, 0)
	avoidRes := make([]ISchedtagCandidateResource, 0)
	for _, res := range ress {
		if res.isNoTag() {
			noTagRes = append(noTagRes, res.ISchedtagCandidateResource)
		} else if res.hasPreferTags() {
			preferRes = append(preferRes, res.ISchedtagCandidateResource)
		} else if res.hasAvoidTags() {
			avoidRes = append(avoidRes, res.ISchedtagCandidateResource)
		}
	}
	for _, ress := range [][]ISchedtagCandidateResource{
		preferRes,
		noTagRes,
		avoidRes,
	} {
		if len(ress) == 0 {
			continue
		}
		if ret := sp.DoSelect(c, input, ress); ret != nil {
			return ret
		}
	}
	return nil
}

type iServerBaseSchedtagPredicate interface {
	ISchedtagPredicateInstance
	GetCandidateResource(core.Candidater) ISchedtagCandidateResource
}

type ServerBaseSchedtagPredicate struct {
	*BaseSchedtagPredicate
	filter iServerBaseSchedtagPredicate
}

func NewServerBaseSchedtagPredicate(filter iServerBaseSchedtagPredicate) *ServerBaseSchedtagPredicate {
	return &ServerBaseSchedtagPredicate{
		BaseSchedtagPredicate: NewBaseSchedtagPredicate(),
		filter:                filter,
	}
}

func (p *ServerBaseSchedtagPredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	return p.BaseSchedtagPredicate.PreExecute(ctx, p.filter, u, cs)
}

func (p *ServerBaseSchedtagPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	return p.BaseSchedtagPredicate.Execute(ctx, p.filter, u, c)
}

func (p *ServerBaseSchedtagPredicate) GetResources(c core.Candidater) []ISchedtagCandidateResource {
	res := p.filter.GetCandidateResource(c)
	if res == nil || gotypes.IsNil(res) {
		return nil
	}
	return []ISchedtagCandidateResource{
		res,
	}
}

func (p *ServerBaseSchedtagPredicate) IsResourceMatchInput(ctx context.Context, input ISchedtagCustomer, res ISchedtagCandidateResource) bool {
	return true
}

func (p *ServerBaseSchedtagPredicate) IsResourceFitInput(ctx context.Context, u *core.Unit, c core.Candidater, res ISchedtagCandidateResource, input ISchedtagCustomer) core.PredicateFailureReason {
	return nil
}

func (p *ServerBaseSchedtagPredicate) DoSelect(
	c core.Candidater,
	input ISchedtagCustomer,
	res []ISchedtagCandidateResource,
) []ISchedtagCandidateResource {
	return res
}

func (p *ServerBaseSchedtagPredicate) AddSelectResult(index int, input ISchedtagCustomer, selectRes []ISchedtagCandidateResource, output *core.AllocatedResource) {
	// resource is host, do nothing
}

func (p *ServerBaseSchedtagPredicate) GetCandidateResourceSortScore(selectRes ISchedtagCandidateResource) int64 {
	// TODO
	return 1
}

func (p *ServerBaseSchedtagPredicate) OnPriorityEnd(u *core.Unit, c core.Candidater) {
	p.BaseSchedtagPredicate.OnPriorityEnd(p.filter, u, c)
}

func (p *ServerBaseSchedtagPredicate) OnSelectEnd(u *core.Unit, c core.Candidater, count int64) {
	p.BaseSchedtagPredicate.OnSelectEnd(p.filter, u, c, count)
}
