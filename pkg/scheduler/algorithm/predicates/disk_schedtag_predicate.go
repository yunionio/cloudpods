package predicates

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/errors"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type DiskStoragesMap map[int][]*api.CandidateStorage

type CandidateDiskStoragesMap map[string]DiskStoragesMap

type DiskSchedtagPredicate struct {
	BasePredicate
	plugin.BasePlugin

	SchedtagPredicate *SchedtagPredicate

	CandidateDiskStoragesMap CandidateDiskStoragesMap
}

func (p *DiskSchedtagPredicate) Name() string {
	return "disk_schedtag"
}

func (p *DiskSchedtagPredicate) Clone() core.FitPredicate {
	return &DiskSchedtagPredicate{
		CandidateDiskStoragesMap: make(map[string]DiskStoragesMap),
	}
}

func (p *DiskSchedtagPredicate) getSchedtagDisks(disks []*api.Disk) ([]*api.Disk, []*api.Disk) {
	noTagDisk := make([]*api.Disk, 0)
	tagDisk := make([]*api.Disk, 0)
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

	// always select each storages to disks
	u.AppendSelectPlugin(p)

	return true, nil
}

type schedtagStorageW struct {
	candidater *api.CandidateStorage
	disk       *api.Disk
}

func (w schedtagStorageW) IndexKey() string {
	return fmt.Sprintf("%d:%s", w.disk.Size, w.disk.Backend)
}

func (w schedtagStorageW) GetDynamicSchedDesc() *jsonutils.JSONDict {
	return nil
}

func (w schedtagStorageW) GetSchedtags() []models.SSchedtag {
	return w.candidater.Schedtags
}

func (w schedtagStorageW) ResourceType() string {
	return models.StorageManager.KeywordPlural()
}

func (p *DiskSchedtagPredicate) check(d *api.Disk, s *api.CandidateStorage) (bool, error) {
	allTags, err := GetAllSchedtags(models.StorageManager.KeywordPlural())
	if err != nil {
		return false, err
	}
	tagPredicate := NewSchedtagPredicate(d.Schedtags, allTags)
	if err := tagPredicate.Check(
		schedtagStorageW{
			candidater: s,
			disk:       d,
		},
	); err != nil {
		return false, err
	}
	return true, nil
}

func (p *DiskSchedtagPredicate) checkStorages(d *api.Disk, storages []*api.CandidateStorage) ([]*api.CandidateStorage, error) {
	errs := make([]error, 0)
	ret := make([]*api.CandidateStorage, 0)
	for _, s := range storages {
		_, err := p.check(d, s)
		if err != nil {
			// append err, storage not suit disk
			errs = append(errs, err)
			continue
		}
		ret = append(ret, s)
	}
	if len(ret) == 0 {
		return nil, errors.NewAggregate(errs)
	}
	return ret, nil
}

func (p *DiskSchedtagPredicate) GetDiskStoragesMap(candidateId string) DiskStoragesMap {
	ret, ok := p.CandidateDiskStoragesMap[candidateId]
	if !ok {
		ret = make(map[int][]*api.CandidateStorage)
		p.CandidateDiskStoragesMap[candidateId] = ret
	}
	return ret
}

func (p *DiskSchedtagPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	//noTagDisks, tagDisks := p.getSchedtagDisks(u.SchedData().Disks)
	storages := c.Getter().Storages()
	ds := p.GetDiskStoragesMap(c.IndexKey())
	disks := u.SchedData().Disks
	for _, d := range disks {
		matchedStorages, err := p.checkStorages(d, storages)
		if err != nil {
			h.Exclude(err.Error())
		}
		ds[d.Index] = matchedStorages
	}

	return h.GetResult()
}

func (p *DiskSchedtagPredicate) OnSelectEnd(u *core.Unit, c core.Candidater, count int64) {
	res := u.GetAllocatedResource(c.IndexKey())
	diskStorages := p.GetDiskStoragesMap(c.IndexKey())
	res.Disks = make([]*core.DiskAllocatedResource, len(diskStorages))
	disks := u.SchedData().Disks
	for idx, ds := range diskStorages {
		res.Disks[idx] = p.allocatedDiskResource(c, disks[idx], ds)
	}
	log.Errorf("============OnSelectEnd %s called: %#v", c.Getter().Name(), jsonutils.Marshal(res.Disks).String())
}

func (p *DiskSchedtagPredicate) allocatedDiskResource(c core.Candidater, disk *api.Disk, storages []*api.CandidateStorage) *core.DiskAllocatedResource {
	storage := p.selectStorage(disk, storages)
	return &core.DiskAllocatedResource{
		Index:     disk.Index,
		StorageId: storage.Id,
	}
}

func (p *DiskSchedtagPredicate) selectStorage(d *api.Disk, storages []*api.CandidateStorage) *api.CandidateStorage {
	return storages[0]
}
