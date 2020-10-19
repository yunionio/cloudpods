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

package multicloud

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SVpc struct {
	SResourceBase
}

func (self *SVpc) GetINatGateways() ([]cloudprovider.ICloudNatGateway, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetINatGateways")
}

func (self *SVpc) GetICloudVpcPeeringConnections() ([]cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudVpcPeeringConnections")
}

func (self *SVpc) GetICloudVpcPeeringConnectionById(id string) (cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudVpcPeeringConnectionById")
}

func (self *SVpc) CreateICloudVpcPeeringConnection(opts *cloudprovider.VpcPeeringConnectionCreateOptions) (cloudprovider.ICloudVpcPeeringConnection, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateICloudVpcPeeringConnection")
}

func (self *SVpc) AcceptICloudVpcPeeringConnection(id string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "AcceptICloudVpcPeeringConnection")
}

func (self *SVpc) GetAuthorityOwnerId() string {
	return ""
}

func (self *SVpc) CreateRouteToVpcPeeringConnection(cidrBlock, peerId string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateRouteToVpcPeeringConnection")
}

func (self *SVpc) DeleteVpcPeeringConnectionRoute(vpcPeeringConnectionId string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "DeleteVpcPeeringConnectionRoute")
}
