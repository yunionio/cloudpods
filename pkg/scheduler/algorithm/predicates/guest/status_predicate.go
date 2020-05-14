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
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

const (
	ExpectedStatus       = "running"
	ExpectedHostStatus   = "online"
	ExpectedEnableStatus = "enable"
)

// StatusPredicate is to filter the current state of host is available,
// not available host's capacity will be set to 0 and filtered out.
type StatusPredicate struct {
	predicates.BasePredicate
}

func (p *StatusPredicate) Name() string {
	return "host_status"
}

func (p *StatusPredicate) Clone() core.FitPredicate {
	return &StatusPredicate{}
}

func (p *StatusPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	getter := c.Getter()
	curStatus := getter.Status()
	curHostStatus := getter.HostStatus()
	curEnableStatus := getter.Enabled()

	if curStatus != ExpectedStatus {
		h.Exclude2("status", curStatus, ExpectedStatus)
	}

	if curHostStatus != ExpectedHostStatus {
		h.Exclude2("host_status", curHostStatus, ExpectedHostStatus)
	}

	if !curEnableStatus {
		h.Exclude2("enable_status", curEnableStatus, true)
	}

	zone := getter.Zone()
	if zone.Status != ExpectedEnableStatus {
		h.Exclude2("zone_status", zone.Status, ExpectedEnableStatus)
	}

	cloudprovider := getter.Cloudprovider()
	if cloudprovider != nil {
		if !utils.IsInStringArray(cloudprovider.Status, api.CLOUD_PROVIDER_VALID_STATUS) {
			h.Exclude2("cloud_provider_status", cloudprovider.Status, api.CLOUD_PROVIDER_VALID_STATUS)
		}
		if !utils.IsInStringArray(cloudprovider.HealthStatus, api.CLOUD_PROVIDER_VALID_HEALTH_STATUS) {
			h.Exclude2("cloud_provider_health_status", cloudprovider.HealthStatus, api.CLOUD_PROVIDER_VALID_HEALTH_STATUS)
		}
		if !cloudprovider.Enabled.Bool() {
			h.Exclude2("cloud_provider_enable", cloudprovider.Enabled.Bool(), true)
		}
	}

	return h.GetResult()
}
