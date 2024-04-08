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
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type NetBondingPredicate struct {
	BasePredicate
}

func (p *NetBondingPredicate) Name() string {
	return "net_bonding"
}

func (p *NetBondingPredicate) Clone() core.FitPredicate {
	return &NetBondingPredicate{}
}

func (p *NetBondingPredicate) PreExecute(ctx context.Context, u *core.Unit, _ []core.Candidater) (bool, error) {
	netConfs := u.SchedData().Networks
	requireTeaming := false
	for _, conf := range netConfs {
		if conf.RequireTeaming {
			requireTeaming = true
			break
		}
	}
	if requireTeaming {
		return true, nil
	}
	return false, nil
}

func (p *NetBondingPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	wireNetIfs := c.Getter().NetInterfaces()
	netConfs := u.SchedData().Networks

	bondingCount := make(map[string]int)

	for _, netConf := range netConfs {
		if !netConf.RequireTeaming {
			continue
		}
		count := 0
		wireId := netConf.Wire
		if len(wireId) == 0 && len(netConf.Network) > 0 {
			for _, n := range c.Getter().Networks() {
				if n.Id == netConf.Network || n.Name == netConf.Network {
					wireId = n.WireId
					break
				}
			}
		}
		if _, ok := bondingCount[wireId]; ok {
			count = bondingCount[wireId]
		}
		bondingCount[wireId] = count + 2
	}
	for wireId, count := range bondingCount {
		if len(wireId) > 0 {
			ifCount := len(wireNetIfs[wireId])
			if ifCount < count {
				h.Exclude(fmt.Sprintf("Wire %s has %d netifs, require %d, can't do bonding", wireId, ifCount, count))
				return h.GetResult()
			}
		} else {
			maxIfCount := 0
			for _, netIfs := range wireNetIfs {
				ifCount := len(netIfs)
				if ifCount > maxIfCount {
					maxIfCount = ifCount
				}
				if ifCount >= count {
					return h.GetResult()
				}
			}
			h.Exclude(fmt.Sprintf("Not enough netifs for bonding, require %d netifs, max count %d", count, maxIfCount))
		}
	}
	return h.GetResult()
}
