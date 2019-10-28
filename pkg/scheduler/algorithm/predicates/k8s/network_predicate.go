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

package k8s

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
)

const (
	// k8s annotations for create pod
	YUNION_CNI_NETWORK_ANNOTATION = "cni.yunion.io/network"
	YUNION_CNI_IPADDR_ANNOTATION  = "cni.yunion.io/ip"
)

type NetworkPredicate struct {
	network string
	ipAddr  string
}

func (p *NetworkPredicate) Clone() IPredicate {
	return &NetworkPredicate{}
}

func (p *NetworkPredicate) Name() string {
	return "network"
}

func (p *NetworkPredicate) PreExecute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) bool {
	net, netCont := pod.Annotations[YUNION_CNI_NETWORK_ANNOTATION]
	ipAddr, ipCont := pod.Annotations[YUNION_CNI_IPADDR_ANNOTATION]
	p.network = net
	p.ipAddr = ipAddr
	return netCont || ipCont
}

func (p *NetworkPredicate) Execute(cli *kubernetes.Clientset, pod *v1.Pod, node *v1.Node, host *candidate.HostDesc) (bool, error) {
	hostNets := host.Networks
	if p.network != "" {
		err := p.checkByNetworks(hostNets)
		if err != nil {
			return false, err
		}
	}
	if p.ipAddr != "" {
		err := p.checkNetworksIP(p.ipAddr, hostNets)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (p NetworkPredicate) checkByNetworks(nets []*api.CandidateNetwork) error {
	if len(nets) == 0 {
		return fmt.Errorf("Network is empty")
	}
	errs := make([]error, 0)
	for _, net := range nets {
		err := p.checkByNetwork(net.SNetwork)
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (p NetworkPredicate) checkByNetwork(net *models.SNetwork) error {
	if net.GetPorts() <= 0 {
		return fmt.Errorf("Network %s no free IPs", net.Name)
	}
	if !(p.network == net.Name || p.network == net.Id) {
		return fmt.Errorf("Network %s:%s or id not match %s", net.Name, net.Id, p.network)
	}
	return nil
}

func (p NetworkPredicate) checkNetworksIP(ip string, nets []*api.CandidateNetwork) error {
	if len(nets) == 0 {
		return fmt.Errorf("Network is empty")
	}
	errs := make([]error, 0)
	for _, net := range nets {
		err := p.checkNetworkIP(ip, net.SNetwork)
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (p NetworkPredicate) checkNetworkIP(ip string, net *models.SNetwork) error {
	ipAddr, err := netutils.NewIPV4Addr(ip)
	if err != nil {
		return err
	}
	if ok := net.GetIPRange().Contains(ipAddr); !ok {
		return fmt.Errorf("Network %s not contains ip %s", net.Name, ip)
	}
	return nil
}
