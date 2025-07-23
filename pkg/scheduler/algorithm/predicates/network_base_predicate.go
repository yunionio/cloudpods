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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/scheduler/core"
	schedmodels "yunion.io/x/onecloud/pkg/scheduler/models"
)

type NetworkBasePredicate struct {
	NetworkNicCountGetter INetworkNicCountGetter
	networkFreePortCount  map[string]int
}

func NewNetworkBasePredicate() *NetworkBasePredicate {
	return &NetworkBasePredicate{
		NetworkNicCountGetter: nil,
		networkFreePortCount:  make(map[string]int),
	}
}

func (p *NetworkBasePredicate) Clone() *NetworkBasePredicate {
	return &NetworkBasePredicate{
		NetworkNicCountGetter: p.NetworkNicCountGetter,
		networkFreePortCount:  p.networkFreePortCount,
	}
}

func (p *NetworkBasePredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	networkIds := sets.NewString()
	for i := range cs {
		for _, net := range cs[i].Getter().Networks() {
			networkIds.Insert(net.GetId())
		}
	}

	if p.NetworkNicCountGetter != nil {
		netCounts, err := p.NetworkNicCountGetter.GetTotalNicCount(networkIds.UnsortedList())
		if err != nil {
			return false, errors.Wrap(err, "unable to GetTotalNicCount")
		}
		for i := range cs {
			for _, net := range cs[i].Getter().Networks() {
				p.networkFreePortCount[net.Id] = net.GetTotalAddressCount() - netCounts[net.Id]
			}
		}
	}

	return true, nil
}

func (p *NetworkBasePredicate) GetFreePort(id string, c core.Candidater) int {
	if _, ok := p.networkFreePortCount[id]; ok {
		return p.networkFreePortCount[id] - schedmodels.HostPendingUsageManager.GetNetPendingUsage(id)
	}
	return c.Getter().GetFreePort(id)
}
