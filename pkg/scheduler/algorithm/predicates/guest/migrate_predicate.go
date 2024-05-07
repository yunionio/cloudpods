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

package guest

import (
	"context"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// MigratePredicate filters whether the current candidate can be migrated.
type MigratePredicate struct {
	predicates.BasePredicate
}

func (p *MigratePredicate) Name() string {
	return "host_migrate"
}

func (p *MigratePredicate) Clone() core.FitPredicate {
	return &MigratePredicate{}
}

func (p *MigratePredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	if u.SchedData().ResetCpuNumaPin {
		return false, nil
	}

	return len(u.SchedData().HostId) > 0, nil
}

func (p *MigratePredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	schedData := u.SchedData()

	if schedData.HostId == c.IndexKey() {
		h.Exclude(predicates.ErrHostIsSpecifiedForMigration)
		return h.GetResult()
	}

	// live migrate check
	if schedData.LiveMigrate {
		host := c.Getter().Host()

		guestHypervisor := u.SchedData().Hypervisor
		if guestHypervisor != compute.HYPERVISOR_ESXI {
			// target host mem page size check
			if schedData.HostMemPageSizeKB != host.PageSizeKB {
				h.Exclude(predicates.ErrHostMemPageSizeNotMatchForLiveMigrate)
				return h.GetResult()
			}

			// target host cpu check
			if schedData.CpuMode != compute.CPU_MODE_QEMU && (schedData.SkipCpuCheck == nil || *schedData.SkipCpuCheck == false) {
				if schedData.CpuDesc != host.CpuDesc {
					h.Exclude(predicates.ErrHostCpuModelIsNotMatchForLiveMigrate)
					return h.GetResult()
				}
				if len(schedData.CpuMicrocode) > 0 && schedData.CpuMicrocode != host.CpuMicrocode {
					h.Exclude(predicates.ErrHostCpuMicrocodeNotMatchForLiveMigrate)
					return h.GetResult()
				}
			}

			// target host kernel check
			if schedData.SkipKernelCheck != nil && !*schedData.SkipKernelCheck {
				kv, _ := host.SysInfo.GetString("kernel_version")
				if schedData.TargetHostKernel != "" && schedData.TargetHostKernel != kv {
					h.Exclude2(predicates.ErrHostKernelNotMatchForLiveMigrate, kv, schedData.TargetHostKernel)
					return h.GetResult()
				}
			}
		}
	}
	return h.GetResult()
}
