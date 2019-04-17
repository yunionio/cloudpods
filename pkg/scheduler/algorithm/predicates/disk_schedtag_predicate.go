package predicates

import (
	"fmt"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type DiskStoragesMap map[int][]*PredicatedStorage

func (m DiskStoragesMap) getAllTags(isPrefer bool) []computeapi.SchedtagConfig {
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

func (m DiskStoragesMap) GetPreferTags() []computeapi.SchedtagConfig {
	return m.getAllTags(true)
}

func (m DiskStoragesMap) GetAvoidTags() []computeapi.SchedtagConfig {
	return m.getAllTags(false)
}

type CandidateDiskStoragesMap map[string]DiskStoragesMap

type PredicatedStorage struct {
	*api.CandidateStorage
	PreferTags []computeapi.SchedtagConfig
	AvoidTags  []computeapi.SchedtagConfig
}

func newPredicatedStorage(s *api.CandidateStorage, preferTags, avoidTags []computeapi.SchedtagConfig) *PredicatedStorage {
	return &PredicatedStorage{
		CandidateStorage: s,
		PreferTags:       preferTags,
		AvoidTags:        avoidTags,
	}
}

func (s *PredicatedStorage) isNoTag() bool {
	return len(s.PreferTags) == 0 && len(s.AvoidTags) == 0
}

func (s *PredicatedStorage) hasPreferTags() bool {
	return len(s.PreferTags) != 0
}

func (s *PredicatedStorage) hasAvoidTags() bool {
	return len(s.AvoidTags) != 0
}

type DiskSchedtagPredicate struct {
	BasePredicate
	plugin.BasePlugin

	SchedtagPredicate *SchedtagPredicate

	CandidateDiskStoragesMap CandidateDiskStoragesMap

	Hypervisor string
}

func (p *DiskSchedtagPredicate) Name() string {
	return "disk_schedtag"
}

func (p *DiskSchedtagPredicate) Clone() core.FitPredicate {
	return &DiskSchedtagPredicate{
		CandidateDiskStoragesMap: make(map[string]DiskStoragesMap),
	}
}

func (p *DiskSchedtagPredicate) getSchedtagDisks(disks []*computeapi.DiskConfig) ([]*computeapi.DiskConfig, []*computeapi.DiskConfig) {
	noTagDisk := make([]*computeapi.DiskConfig, 0)
	tagDisk := make([]*computeapi.DiskConfig, 0)
	for _, d := range disks {
		if len(d.Schedtags) != 0 {
			tagDisk = append(tagDisk, d)
		} else {
			noTagDisk = append(noTagDisk, d)
		}
	}
	return noTagDisk, tagDisk
}

func (p *DiskSchedtagPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	disks := u.SchedData().Disks
	if len(disks) == 0 {
		return false, nil
	}

	p.Hypervisor = computeapi.HOSTTYPE_HYPERVISOR[u.SchedData().Hypervisor]

	// always select each storages to disks
	u.AppendSelectPlugin(p)

	return true, nil
}

type schedtagStorageW struct {
	candidater *api.CandidateStorage
	disk       *computeapi.DiskConfig
}

func (w schedtagStorageW) IndexKey() string {
	return fmt.Sprintf("%s:%s", w.candidater.GetName(), w.candidater.StorageType)
}

func (w schedtagStorageW) GetDynamicSchedDesc() *jsonutils.JSONDict {
	ret := jsonutils.NewDict()
	storageSchedDesc := w.candidater.GetDynamicConditionInput()
	diskSchedDesc := w.disk.JSON(w.disk)
	ret.Add(storageSchedDesc, models.StorageManager.Keyword())
	ret.Add(diskSchedDesc, models.DiskManager.Keyword())
	return ret
}

func (w schedtagStorageW) GetSchedtags() []models.SSchedtag {
	return w.candidater.Schedtags
}

func (w schedtagStorageW) ResourceType() string {
	return models.StorageManager.KeywordPlural()
}

func (p *DiskSchedtagPredicate) check(d *computeapi.DiskConfig, s *api.CandidateStorage, u *core.Unit, c core.Candidater) (*PredicatedStorage, error) {
	allTags, err := GetAllSchedtags(models.StorageManager.KeywordPlural())
	if err != nil {
		return nil, err
	}
	tagPredicate := NewSchedtagPredicate(d.Schedtags, allTags)
	shouldExec := u.ShouldExecuteSchedtagFilter(c.Getter().Id())
	ps := newPredicatedStorage(s, nil, nil)
	if shouldExec {
		if err := tagPredicate.Check(
			schedtagStorageW{
				candidater: s,
				disk:       d,
			},
		); err != nil {
			return nil, err
		}
		ps.PreferTags = tagPredicate.GetPreferTags()
		ps.AvoidTags = tagPredicate.GetAvoidTags()
	}
	return ps, nil
}

func (p *DiskSchedtagPredicate) checkStorages(d *computeapi.DiskConfig, storages []*api.CandidateStorage, u *core.Unit, c core.Candidater) ([]*PredicatedStorage, error) {
	errs := make([]error, 0)
	ret := make([]*PredicatedStorage, 0)
	for _, s := range storages {
		ps, err := p.check(d, s, u, c)
		if err != nil {
			// append err, storage not suit disk
			errs = append(errs, err)
			continue
		}
		ret = append(ret, ps)
	}
	if len(ret) == 0 {
		return nil, errors.NewAggregate(errs)
	}
	return ret, nil
}

func (p *DiskSchedtagPredicate) GetDiskStoragesMap(candidateId string) DiskStoragesMap {
	ret, ok := p.CandidateDiskStoragesMap[candidateId]
	if !ok {
		ret = make(map[int][]*PredicatedStorage)
		p.CandidateDiskStoragesMap[candidateId] = ret
	}
	return ret
}

func (p *DiskSchedtagPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	storages := c.Getter().Storages()

	ds := p.GetDiskStoragesMap(c.IndexKey())
	disks := u.SchedData().Disks
	for idx, d := range disks {
		fitStorages := make([]*api.CandidateStorage, 0)
		for _, s := range storages {
			if p.isStorageFitDisk(s, d) {
				fitStorages = append(fitStorages, s)
			}
		}
		if len(fitStorages) == 0 {
			h.Exclude(fmt.Sprintf("Not found available storages for disk backend %q", d.Backend))
			break
		}

		matchedStorages, err := p.checkStorages(d, fitStorages, u, c)
		if err != nil {
			h.Exclude(err.Error())
		}
		ds[idx] = matchedStorages
	}

	return h.GetResult()
}

func (p *DiskSchedtagPredicate) OnPriorityEnd(u *core.Unit, c core.Candidater) {
	storageTags := []models.SSchedtag{}
	for _, s := range c.Getter().Storages() {
		storageTags = append(storageTags, s.Schedtags...)
	}

	ds := p.GetDiskStoragesMap(c.IndexKey())
	avoidTags := ds.GetAvoidTags()
	preferTags := ds.GetPreferTags()

	avoidCountMap := GetSchedtagCount(avoidTags, storageTags, api.AggregateStrategyAvoid)
	preferCountMap := GetSchedtagCount(preferTags, storageTags, api.AggregateStrategyPrefer)

	setScore := SetCandidateScoreBySchedtag

	setScore(u, c, preferCountMap, true)
	setScore(u, c, avoidCountMap, false)
}

func (p *DiskSchedtagPredicate) OnSelectEnd(u *core.Unit, c core.Candidater, count int64) {
	res := u.GetAllocatedResource(c.IndexKey())
	diskStorages := p.GetDiskStoragesMap(c.IndexKey())
	res.Disks = make([]*schedapi.CandidateDisk, len(diskStorages))
	disks := u.SchedData().Disks
	for idx, ds := range diskStorages {
		res.Disks[idx] = p.allocatedDiskResource(c, disks[idx], ds)
	}
}

func (p *DiskSchedtagPredicate) allocatedDiskResource(c core.Candidater, disk *computeapi.DiskConfig, storages []*PredicatedStorage) *schedapi.CandidateDisk {
	sortStorages := p.selectStorages(disk, storages)
	storageIds := []string{}
	for _, s := range sortStorages {
		storageIds = append(storageIds, s.Id)
	}
	log.Debugf("Suggestion %s storages %v for disk: %d", c.Getter().Name(), storageIds, disk.Index)
	return &schedapi.CandidateDisk{
		Index:      disk.Index,
		StorageIds: storageIds,
	}
}

func (p *DiskSchedtagPredicate) selectStorages(d *computeapi.DiskConfig, storages []*PredicatedStorage) []*api.CandidateStorage {
	preferStorages := []*api.CandidateStorage{}
	noTagStorages := []*api.CandidateStorage{}
	avoidStorages := []*api.CandidateStorage{}
	for _, storage := range storages {
		candi := storage.CandidateStorage
		if storage.isNoTag() {
			noTagStorages = append(noTagStorages, candi)
		} else if storage.hasPreferTags() {
			preferStorages = append(preferStorages, candi)
		} else if storage.hasAvoidTags() {
			avoidStorages = append(avoidStorages, candi)
		}
	}
	sortStorages := []*api.CandidateStorage{}
	sortStorages = append(sortStorages, preferStorages...)
	sortStorages = append(sortStorages, noTagStorages...)
	sortStorages = append(sortStorages, avoidStorages...)
	return p.SortLeastUsedStorage(sortStorages, d.Backend)
}

func (p *DiskSchedtagPredicate) SortLeastUsedStorage(storages []*api.CandidateStorage, backend string) []*api.CandidateStorage {
	backendStorages := make([]*api.CandidateStorage, 0)
	if backend != "" {
		for _, s := range storages {
			if s.StorageType == backend {
				backendStorages = append(backendStorages, s)
			}
		}
	} else {
		backendStorages = storages
	}
	if len(backendStorages) == 0 {
		backendStorages = storages
	}
	return p.sortByLeastUsedStorages(backendStorages)
}

type sortStorages []*api.CandidateStorage

func (s sortStorages) Len() int {
	return len(s)
}

func (s sortStorages) Less(i, j int) bool {
	s1, s2 := s[i], s[j]
	cap1 := s1.GetFreeCapacity()
	cap2 := s2.GetFreeCapacity()
	return cap1 > cap2
}

func (s sortStorages) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (p *DiskSchedtagPredicate) sortByLeastUsedStorages(storages []*api.CandidateStorage) []*api.CandidateStorage {
	ss := sortStorages(storages)
	sort.Sort(ss)
	return []*api.CandidateStorage(ss)
}

func (p *DiskSchedtagPredicate) GetHypervisorDriver() models.IGuestDriver {
	return models.GetDriver(p.Hypervisor)
}

func (p *DiskSchedtagPredicate) isStorageFitDisk(storage *api.CandidateStorage, d *computeapi.DiskConfig) bool {
	if d.Storage != "" {
		if storage.Id == d.Storage || storage.Name == d.Storage {
			return true
		}
		return false
	}
	if storage.StorageType == d.Backend {
		return true
	}

	for _, stype := range p.GetHypervisorDriver().GetStorageTypes() {
		if storage.StorageType == stype {
			return true
		}
	}
	return false
}
