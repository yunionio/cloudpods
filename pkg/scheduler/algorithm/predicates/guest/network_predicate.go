package guest

import (
	"fmt"
	"strings"
	"sync"

	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/compute/models"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// NetworkPredicate will filter the current network information with
// the specified scheduling information to match, if not specified will
// randomly match the available network resources.
type NetworkPredicate struct {
	predicates.BasePredicate
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
	isMatchServerType := func(network *models.SNetwork) bool {
		return sets.NewString("guest", "", "container").Has(network.ServerType)
	}

	counterOfNetwork := func(u *core.Unit, n *models.SNetwork, r int) core.Counter {
		counter := u.CounterManager.GetOrCreate("net:"+n.Id, func() core.Counter {
			return core.NewNormalCounter(int64(n.GetPorts() - r))
		})

		u.SharedResourceManager.Add(n.GetId(), counter)
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

			if !isMatchServerType(&n) {
				appendError(predicates.ErrServerTypeIsNotMatch)
			}

			if n.IsExitNetwork() != exit {
				appendError(predicates.ErrExitIsNotMatch)
			}

			if !(n.GetPorts() > 0 || isMigrate()) {
				appendError(predicates.ErrNoPorts)
			}

			if wire != "" && !utils.HasPrefix(wire, n.WireId) && !utils.HasPrefix(wire, n.GetWire().GetName()) { // re
				appendError(predicates.ErrWireIsNotMatch)
			}

			if !((!private && n.IsPublic) || (private && !n.IsPublic && n.ProjectId == d.OwnerTenantID)) {
				appendError(predicates.ErrNotOwner)
			}

			if len(errMsgs) == 0 {
				// add resource
				reservedNetworks := 0
				counter := counterOfNetwork(u, &n, reservedNetworks)
				p.SelectedNetworks.Store(n.GetId(), counter.GetCount())
				counters.Add(counter)
				found = true
			} else {
				fullErrMsgs = append(fullErrMsgs,
					fmt.Sprintf("%s: %s", n.Id, strings.Join(errMsgs, ",")),
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
		networks []models.SNetwork) string {
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
			/*if !isMatchServerType(net) {
				errMsgs = append(errMsgs, fmt.Sprintf("%v(%v): server type not matched", net.Name, net.ID))
				continue
			}*/
			if !(n.Idx == net.GetId() || n.Idx == net.GetName()) {
				errMsgs = append(errMsgs, fmt.Sprintf("%v(%v): id/name not matched", net.Name, net.Id))
			} else if !(net.IsPublic || net.ProjectId == d.OwnerTenantID) {
				errMsgs = append(errMsgs, fmt.Sprintf("%v(%v): not owner (%v != %v)", net.Name, net.Id, net.ProjectId, d.OwnerTenantID))
			} else if !(net.GetPorts() > 0 || isMigrate()) {
				errMsgs = append(errMsgs, fmt.Sprintf("%v(%v): ports use up", net.Name, net.Id))
			} else {
				// add resource
				reservedNetworks := 0
				counter := counterOfNetwork(u, &net, reservedNetworks)
				if counter.GetCount() < d.Count {
					errMsgs = append(errMsgs, fmt.Sprintf("%s: ports not enough, free: %d, required: %d", net.Name, counter.GetCount(), d.Count))
					continue
				}
				p.SelectedNetworks.Store(net.Id, counter.GetCount())
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
