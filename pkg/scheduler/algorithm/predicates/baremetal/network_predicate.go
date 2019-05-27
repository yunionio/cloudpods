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
	"strings"
	"sync"

	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type NetworkPredicate struct {
	BasePredicate
	plugin.BasePlugin
	SelectedNetworks sync.Map
}

func (p *NetworkPredicate) Name() string {
	return "baremetal_network"
}

func (p *NetworkPredicate) Clone() core.FitPredicate {
	return &NetworkPredicate{}
}

func (p *NetworkPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	notIgnore, _ := p.BasePredicate.PreExecute(u, cs)
	if !notIgnore {
		return false, nil
	}

	u.AppendSelectPlugin(p)
	d := u.SchedData()
	if len(d.HostId) > 0 && len(d.Networks) == 0 {
		return false, nil
	}

	return true, nil
}

func (p *NetworkPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	schedData := u.SchedData()

	networks := c.Getter().Networks()

	counters := core.NewCounters()

	isMigrate := func() bool {
		return len(schedData.HostId) > 0
	}

	isRandomNetworkAvailable := func(private bool, exit bool, wire string) string {
		var errMsgs []string
		for _, network := range networks {
			appendError := func(errMsg string) {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", network.Id, errMsg))
			}
			if !((network.GetPorts() > 0 || isMigrate()) && network.IsExitNetwork() == exit) {
				appendError(predicates.ErrNoPorts)
			}
			if wire != "" && !utils.HasPrefix(wire, network.WireId) && !utils.HasPrefix(wire, network.GetWire().GetName()) { // re
				appendError(predicates.ErrWireIsNotMatch)
			}
			if (!private && network.IsPublic) || (private && !network.IsPublic && network.ProjectId == schedData.Project) {
				// TODO: support reservedNetworks
				reservedNetworks := 0
				restPort := int64(network.GetPorts() - reservedNetworks)
				if restPort == 0 {
					appendError("not enough network port")
					continue
				}
				counter := u.CounterManager.GetOrCreate("net:"+network.Id, func() core.Counter {
					return core.NewNormalCounter(restPort)
				})

				u.SharedResourceManager.Add(network.Id, counter)
				counters.Add(counter)
				p.SelectedNetworks.Store(network.Id, counter.GetCount())
				return ""
			} else {
				appendError(predicates.ErrNotOwner)
			}
		}

		return strings.Join(errMsgs, ";")
	}

	filterByRandomNetwork := func() {
		if err_msg := isRandomNetworkAvailable(false, false, ""); err_msg != "" {
			h.Exclude(err_msg)
		}
		h.SetCapacityCounter(counters)
	}

	isNetworkAvaliable := func(network *computeapi.NetworkConfig) string {
		if network.Network == "" {
			return isRandomNetworkAvailable(network.Private, network.Exit, network.Wire)
		}
		for _, net := range networks {
			if (network.Network == net.Id || network.Network == net.Name) && (net.IsPublic || net.ProjectId == schedData.Project) && (net.GetPorts() > 0 || isMigrate()) {
				h.SetCapacity(1)
				return ""
			}
		}

		return predicates.ErrUnknown
	}

	filterBySpecifiedNetworks := func() {
		var errMsgs []string

		for _, network := range schedData.Networks {
			if err_msg := isNetworkAvaliable(network); err_msg != "" {
				errMsgs = append(errMsgs, err_msg)
			}
		}

		if len(errMsgs) > 0 {
			h.Exclude(strings.Join(errMsgs, ", "))
		}
	}

	loadUnknownNetworks := func() bool {
		return true // TODO: ???
	}

	loadUnknownNetworks()

	// Randomly assign networks if no network is specified.
	if len(schedData.Networks) == 0 {
		filterByRandomNetwork()
	} else {
		filterBySpecifiedNetworks()
	}

	return h.GetResult()
}
