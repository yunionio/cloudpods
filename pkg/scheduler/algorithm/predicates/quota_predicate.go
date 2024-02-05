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

package predicates

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/options"
)

type SQuotaPredicate struct {
	BasePredicate
}

func (p *SQuotaPredicate) Name() string {
	return "quota"
}

func (p *SQuotaPredicate) Clone() core.FitPredicate {
	return &SQuotaPredicate{}
}

func (p *SQuotaPredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	if !options.Options.EnableQuotaCheck {
		return false, nil
	}
	if len(u.SchedData().HostId) > 0 {
		return false, nil
	}
	return true, nil
}

func fetchGuestUsageFromSchedInfo(ctx context.Context, s *api.SchedInfo) (computemodels.SQuota, computemodels.SRegionQuota) {
	vcpuCount := s.Ncpu
	if vcpuCount == 0 {
		vcpuCount = 1
	}

	vmemSize := s.Memory

	diskSize := 0

	for _, diskConfig := range s.Disks {
		diskSize += diskConfig.SizeMb
	}

	devCount := len(s.IsolatedDevices)

	eNicCnt := 0
	iNicCnt := 0

	for _, netConfig := range s.Networks {
		if computemodels.IsExitNetworkInfo(ctx, s.UserCred, netConfig) {
			eNicCnt += 1
		} else {
			iNicCnt += 1
		}
	}

	req := computemodels.SQuota{
		Count:          1,
		Cpu:            int(vcpuCount),
		Memory:         int(vmemSize),
		Storage:        diskSize,
		IsolatedDevice: devCount,
	}
	regionReq := computemodels.SRegionQuota{
		Port:  iNicCnt,
		Eport: eNicCnt,
	}
	return req, regionReq
}

func (p *SQuotaPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	d := u.SchedData()

	computePending := computemodels.SQuota{}
	regionPending := computemodels.SRegionQuota{}
	if len(d.PendingUsages) > 0 {
		d.PendingUsages[0].Unmarshal(&computePending)
	}
	if len(d.PendingUsages) > 1 {
		d.PendingUsages[1].Unmarshal(&regionPending)
	}
	computePending.ProjectId = d.Project
	computePending.DomainId = d.Domain
	regionPending.ProjectId = d.Project
	regionPending.DomainId = d.Domain

	computeKeys := c.Getter().GetQuotaKeys(d)

	computeQuota, regionQuota := fetchGuestUsageFromSchedInfo(ctx, d)

	computeQuota.SetKeys(computeKeys)
	regionQuota.SetKeys(computeKeys.SRegionalCloudResourceKeys)

	minCnt := -1
	computeCnt, _ := quotas.GetQuotaCount(ctx, &computeQuota, computePending.GetKeys())
	if computeCnt >= 0 && (minCnt < 0 || minCnt > computeCnt) {
		minCnt = computeCnt
	}
	regionCnt, _ := quotas.GetQuotaCount(ctx, &regionQuota, regionPending.GetKeys())
	if regionCnt >= 0 && (minCnt < 0 || minCnt > regionCnt) {
		minCnt = regionCnt
	}

	if minCnt > 0 {
		h.SetCapacity(int64(minCnt))
	} else if minCnt == 0 {
		h.Exclude("out_of_quota")
	}

	return h.GetResult()
}
