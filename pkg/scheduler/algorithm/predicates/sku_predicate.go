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
	"fmt"

	"yunion.io/x/onecloud/pkg/scheduler/core"
	skuman "yunion.io/x/onecloud/pkg/scheduler/data_manager/sku"
)

type InstanceTypePredicate struct {
	BasePredicate
}

func (p *InstanceTypePredicate) Name() string {
	return "instance_type"
}

func (p *InstanceTypePredicate) Clone() core.FitPredicate {
	return &InstanceTypePredicate{}
}

func (p *InstanceTypePredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	if u.SchedData().InstanceType == "" || !u.IsPublicCloudProvider() {
		return false, nil
	}
	return true, nil
}

func (p *InstanceTypePredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	d := u.SchedData()

	zoneId := c.Getter().Zone().Id
	zoneName := c.Getter().Zone().Name
	instanceType := d.InstanceType

	sku := skuman.GetByZone(instanceType, zoneId)
	if sku == nil {
		h.Exclude(fmt.Sprintf("Not found server sku %s at zone %s", instanceType, zoneName))
	}

	return h.GetResult()
}
