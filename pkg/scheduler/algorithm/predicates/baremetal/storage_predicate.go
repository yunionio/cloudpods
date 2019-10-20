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

package baremetal

import (
	"fmt"

	// computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
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

/*func toBaremetalDisks(disks []*computeapi.DiskConfig) []*baremetal.Disk {
	ret := make([]*baremetal.Disk, len(disks))
	for i, disk := range disks {
		ret[i] = &baremetal.Disk{
			Backend:    disk.Backend,
			ImageID:    disk.ImageId,
			Fs:         &disk.Fs,
			Format:     disk.Format,
			MountPoint: &disk.Mountpoint,
			Driver:     &disk.Driver,
			Cache:      &disk.Cache,
			Size:       int64(disk.SizeMb),
			Storage:    &disk.Storage,
		}
	}
	return ret
}*/

func (p *StoragePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	schedData := u.SchedData()

	storageInfo := c.Getter().StorageInfo()

	layouts, err := baremetal.CalculateLayout(
		schedData.BaremetalDiskConfigs,
		storageInfo,
	)

	if err == nil && baremetal.IsDisksAllocable(layouts, schedData.Disks) {
		h.SetCapacity(int64(1))
	} else {
		h.SetCapacity(int64(0))
		h.AppendPredicateFailMsg(fmt.Sprintf("%s err: %v", predicates.ErrNoEnoughStorage, err))
	}

	return h.GetResult()
}
