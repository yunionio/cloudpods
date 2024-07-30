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

func (p *InstanceTypePredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	driver := u.GetHypervisorDriver()
	if u.SchedData().ResetCpuNumaPin {
		return false, nil
	}

	if u.SchedData().InstanceType == "" || (driver == nil || !driver.DoScheduleSKUFilter()) {
		return false, nil
	}
	return true, nil
}

func (p *InstanceTypePredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	d := u.SchedData()

	regionId := c.Getter().Region().Id
	regionName := c.Getter().Region().Name
	zoneId := c.Getter().Zone().Id
	zoneName := c.Getter().Zone().Name
	instanceType := d.InstanceType

	reqRegion := d.PreferRegion
	reqZone := d.PreferZone

	if reqRegion != "" && reqZone == "" {
		skus := skuman.GetByRegion(instanceType, regionId)
		if len(skus) == 0 {
			h.Exclude(fmt.Sprintf("Not found server sku %s at region %s", instanceType, regionName))
		} else {
			zoneMatch := false
			for idx := range skus {
				sku := skus[idx]
				if sku.ZoneId == zoneId {
					zoneMatch = true
					break
				}
			}
			if !zoneMatch {
				h.Exclude(fmt.Sprintf("Not found server sku %s at zone %s", instanceType, zoneName))
			}
		}
	} else {
		sku := skuman.GetByZone(instanceType, zoneId)
		if sku == nil {
			h.Exclude(fmt.Sprintf("Not found server sku %s at zone %s", instanceType, zoneName))
		}
	}

	return h.GetResult()
}
