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

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type CdromBootPredicate struct {
	BasePredicate
}

func (p *CdromBootPredicate) Name() string {
	return "cdrom_boot"
}

func (p *CdromBootPredicate) Clone() core.FitPredicate {
	return &CdromBootPredicate{}
}

func (p *CdromBootPredicate) PreExecute(u *core.Unit, _ []core.Candidater) (bool, error) {
	cdrom := u.SchedData().Cdrom
	if len(cdrom) == 0 {
		return false, nil
	}
	return true, nil
}

func (p *CdromBootPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	info := c.Getter().GetIpmiInfo()
	if !info.CdromBoot {
		h.Exclude(fmt.Sprintf("ipmi not support cdrom boot"))
	}
	return h.GetResult()
}
