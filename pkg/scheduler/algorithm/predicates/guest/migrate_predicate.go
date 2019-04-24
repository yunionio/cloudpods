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

func (p *MigratePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	return len(u.SchedData().HostId) > 0, nil
}

func (p *MigratePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	if u.SchedData().HostId == c.IndexKey() {
		h.Exclude(predicates.ErrHostIsSpecifiedForMigration)
	}

	return h.GetResult()
}
