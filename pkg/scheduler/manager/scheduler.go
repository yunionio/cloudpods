package manager

import (
	"github.com/yunionio/onecloud/pkg/scheduler/api"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
	"github.com/yunionio/onecloud/pkg/scheduler/data_manager"
	"github.com/yunionio/onecloud/pkg/scheduler/factory"
)

type CandidatesProvider interface {
	ProviderType() string
	CandidateType() string
	Candidates() ([]core.Candidater, error)
	CandidateManager() *data_manager.CandidateManager
}

func candidatesByProvider(provider CandidatesProvider, schedData *api.SchedData) ([]core.Candidater, error) {
	var hosts []core.Candidater
	var err error

	candidateManager := provider.CandidateManager()
	if len(schedData.Candidates) > 0 {
		hosts, err = candidateManager.GetCandidatesByIds(provider.CandidateType(), schedData.Candidates)
	} else {
		args := data_manager.CandidateGetArgs{
			ResType: provider.CandidateType(),
			ZoneID:  schedData.ZoneID,
			PoolID:  schedData.PoolID,
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
	SchedData() *api.SchedData
	CandidateManager() *data_manager.CandidateManager

	// Schedule process
	BeforePredicate() error
	Predicates() (map[string]core.FitPredicate, error)
	PriorityConfigs() ([]core.PriorityConfig, error)

	// Schedule input get function
	Unit() *core.Unit
	Candidates() ([]core.Candidater, error)

	DirtySelectedCandidates([]*core.SelectedCandidate)
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

func (s *BaseScheduler) SchedData() *api.SchedData {
	return s.schedInfo.Data
}

func (s *BaseScheduler) Unit() *core.Unit {
	return s.NewSchedUnit()
}

func (s *BaseScheduler) BeforePredicate() error {
	return nil
}

func (s *BaseScheduler) DirtySelectedCandidates(scs []*core.SelectedCandidate) {
	s.CandidateManager().SetCandidatesDirty(scs)
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
