package baremetal

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type StoragePredicate struct {
	BasePredicate
}

func (p *StoragePredicate) Name() string {
	return "baremetal_storage"
}

func (p *StoragePredicate) Clone() core.FitPredicate {
	return &StoragePredicate{}
}

func toBaremetalDisks(disks []*api.Disk) []*baremetal.Disk {
	ret := make([]*baremetal.Disk, len(disks))
	for i, disk := range disks {
		ret[i] = &baremetal.Disk{
			Backend:         disk.Backend,
			ImageID:         disk.ImageID,
			Fs:              disk.Fs,
			Os:              disk.Os,
			OSDistribution:  disk.OSDistribution,
			Format:          disk.Format,
			MountPoint:      disk.MountPoint,
			Driver:          disk.Driver,
			Cache:           disk.Cache,
			ImageDiskFormat: disk.ImageDiskFormat,
			Size:            disk.Size,
			Storage:         disk.Storage,
		}
	}
	return ret
}

func (p *StoragePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	schedData := u.SchedData()

	candidate, err := h.BaremetalCandidate()
	if err != nil {
		return false, nil, err
	}

	layouts, err := baremetal.CalculateLayout(
		schedData.BaremetalDiskConfigs,
		candidate.Storages,
	)

	if err == nil && baremetal.CheckDisksAllocable(layouts, toBaremetalDisks(schedData.Disks)) {
		h.SetCapacity(int64(1))
	} else {
		h.SetCapacity(int64(0))
		h.AppendPredicateFailMsg(fmt.Sprintf("%s err: %v", predicates.ErrNoEnoughStorage, err))
	}

	return h.GetResult()
}
