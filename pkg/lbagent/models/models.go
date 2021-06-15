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
	"yunion.io/x/onecloud/pkg/mcclient/models"
)

type IModel interface {
}

type Network struct {
	*models.Network
}

type LoadbalancerNetwork struct {
	*models.LoadbalancerNetwork

	Loadbalancer *Loadbalancer
	Network      *Network
}

type Loadbalancer struct {
	*models.Loadbalancer

	LoadbalancerNetwork *LoadbalancerNetwork
	Listeners           LoadbalancerListeners
	BackendGroups       LoadbalancerBackendGroups

	ListenAddress string
}

func (lb *Loadbalancer) GetAddress() string {
	if lb.NetworkType == computeapi.LB_NETWORK_TYPE_VPC {
		return lb.ListenAddress
	}
	return lb.Address
}

type LoadbalancerListener struct {
	*models.LoadbalancerListener

	loadbalancer *Loadbalancer
	certificate  *LoadbalancerCertificate
	rules        LoadbalancerListenerRules
}

type LoadbalancerListenerRule struct {
	*models.LoadbalancerListenerRule

	listener *LoadbalancerListener
}

type LoadbalancerBackendGroup struct {
	*models.LoadbalancerBackendGroup

	Backends     LoadbalancerBackends
	loadbalancer *Loadbalancer
}

type LoadbalancerBackend struct {
	*models.LoadbalancerBackend

	backendGroup *LoadbalancerBackendGroup

	ConnectAddress string
	ConnectPort    int
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
	*models.LoadbalancerAcl
}

type LoadbalancerCertificate struct {
	*models.LoadbalancerCertificate
}
