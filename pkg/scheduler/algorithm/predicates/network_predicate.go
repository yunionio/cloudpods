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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/plugin"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

// NetworkPredicate will filter the current network information with
// the specified scheduling information to match, if not specified will
// randomly match the available network resources.
type NetworkPredicate struct {
	BasePredicate
	plugin.BasePlugin
	NetworkNicCountGetter INetworkNicCountGetter
	networkFreePortCount  map[string]int
}

func NewNetworkPredicate() *NetworkPredicate {
	return &NetworkPredicate{
		NetworkNicCountGetter: nil,
		networkFreePortCount:  make(map[string]int),
	}
}

func NewNetworkPredicateWithNicCounter() *NetworkPredicate {
	return NewNetworkPredicate().WithNetworkCountGetter(models.GetNetworkManager())
}

func (p *NetworkPredicate) WithNetworkCountGetter(getter INetworkNicCountGetter) *NetworkPredicate {
	p.NetworkNicCountGetter = getter
	return p
}

func (p *NetworkPredicate) Name() string {
	return "host_network"
}

func (p *NetworkPredicate) Clone() core.FitPredicate {
	return &NetworkPredicate{
		NetworkNicCountGetter: p.NetworkNicCountGetter,
		networkFreePortCount:  p.networkFreePortCount,
	}
}

type INetworkNicCountGetter interface {
	GetTotalNicCount([]string) (map[string]int, error)
}

func (p *NetworkPredicate) PreExecute(ctx context.Context, u *core.Unit, cs []core.Candidater) (bool, error) {
	data := u.SchedData()

	if data.ResetCpuNumaPin {
		return false, nil
	}

	if len(data.Networks) == 0 {
		return false, nil
	}
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

func IsNetworksAvailable(ctx context.Context, c core.Candidater, data *api.SchedInfo, req *computeapi.NetworkConfig, networks []*api.CandidateNetwork, netTypes []computeapi.TNetworkType, getFreePort func(string) int) (int, []core.PredicateFailureReason) {
	var fullErrMsgs []core.PredicateFailureReason
	var freeCnt int

	if len(networks) == 0 {
		return 0, []core.PredicateFailureReason{FailReason{Reason: ErrNoAvailableNetwork}}
	}

	ovnCapable := c.Getter().OvnCapable()
	ovnNetworks := []*api.CandidateNetwork{}
	for i := len(networks) - 1; i >= 0; i -= 1 {
		net := networks[i]
		if net.Provider == computeapi.CLOUD_PROVIDER_ONECLOUD || net.Provider == computeapi.CLOUD_PROVIDER_CLOUDPODS {
			networks = append(networks[:i], networks[i+1:]...)
			ovnNetworks = append(ovnNetworks, net)
		}
	}

	checkNets := func(tmpNets []*api.CandidateNetwork) {
		for _, n := range tmpNets {
			if errMsg := IsNetworkAvailable(ctx, c, data, req, n, netTypes, getFreePort); errMsg != nil {
				fullErrMsgs = append(fullErrMsgs, errMsg)
			} else {
				freeCnt = freeCnt + getFreePort(n.GetId())
			}
		}
	}

	checkNets(networks)

	if ovnCapable {
		checkNets(ovnNetworks)
	}

	// reuse network
	if data.ReuseNetwork {
		return freeCnt, nil
	}

	if freeCnt <= 0 {
		return freeCnt, fullErrMsgs
	}
	if freeCnt < data.Count {
		fullErrMsgs = append(fullErrMsgs, FailReason{
			Reason: fmt.Sprintf("total random ports not enough, free: %d, required: %d", freeCnt, data.Count),
			Type:   NetworkFreeCount,
		})
	}
	return freeCnt, nil
}

func checkSriovNic(
	c core.Candidater, netWireId string, dev *computeapi.IsolatedDeviceConfig,
) error {
	if dev.WireId != "" {
		if dev.WireId != netWireId {
			return fmt.Errorf("Network wire not matched sriov nic wire %s != %s", netWireId, dev.WireId)
		} else {
			return nil
		}
	}
	getter := c.Getter()
	devs := getter.UnusedIsolatedDevicesByModelAndWire(dev.Model, netWireId)
	if len(devs) == 0 {
		return fmt.Errorf("Network wire no sriov nic available")
	}
	return nil
}

func IsNetworkAvailable(
	ctx context.Context,
	c core.Candidater, data *api.SchedInfo,
	req *computeapi.NetworkConfig, n *api.CandidateNetwork,
	netTypes []computeapi.TNetworkType, getFreePort func(string) int,
) core.PredicateFailureReason {
	address := req.Address
	private := req.Private
	exit := req.Exit
	wire := req.Wire

	isMatchServerType := func(network *models.SNetwork) bool {
		return computeapi.IsInNetworkTypes(network.ServerType, netTypes)
	}

	isMigrate := func() bool {
		return len(data.HostId) > 0
	}

	if isExit := n.IsExitNetwork(); isExit && isExit != exit {
		return FailReason{
			Reason: fmt.Sprintf("%v(%v): %s", n.Name, n.Id, ErrExitIsNotMatch),
		}
	}

	if getFreePort == nil {
		getFreePort = c.Getter().GetFreePort
	}

	if !(getFreePort(n.GetId()) > 0 || isMigrate()) {
		return FailReason{
			Reason: fmt.Sprintf("%v(%v): ports use up", n.Name, n.Id),
			Type:   NetworkPort,
		}
	}

	if wire != "" && !utils.HasPrefix(wire, n.WireId) {
		_wire, _ := n.GetWire()
		if !utils.HasPrefix(wire, _wire.GetName()) {
			return FailReason{
				Reason: fmt.Sprintf("Wire %s != %s", wire, n.WireId),
				Type:   NetworkWire,
			}
		}
	}

	if req.SriovDevice != nil && n.VpcId == computeapi.DEFAULT_VPC_ID {
		err := checkSriovNic(c, n.WireId, req.SriovDevice)
		if err != nil {
			return FailReason{
				Reason: err.Error(),
				Type:   NetworkWire,
			}
		}
	}

	if n.IsPublic && n.PublicScope == string(rbacscope.ScopeSystem) {
		// system-wide share
	} else if n.IsPublic && n.PublicScope == string(rbacscope.ScopeDomain) && (n.DomainId == data.Domain || utils.IsInStringArray(data.Domain, n.GetSharedDomains())) {
		// domain-wide share
	} else if n.PublicScope == string(rbacscope.ScopeProject) && utils.IsInStringArray(data.Project, n.GetSharedProjects()) {
		// project-wide share
	} else if n.ProjectId == data.Project {
		// owner
	} else if db.IsAdminAllowGet(ctx, data.UserCred, n) {
		// system admin, can do anything
	} else if db.IsDomainAllowGet(ctx, data.UserCred, n) && data.UserCred.GetProjectDomainId() == n.DomainId {
		// domain admin, can do anything with domain network
	} else {
		return FailReason{
			Reason: fmt.Sprintf("Network %s not accessible", n.Name),
			Type:   NetworkOwnership,
		}
	}

	if private && n.IsPublic {
		return FailReason{
			Reason: fmt.Sprintf("Network %s is public", n.Name),
			Type:   NetworkPublic,
		}
	}

	if req.Network != "" {
		if !(req.Network == n.GetId() || req.Network == n.GetName()) {
			return FailReason{
				Reason: fmt.Sprintf("%v(%v): id/name not matched", n.Name, n.Id),
				Type:   NetworkMatch,
			}
		}
	} else {
		if !isMatchServerType(n.SNetwork) {
			return FailReason{
				Reason: fmt.Sprintf("Network %s type %s match", n.Name, n.ServerType),
				Type:   NetworkTypeMatch,
			}
		}

	}

	if req.Network == "" && n.IsAutoAlloc.IsFalse() {
		return FailReason{Reason: fmt.Sprintf("Network %s is not auto alloc", n.Name), Type: NetworkPrivate}
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

	if err := checkAddress(address, n.SNetwork); err != nil {
		return FailReason{Reason: err.Error(), Type: NetworkRange}
	}

	return nil
}

func (p *NetworkPredicate) GetNetworkTypes(u *core.Unit, specifyType computeapi.TNetworkType) []computeapi.TNetworkType {
	netTypes := []computeapi.TNetworkType{}
	driver := p.GetHypervisorDriver(u)
	if driver != nil {
		netTypes = driver.GetRandomNetworkTypes()
	}
	if len(specifyType) > 0 {
		netTypes = []computeapi.TNetworkType{specifyType}
	}
	return netTypes
}

func (p *NetworkPredicate) Execute(ctx context.Context, u *core.Unit, c core.Candidater) (bool, []core.PredicateFailureReason, error) {
	h := NewPredicateHelper(p, u, c)

	getter := c.Getter()
	networks := getter.Networks()
	d := u.SchedData()

	getFreePort := func(id string) int {
		if _, ok := p.networkFreePortCount[id]; ok {
			return p.networkFreePortCount[id] - c.Getter().GetPendingUsage().NetUsage.Get(id)
		}
		return c.Getter().GetFreePort(id)
	}

	for _, reqNet := range d.Networks {
		netTypes := p.GetNetworkTypes(u, reqNet.NetType)
		freePortCnt, errs := IsNetworksAvailable(ctx, c, d, reqNet, networks, netTypes, getFreePort)
		if len(errs) > 0 {
			h.ExcludeByErrors(errs)
			return h.GetResult()
		}
		if !d.ReuseNetwork {
			h.SetCapacity(int64(freePortCnt))
		}
	}

	return h.GetResult()
}
