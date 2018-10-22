package guest

import (
	"fmt"
	"strings"
	"sync"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	networks "yunion.io/x/onecloud/pkg/scheduler/db/models"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
)

// NetworkPredicate will filter the current network information with
// the specified scheduling information to match, if not specified will
// randomly match the available network resources.
type NetworkPredicate struct {
	predicates.BasePredicate
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
	if len(data.HostID) > 0 && len(data.Networks) == 0 {
		return false, nil
	}

	return true, nil
}

func (p *NetworkPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)

	hc, err := h.HostCandidate()
	if err != nil {
		return false, nil, err
	}

	d := u.SchedData()

	isMigrate := func() bool {
		return len(d.HostID) > 0
	}

	// ServerType's value is 'guest', 'container' or ''(support all type) will return true.
	isMatchServerType := func(network *networks.NetworkSchedResult) bool {
		return sets.NewString("guest", "", "container").Has(network.ServerType)
	}

	counterOfNetwork := func(u *core.Unit, n *networks.NetworkSchedResult, r int) core.Counter {
		counter := u.CounterManager.GetOrCreate("net:"+n.ID, func() core.Counter {
			return core.NewNormalCounter(int64(n.Ports - r))
		})

		u.SharedResourceManager.Add(n.ID, counter)
		return counter
	}

	isRandomNetworkAvailable := func(private bool, exit bool, wire string,
		counters core.MultiCounter) string {

		var fullErrMsgs []string
		found := false

		for _, n := range hc.Networks {
			errMsgs := []string{}
			appendError := func(errMsg string) {
				errMsgs = append(errMsgs, errMsg)
			}

			if !isMatchServerType(n) {
				appendError(predicates.ErrServerTypeIsNotMatch)
			}

			if n.IsExit != exit {
				appendError(predicates.ErrExitIsNotMatch)
			}

			if !(n.Ports > 0 || isMigrate()) {
				appendError(predicates.ErrNoPorts)
			}

			if wire != "" && !utils.HasPrefix(wire, n.Wire) && !utils.HasPrefix(wire, n.WireID) { // re
				appendError(predicates.ErrWireIsNotMatch)
			}

			if !((!private && n.IsPublic) || (private && !n.IsPublic && n.TenantID == d.OwnerTenantID)) {
				appendError(predicates.ErrNotOwner)
			}

			if len(errMsgs) == 0 {
				// add resource
				reservedNetworks := 0
				counter := counterOfNetwork(u, n, reservedNetworks)
				p.SelectedNetworks.Store(n.ID, counter.GetCount())
				counters.Add(counter)
				found = true

				if counters.GetCount() >= d.Count {
					break
				}
			} else {
				fullErrMsgs = append(fullErrMsgs,
					fmt.Sprintf("%s: %s", n.ID, strings.Join(errMsgs, ",")),
				)
			}
		}

		if !found {
			return strings.Join(fullErrMsgs, "; ")
		}

		return ""
	}

	filterByRandomNetwork := func() {
		counters := core.NewCounters()
		if err_msg := isRandomNetworkAvailable(false, false, "", counters); err_msg != "" {
			h.AppendPredicateFailMsg(err_msg)
		}
		h.SetCapacityCounter(counters)
	}

	isNetworkAvaliable := func(n *api.Network, counters *core.MinCounters,
		networks []*networks.NetworkSchedResult) string {
		if n.Idx == "" {
			counters0 := core.NewCounters()
			ret_msg := isRandomNetworkAvailable(n.Private, n.Exit, n.Wire, counters0)
			counters.Add(counters0)
			return ret_msg
		}
		if len(hc.Networks) == 0 {
			return predicates.ErrNoAvailableNetwork
		}

		errMsgs := make([]string, 0)

		for _, net := range hc.Networks {
			if !isMatchServerType(net) {
				errMsgs = append(errMsgs, fmt.Sprintf("%v(%v): server type not matched", net.Name, net.ID))
				continue
			}
			if !(n.Idx == net.ID || n.Idx == net.Name) {
				//errMsgs = append(errMsgs, fmt.Sprintf("%v(%v): id/name not matched", net.Name, net.ID))
			} else if !(net.IsPublic || net.TenantID == d.OwnerTenantID) {
				errMsgs = append(errMsgs, fmt.Sprintf("%v(%v): not owner (%v != %v)", net.Name, net.ID, net.TenantID, d.OwnerTenantID))
			} else if !(net.Ports > 0 || isMigrate()) {
				errMsgs = append(errMsgs, fmt.Sprintf("%v(%v): ports use up", net.Name, net.ID))
			} else {
				// add resource
				reservedNetworks := 0
				counter := counterOfNetwork(u, net, reservedNetworks)
				if counter.GetCount() < d.Count {
					errMsgs = append(errMsgs, fmt.Sprintf("%s: ports not enough, free: %d, required: %d", net.Name, counter.GetCount(), d.Count))
					continue
				}
				p.SelectedNetworks.Store(net.ID, counter.GetCount())
				counters.Add(counter)
				return ""
			}
		}

		if len(errMsgs) == 0 {
			return predicates.ErrUnknown
		}

		return strings.Join(errMsgs, "; ")
	}

	filterBySpecifiedNetworks := func() {
		counters := core.NewMinCounters()
		var errMsgs []string

		for _, n := range d.Networks {
			if err_msg := isNetworkAvaliable(n, counters, hc.Networks); err_msg != "" {
				errMsgs = append(errMsgs, err_msg)
			}
		}

		if len(errMsgs) > 0 {
			h.AppendPredicateFailMsg(strings.Join(errMsgs, ", "))
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

func (p *NetworkPredicate) OnSelect(u *core.Unit, c core.Candidater) bool {
	u.SetFiltedData(c.IndexKey(), "networks", &p.SelectedNetworks)
	return true
}

func (p *NetworkPredicate) OnSelectEnd(u *core.Unit, c core.Candidater, count int64) {
}
