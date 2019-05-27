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

/*import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// NestPredicate will filter whether the current host is turned on KVM,
//if the scheduling specified settings are inconsistent, then the host
// will be filtered out.
type NestPredicate struct {
	predicates.BasePredicate
}

func (p *NestPredicate) Name() string {
	return "host_nest"
}

func (p *NestPredicate) Clone() core.FitPredicate {
	return &NestPredicate{}
}

func (p *NestPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	hc, err := h.HostCandidate()
	if err != nil {
		return false, nil, err
	}

	d := u.SchedData()

	if d.Metadata["kvm"] == "enabled" {
		if hc.Metadata["nest"] != "enabled" {
			h.Exclude(predicates.ErrNotSupportNest)
		}
	}

	return h.GetResult()
}*/
