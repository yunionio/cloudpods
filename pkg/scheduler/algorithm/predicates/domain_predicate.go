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

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type DomainPredicate struct {
	BasePredicate
}

func (p *DomainPredicate) Name() string {
	return "host_domain"
}

func (p *DomainPredicate) Clone() core.FitPredicate {
	return &DomainPredicate{}
}

func (p *DomainPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)
	getter := c.Getter()
	if getter.DomainId() == u.SchedInfo.Domain {
	} else if getter.IsPublic() && getter.PublicScope() == string(rbacutils.ScopeSystem) {
	} else if getter.IsPublic() && getter.PublicScope() == string(rbacutils.ScopeDomain) && utils.IsInStringArray(u.SchedInfo.Domain, getter.SharedDomains()) {
		// } else if db.IsAdminAllowGet(context.Background(), u.SchedInfo.UserCred, getter) {
	} else {
		h.Exclude("domain_ownership")
	}
	return h.GetResult()
}
