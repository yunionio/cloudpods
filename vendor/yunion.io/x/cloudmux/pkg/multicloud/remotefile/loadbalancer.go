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

package remotefile

import (
	"context"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancer struct {
	multicloud.SLoadbalancerBase
	SResourceBase

	region       *SRegion
	RegionId     string
	Address      string
	AddressType  string
	NetworkType  string
	VpcId        string
	ZoneId       string
	Zone1Id      string
	InstanceType string
	ChargeType   string
	Bandwidth    int
	NetworkIds   []string
}

func (self *SLoadbalancer) GetAddress() string {
	return self.Address
}

func (self *SLoadbalancer) GetAddressType() string {
	return self.AddressType
}

func (self *SLoadbalancer) GetNetworkType() string {
	return self.NetworkType
}

func (self *SLoadbalancer) GetNetworkIds() []string {
	return self.NetworkIds
}

func (self *SLoadbalancer) GetVpcId() string {
	return self.VpcId
}

func (self *SLoadbalancer) GetZoneId() string {
	return self.ZoneId
}

func (self *SLoadbalancer) GetZone1Id() string {
	return self.Zone1Id
}

func (self *SLoadbalancer) GetLoadbalancerSpec() string {
	return self.InstanceType
}

func (self *SLoadbalancer) GetChargeType() string {
	return self.ChargeType
}

func (self *SLoadbalancer) GetEgressMbps() int {
	return self.Bandwidth
}

func (self *SLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SLoadbalancer) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) Start() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	return nil, cloudprovider.ErrNotSupported
}
