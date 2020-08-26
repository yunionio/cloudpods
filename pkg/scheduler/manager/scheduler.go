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

package manager

import (
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager"
	"yunion.io/x/onecloud/pkg/scheduler/factory"
)

type CandidatesProvider interface {
	ProviderType() string
	CandidateType() string
	Candidates() ([]core.Candidater, error)
	CandidateManager() *data_manager.CandidateManager
}

func candidatesByProvider(provider CandidatesProvider, schedData *api.SchedInfo) ([]core.Candidater, error) {
	var hosts []core.Candidater
	var err error

	candidateManager := provider.CandidateManager()
	if len(schedData.PreferCandidates) >= schedData.RequiredCandidates {
		hosts, err = candidateManager.GetCandidatesByIds(provider.CandidateType(), schedData.PreferCandidates)
	} else {
		args := data_manager.CandidateGetArgs{
			ResType:   provider.CandidateType(),
			ZoneID:    schedData.PreferZone,
			RegionID:  schedData.PreferRegion,
			ManagerID: schedData.PreferManager,
			HostTypes: schedData.GetCandidateHostTypes(),
		}
		hosts, err = candidateManager.GetCandidates(args)
	}
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

type BaseCandidateProvider struct {
	scheduler Scheduler
}

func (b *BaseCandidateProvider) CandidateManager() *data_manager.CandidateManager {
	return b.scheduler.CandidateManager()
}

type HostCandidatesProvider struct {
	*BaseCandidateProvider
}

func NewHostCandidatesProvider(s Scheduler) *HostCandidatesProvider {
	return &HostCandidatesProvider{
		BaseCandidateProvider: &BaseCandidateProvider{scheduler: s},
	}
}

func (h *HostCandidatesProvider) ProviderType() string {
	return factory.DefaultProvider
}

func (h *HostCandidatesProvider) CandidateType() string {
	return api.HostTypeHost
}

func (h *HostCandidatesProvider) Candidates() ([]core.Candidater, error) {
	return candidatesByProvider(h, h.scheduler.SchedData())
}

type BaremetalCandidatesProvider struct {
	*BaseCandidateProvider
}

func NewBaremetalCandidatesProvider(s Scheduler) *BaremetalCandidatesProvider {
	return &BaremetalCandidatesProvider{
		BaseCandidateProvider: &BaseCandidateProvider{scheduler: s},
	}
}

func (b *BaremetalCandidatesProvider) ProviderType() string {
	return factory.BaremetalProvider
}

func (b *BaremetalCandidatesProvider) CandidateType() string {
	return api.SchedTypeBaremetal
}

func (b *BaremetalCandidatesProvider) Candidates() ([]core.Candidater, error) {
	return candidatesByProvider(b, b.scheduler.SchedData())
}

type Scheduler interface {
	SchedData() *api.SchedInfo
	CandidateManager() *data_manager.CandidateManager

	// Schedule process
	BeforePredicate() error
	Predicates() (map[string]core.FitPredicate, error)
	PriorityConfigs() ([]core.PriorityConfig, error)

	// Schedule input get function
	Unit() *core.Unit
	Candidates() ([]core.Candidater, error)

	//DirtySelectedCandidates([]*core.SelectedCandidate)
}

type BaseScheduler struct {
	schedManager *SchedulerManager
	schedInfo    *api.SchedInfo
}

func newBaseScheduler(manager *SchedulerManager, info *api.SchedInfo) (*BaseScheduler, error) {
	s := &BaseScheduler{
		schedManager: manager,
		schedInfo:    info,
	}
	return s, nil
}

func (s *BaseScheduler) NewSchedUnit() *core.Unit {
	return core.NewScheduleUnit(s.schedInfo, s.schedManager)
}

func (s *BaseScheduler) CandidateManager() *data_manager.CandidateManager {
	return s.schedManager.CandidateManager
}

func (s *BaseScheduler) SchedData() *api.SchedInfo {
	return s.schedInfo
}

func (s *BaseScheduler) Unit() *core.Unit {
	return s.NewSchedUnit()
}

func (s *BaseScheduler) BeforePredicate() error {
	return nil
}

// GuestScheduler for guest type schedule
type GuestScheduler struct {
	*BaseScheduler
	algorithmProvider  *factory.AlgorithmProviderConfig
	candidatesProvider *HostCandidatesProvider
}

func newGuestScheduler(manager *SchedulerManager, info *api.SchedInfo) (*GuestScheduler, error) {
	bs, err := newBaseScheduler(manager, info)
	if err != nil {
		return nil, err
	}

	algorithmProvider, err := factory.GetAlgorithmProvider(factory.DefaultProvider)
	if err != nil {
		return nil, err
	}

	gs := &GuestScheduler{
		BaseScheduler:     bs,
		algorithmProvider: algorithmProvider,
	}
	candidatesProvider := NewHostCandidatesProvider(gs)
	gs.candidatesProvider = candidatesProvider

	return gs, nil
}

func (gs *GuestScheduler) Candidates() ([]core.Candidater, error) {
	return gs.candidatesProvider.Candidates()
}

func (gs *GuestScheduler) Predicates() (map[string]core.FitPredicate, error) {
	return factory.GetPredicates(gs.algorithmProvider.FitPredicateKeys)
}

func (gs *GuestScheduler) PriorityConfigs() ([]core.PriorityConfig, error) {
	return factory.GetPriorityConfigs(gs.algorithmProvider.PriorityKeys)
}

// BaremetalScheduler for baremetal type schedule
type BaremetalScheduler struct {
	*BaseScheduler
	algorithmProvider  *factory.AlgorithmProviderConfig
	candidatesProvider *BaremetalCandidatesProvider
}

func newBaremetalScheduler(manager *SchedulerManager, info *api.SchedInfo) (*BaremetalScheduler, error) {
	bs, err := newBaseScheduler(manager, info)
	if err != nil {
		return nil, err
	}

	algorithmProvider, err := factory.GetAlgorithmProvider(factory.BaremetalProvider)
	if err != nil {
		return nil, err
	}

	bms := &BaremetalScheduler{
		BaseScheduler:     bs,
		algorithmProvider: algorithmProvider,
	}

	cp := NewBaremetalCandidatesProvider(bms)
	bms.candidatesProvider = cp

	return bms, nil
}

func (bs *BaremetalScheduler) Candidates() ([]core.Candidater, error) {
	return bs.candidatesProvider.Candidates()
}

func (bs *BaremetalScheduler) Predicates() (map[string]core.FitPredicate, error) {
	return factory.GetPredicates(bs.algorithmProvider.FitPredicateKeys)
}

func (bs *BaremetalScheduler) PriorityConfigs() ([]core.PriorityConfig, error) {
	return factory.GetPriorityConfigs(bs.algorithmProvider.PriorityKeys)
}
