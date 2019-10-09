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
	"sync"

	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

// NetworkPredicate will filter the current network information with
// the specified scheduling information to match, if not specified will
// randomly match the available network resources.
type NetworkPredicate struct {
	BasePredicate
	plugin.BasePlugin
	SelectedNetworks sync.Map
}

func (p *NetworkPredicate) Name() string {
	return "host_network"
}

func (p *NetworkPredicate) Clone() core.FitPredicate {
	return &NetworkPredicate{}
}

func (p *NetworkPredicate) PreExecute(u *core.Unit, cs []core.Candidater) (bool, error) {
	data := u.SchedData()
	if len(data.HostId) > 0 && len(data.Networks) == 0 {
		return false, nil
	}

	return true, nil
}

func (p *NetworkPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	getter := c.Getter()
	networks := getter.Networks()

	d := u.SchedData()

	isMigrate := func() bool {
		return len(d.HostId) > 0
	}

	counterOfNetwork := func(u *core.Unit, n *models.SNetwork, r int) core.Counter {
		counter := u.CounterManager.GetOrCreate("net:"+n.Id, func() core.Counter {
			return core.NewNormalCounter(int64(getter.GetFreePort(n.Id) - r))
		})

		u.SharedResourceManager.Add(n.GetId(), counter)
		return counter
	}

	isMatchServerType := func(network *models.SNetwork) bool {
		if d.Hypervisor == computeapi.HYPERVISOR_BAREMETAL {
			return network.ServerType == computeapi.NETWORK_TYPE_BAREMETAL
		}
		return sets.NewString(
			"", computeapi.NETWORK_TYPE_GUEST,
			computeapi.NETWORK_TYPE_CONTAINER).Has(network.ServerType)
	}

	checkAddress := func(addr string, net *models.SNetwork) error {
		if len(addr) == 0 {
			return nil
		}
		ipAddr, err := netutils.NewIPV4Addr(addr)
		if err != nil {
			return fmt.Errorf("Invalid ip address %s: %v", addr, err)
		}
		if !net.GetIPRange().Contains(ipAddr) {
			return fmt.Errorf("Address %s not in network %s range", addr, net.Name)
		}
		return nil
	}

	checkNetCount := func(net *models.SNetwork, reqCount int) (core.Counter, core.PredicateFailureReason) {
		counter := counterOfNetwork(u, net, 0)
		if counter.GetCount() < int64(reqCount) {
			return nil, FailReason{
				Reason: fmt.Sprintf("%s: ports not enough, free: %d, required: %d", net.Name, counter.GetCount(), d.Count),
				Type:   NetworkFreeCount,
			}
		}
		return counter, nil
	}

	isRandomNetworkAvailable := func(address, domain string, private bool, exit bool, wire string, counters core.MultiCounter) []core.PredicateFailureReason {
		var fullErrMsgs []core.PredicateFailureReason
		found := false

		for _, n := range networks {
			errMsgs := []core.PredicateFailureReason{}
			appendError := func(msg core.PredicateFailureReason) {
				errMsgs = append(errMsgs, msg)
			}

			if !isMatchServerType(n.SNetwork) {
				appendError(FailReason{
					Reason: fmt.Sprintf("Network %s type %s match", n.Name, n.ServerType),
					Type:   NetworkTypeMatch,
				})
			}

			if n.IsExitNetwork() != exit {
				appendError(FailReason{Reason: ErrExitIsNotMatch})
			}

			if !(n.GetPorts() > 0 || isMigrate()) {
				appendError(FailReason{
					Reason: fmt.Sprintf("%v(%v): ports use up", n.Name, n.Id),
					Type:   NetworkPort,
				})
			}

			if wire != "" && !utils.HasPrefix(wire, n.WireId) && !utils.HasPrefix(wire, n.GetWire().GetName()) {
				appendError(FailReason{
					Reason: fmt.Sprintf("Wire %s != %s", wire, n.WireId),
					Type:   NetworkWire,
				})
			}

			schedData := u.SchedData()
			if private {
				if n.IsPublic {
					appendError(FailReason{
						Reason: fmt.Sprintf("Network %s is public", n.Name),
						Type:   NetworkPublic,
					})
				} else if n.ProjectId != schedData.Project && utils.IsInStringArray(schedData.Project, n.GetSharedProjects()) {
					appendError(FailReason{
						Reason: fmt.Sprintf("Network project %s + %v not owner by %s", n.ProjectId, n.GetSharedProjects(), schedData.Project),
						Type:   NetworkOwner,
					})
				}
			} else {
				if !n.IsPublic {
					appendError(FailReason{Reason: fmt.Sprintf("Network %s is private", n.Name), Type: NetworkPrivate})
				} else if rbacutils.TRbacScope(n.PublicScope) == rbacutils.ScopeDomain {
					netDomain := n.DomainId
					reqDomain := domain
					if netDomain != reqDomain {
						appendError(FailReason{Reason: fmt.Sprintf("Network %s domain scope %s not owner by %s", n.Name, netDomain, reqDomain), Type: NetworkDomain})
					}
				}
			}

			if err := checkAddress(address, n.SNetwork); err != nil {
				appendError(FailReason{Reason: err.Error(), Type: NetworkRange})
			}

			if len(errMsgs) == 0 {
				// add resource
				counter := counterOfNetwork(u, n.SNetwork, 0)
				p.SelectedNetworks.Store(n.GetId(), counter.GetCount())
				counters.Add(counter)
				found = true
			} else {
				fullErrMsgs = append(fullErrMsgs, errMsgs...)
			}
		}

		if counters.GetCount() < int64(d.Count) {
			found = false
			fullErrMsgs = append(fullErrMsgs, FailReason{
				Reason: fmt.Sprintf("total random ports not enough, free: %d, required: %d", counters.GetCount(), d.Count),
				Type:   NetworkFreeCount,
			})
		}

		if !found {
			return fullErrMsgs
		}

		return nil
	}

	filterByRandomNetwork := func() {
		counters := core.NewCounters()
		if errMsg := isRandomNetworkAvailable("", u.SchedData().Domain, false, false, "", counters); len(errMsg) != 0 {
			h.ExcludeByErrors(errMsg)
		}
		h.SetCapacityCounter(counters)
	}

	isNetworkAvaliable := func(n *computeapi.NetworkConfig, counters *core.MinCounters, networks []*api.CandidateNetwork) []core.PredicateFailureReason {
		if len(networks) == 0 {
			return []core.PredicateFailureReason{
				FailReason{Reason: ErrNoAvailableNetwork},
			}
		}

		if n.Network == "" {
			counters0 := core.NewCounters()
			retMsg := isRandomNetworkAvailable(n.Address, n.Domain, n.Private, n.Exit, n.Wire, counters0)
			counters.Add(counters0)
			return retMsg
		}

		errMsgs := make([]core.PredicateFailureReason, 0)

		for _, net := range networks {
			if !(n.Network == net.GetId() || n.Network == net.GetName()) {
				errMsgs = append(errMsgs, &FailReason{
					Reason: fmt.Sprintf("%v(%v): id/name not matched", net.Name, net.Id),
					Type:   NetworkMatch,
				})
				continue
			}
			if !(net.GetPorts() > 0 || isMigrate()) {
				errMsgs = append(errMsgs, &FailReason{
					Reason: fmt.Sprintf("%v(%v): ports use up", net.Name, net.Id),
					Type:   NetworkPort,
				})
				continue
			}
			counter, err := checkNetCount(net.SNetwork, d.Count)
			if err != nil {
				errMsgs = append(errMsgs, err)
				continue
			}
			p.SelectedNetworks.Store(net.Id, counter.GetCount())
			counters.Add(counter)
			return nil
		}

		return errMsgs
	}

	filterBySpecifiedNetworks := func() {
		counters := core.NewMinCounters()
		var errMsgs []core.PredicateFailureReason

		for _, n := range d.Networks {
			if errMsg := isNetworkAvaliable(n, counters, networks); len(errMsg) != 0 {
				errMsgs = append(errMsgs, errMsg...)
			}
		}

		if len(errMsgs) > 0 {
			h.ExcludeByErrors(errMsgs)
		} else {
			h.SetCapacityCounter(counters)
		}
	}

	if len(d.Networks) == 0 {
		filterByRandomNetwork()
	} else {
		filterBySpecifiedNetworks()
	}

	return h.GetResult()
}
