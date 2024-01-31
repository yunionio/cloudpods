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

package esxi

import (
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type sFakeVpc struct {
	client *SESXiClient
}

func (vpc *sFakeVpc) GetId() string {
	return vpc.client.GetId()
}

func (vpc *sFakeVpc) GetName() string {
	return vpc.client.GetName()
}

func (vpc *sFakeVpc) GetGlobalId() string {
	return vpc.client.GetGlobalId()
}

func (vpc *sFakeVpc) GetCreatedAt() time.Time {
	return time.Time{}
}

func (vpc *sFakeVpc) GetDescription() string {
	return "fake vpc for esxi"
}

func (vpc *sFakeVpc) GetStatus() string {
	return "'"
}

func (vpc *sFakeVpc) Refresh() error {
	return nil
}

func (vpc *sFakeVpc) IsEmulated() bool {
	return true
}

func (vpc *sFakeVpc) GetSysTags() map[string]string {
	return nil
}

func (vpc *sFakeVpc) GetTags() (map[string]string, error) {
	return nil, nil
}

func (vpc *sFakeVpc) SetTags(tags map[string]string, replace bool) error {
	return errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetGlobalVpcId() string {
	return ""
}

func (vpc *sFakeVpc) IsSupportSetExternalAccess() bool {
	return false
}

func (vpc *sFakeVpc) GetExternalAccessMode() string {
	return ""
}

func (vpc *sFakeVpc) AttachInternetGateway(igwId string) error {
	return errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetRegion() cloudprovider.ICloudRegion {
	return vpc.client
}

func (vpc *sFakeVpc) GetIsDefault() bool {
	return true
}

func (vpc *sFakeVpc) GetCidrBlock() string {
	return ""
}

func (vpc *sFakeVpc) GetCidrBlock6() string {
	return ""
}

func (vpc *sFakeVpc) CreateIWire(opts *cloudprovider.SWireCreateOptions) (cloudprovider.ICloudWire, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) Delete() error {
	return errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	wires, err := vpc.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	for i := range wires {
		if wires[i].GetGlobalId() == wireId {
			return wires[i], nil
		}
	}
	return nil, errors.ErrNotFound
}

func (vpc *sFakeVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) CreateINatGateway(opts *cloudprovider.NatGatewayCreateOptions) (cloudprovider.ICloudNatGateway, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetICloudVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetICloudAccepterVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetICloudVpcPeeringConnectionById(id string) (cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) CreateICloudVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) AcceptICloudVpcPeeringConnection(id string) error {
	return errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetAuthorityOwnerId() string {
	return ""
}

func (vpc *sFakeVpc) ProposeJoinICloudInterVpcNetwork(opts *cloudprovider.SVpcJointInterVpcNetworkOption) error {
	return errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetICloudIPv6Gateways() ([]cloudprovider.ICloudIPv6Gateway, error) {
	return nil, errors.ErrNotSupported
}

func (vpc *sFakeVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	nets, err := vpc.client.GetNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetNetworks")
	}
	wires := make([]cloudprovider.ICloudWire, len(nets))
	for i := range nets {
		wires[i] = &sWire{
			network: nets[i],
			client:  vpc.client,
		}
	}
	return wires, nil
}
