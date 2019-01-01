package baremetal

import (
	"fmt"
	"strings"
	"sync"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	"yunion.io/x/onecloud/pkg/scheduler/api"
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
	if len(d.HostID) > 0 && len(d.Networks) == 0 {
		return false, nil
	}

	return true, nil
}

func (p *NetworkPredicate) Execute(u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := predicates.NewPredicateHelper(p, u, c)
	schedData := u.SchedData()

	candidate, err := h.BaremetalCandidate()
	if err != nil {
		return false, nil, err
	}

	counters := core.NewCounters()

	isMigrate := func() bool {
		return len(schedData.HostID) > 0
	}

	isRandomNetworkAvailable := func(private bool, exit bool, wire string) string {
		var errMsgs []string
		for _, network := range candidate.Networks {
			appendError := func(errMsg string) {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", network.ID, errMsg))
			}
			if !((network.Ports > 0 || isMigrate()) && network.IsExit == exit) {
				appendError(predicates.ErrNoPorts)
			}
			if wire != "" && !utils.HasPrefix(wire, network.Wire) && !utils.HasPrefix(wire, network.WireID) { // re
				appendError(predicates.ErrWireIsNotMatch)
			}
			if (!private && network.IsPublic) || (private && !network.IsPublic && network.TenantID == schedData.OwnerTenantID) {
				// TODO: support reservedNetworks
				reservedNetworks := 0
				restPort := int64(network.Ports - reservedNetworks)
				if restPort == 0 {
					appendError("not enough network port")
					continue
				}
				counter := u.CounterManager.GetOrCreate("net:"+network.ID, func() core.Counter {
					return core.NewNormalCounter(restPort)
				})

				u.SharedResourceManager.Add(network.ID, counter)
				counters.Add(counter)
				p.SelectedNetworks.Store(network.ID, counter.GetCount())
				return ""
			} else {
				appendError(predicates.ErrNotOwner)
			}
		}

		return strings.Join(errMsgs, ";")
	}

	filterByRandomNetwork := func() {
		if err_msg := isRandomNetworkAvailable(false, false, ""); err_msg != "" {
			h.AppendPredicateFailMsg(err_msg)
		}
		h.SetCapacityCounter(counters)
	}

	isNetworkAvaliable := func(network *api.Network) string {
		if network.Idx == "" {
			return isRandomNetworkAvailable(network.Private, network.Exit, network.Wire)
		}
		for _, net := range candidate.Networks {
			if (network.Idx == net.ID || network.Idx == net.Name) && (net.IsPublic || net.TenantID == schedData.OwnerTenantID) && (net.Ports > 0 || isMigrate()) {
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
			h.AppendPredicateFailMsg(strings.Join(errMsgs, ", "))
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
