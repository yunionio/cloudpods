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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

const (
	CONTAINER_ALLOWED_TAG = "container"
)

// HypervisorPredicate is to select candidates match guest hyperviosr
// runtime
type HypervisorPredicate struct {
	predicates.BasePredicate
}

func (f *HypervisorPredicate) Name() string {
	return "host_hypervisor_runtime"
}

func (f *HypervisorPredicate) Clone() core.FitPredicate {
	return &HypervisorPredicate{}
}

func hostHasContainerTag(c core.Candidater) bool {
	aggs := c.Getter().HostSchedtags()
	for _, agg := range aggs {
		if agg.Name == CONTAINER_ALLOWED_TAG {
			return true
		}
	}
	return false
}

func hostAllowRunContainer(c core.Candidater) bool {
	hostType := c.Get("HostType")
	if hostType == api.HostTypeKubelet {
		return true
	}
	if hostHasContainerTag(c) {
		log.Debugf("Host %q has %q tag, allow it run container", c.IndexKey(), CONTAINER_ALLOWED_TAG)
		return true
	}
	return false
}

func (f *HypervisorPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(f, u, c)

	hostType := c.Get("HostType")
	guestNeedType := u.SchedData().Hypervisor

	if guestNeedType != hostType {
		if guestNeedType == api.SchedTypeContainer && hostAllowRunContainer(c) {
			return h.GetResult()
		}
		h.Exclude2(f.Name(), hostType, guestNeedType)
	}
	return h.GetResult()
}
