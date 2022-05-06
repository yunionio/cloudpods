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

package models

import (
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type Network struct {
	*models.SNetwork
}

func (el *Network) Copy() *Network {
	return &Network{
		SNetwork: el.SNetwork,
	}
}

type LoadbalancerNetwork struct {
	*models.SLoadbalancerNetwork

	Loadbalancer *Loadbalancer `json:"-"`
	Network      *Network      `json:"-"`
}

func (el *LoadbalancerNetwork) Copy() *LoadbalancerNetwork {
	return &LoadbalancerNetwork{
		SLoadbalancerNetwork: el.SLoadbalancerNetwork,
	}
}

type Loadbalancer struct {
	*models.SLoadbalancer

	LoadbalancerNetwork *LoadbalancerNetwork `json:"-"`
	Listeners           LoadbalancerListeners
	BackendGroups       LoadbalancerBackendGroups

	ListenAddress string
}

func (el *Loadbalancer) Copy() *Loadbalancer {
	return &Loadbalancer{
		SLoadbalancer: el.SLoadbalancer,
	}
}

func (lb *Loadbalancer) GetAddress() string {
	if lb.NetworkType == computeapi.LB_NETWORK_TYPE_VPC {
		return lb.ListenAddress
	}
	return lb.Address
}

type LoadbalancerListener struct {
	*models.SLoadbalancerListener

	loadbalancer *Loadbalancer
	certificate  *LoadbalancerCertificate
	rules        LoadbalancerListenerRules
}

func (el *LoadbalancerListener) Copy() *LoadbalancerListener {
	return &LoadbalancerListener{
		SLoadbalancerListener: el.SLoadbalancerListener,
	}
}

type LoadbalancerListenerRule struct {
	*models.SLoadbalancerListenerRule

	listener *LoadbalancerListener
}

func (el *LoadbalancerListenerRule) Copy() *LoadbalancerListenerRule {
	return &LoadbalancerListenerRule{
		SLoadbalancerListenerRule: el.SLoadbalancerListenerRule,
	}
}

type LoadbalancerBackendGroup struct {
	*models.SLoadbalancerBackendGroup

	Backends     LoadbalancerBackends
	loadbalancer *Loadbalancer
}

func (el *LoadbalancerBackendGroup) Copy() *LoadbalancerBackendGroup {
	return &LoadbalancerBackendGroup{
		SLoadbalancerBackendGroup: el.SLoadbalancerBackendGroup,
	}
}

type LoadbalancerBackend struct {
	*models.SLoadbalancerBackend

	backendGroup *LoadbalancerBackendGroup

	ConnectAddress string
	ConnectPort    int
}

func (el *LoadbalancerBackend) Copy() *LoadbalancerBackend {
	return &LoadbalancerBackend{
		SLoadbalancerBackend: el.SLoadbalancerBackend,
	}
}

func (lbbackend *LoadbalancerBackend) GetAddressPort() (addr string, port int) {
	backendGroup := lbbackend.backendGroup
	if backendGroup == nil {
		return
	}
	lb := backendGroup.loadbalancer
	if lb == nil {
		return
	}
	if lb.NetworkType == computeapi.LB_NETWORK_TYPE_VPC {
		return lbbackend.ConnectAddress, lbbackend.ConnectPort
	}
	return lbbackend.Address, lbbackend.Port
}

type LoadbalancerAcl struct {
	*models.SLoadbalancerAcl
}

func (el *LoadbalancerAcl) Copy() *LoadbalancerAcl {
	return &LoadbalancerAcl{
		SLoadbalancerAcl: el.SLoadbalancerAcl,
	}
}

type LoadbalancerCertificate struct {
	*models.SLoadbalancerCertificate
}

func (el *LoadbalancerCertificate) Copy() *LoadbalancerCertificate {
	return &LoadbalancerCertificate{
		SLoadbalancerCertificate: el.SLoadbalancerCertificate,
	}
}
