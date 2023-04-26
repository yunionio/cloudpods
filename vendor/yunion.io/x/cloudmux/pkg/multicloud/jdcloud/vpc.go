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

package jdcloud

import (
	"fmt"

	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vpc/models"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVpc struct {
	multicloud.SVpc
	JdcloudTags
	models.Vpc

	secgroups []cloudprovider.ICloudSecurityGroup
	iwires    []cloudprovider.ICloudWire

	region *SRegion
}

func (v *SVpc) GetId() string {
	return v.VpcId
}

func (v *SVpc) GetName() string {
	return v.VpcName
}

func (v *SVpc) GetGlobalId() string {
	return v.GetId()
}

func (v *SVpc) GetStatus() string {
	return api.VPC_STATUS_AVAILABLE
}

func (v *SVpc) Refresh() error {
	return nil
}

func (v *SVpc) IsEmulated() bool {
	return false
}

func (self *SVpc) IsPublic() bool {
	return false
}

func (v *SVpc) GetRegion() cloudprovider.ICloudRegion {
	return v.region
}

func (v *SVpc) GetIsDefault() bool {
	return false
}

func (v *SVpc) GetCidrBlock() string {
	return ""
}

func (v *SVpc) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if v.iwires != nil {
		return v.iwires, nil
	}
	wire := SWire{
		vpc: v,
	}
	v.iwires = []cloudprovider.ICloudWire{&wire}
	return v.iwires, nil
}

func (v *SVpc) GetWire() *SWire {
	return &SWire{
		vpc: v,
	}
}

func (v *SVpc) fetchSecurityGroups() error {
	secgroups := make([]SSecurityGroup, 0)
	n := 1
	for {
		parts, total, err := v.region.GetSecurityGroups(v.GetId(), []string{}, n, 100)
		if err != nil {
			return err
		}
		secgroups = append(secgroups, parts...)
		if len(secgroups) >= total {
			break
		}
		n++
	}
	v.secgroups = make([]cloudprovider.ICloudSecurityGroup, len(secgroups))
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].vpc = v
		v.secgroups[i] = &secgroups[i]
	}
	return nil
}

func (v *SVpc) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	if v.secgroups == nil {
		err := v.fetchSecurityGroups()
		if err != nil {
			return nil, err
		}
	}
	return v.secgroups, nil
}

func (v *SVpc) GetIRouteTables() ([]cloudprovider.ICloudRouteTable, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (v *SVpc) GetIRouteTableById(routeTableId string) (cloudprovider.ICloudRouteTable, error) {
	return nil, nil
}

func (v *SVpc) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (v *SVpc) GetIWireById(wireId string) (cloudprovider.ICloudWire, error) {
	iwires, err := v.GetIWires()
	if err != nil {
		return nil, err
	}
	for i := range iwires {
		if iwires[i].GetGlobalId() == wireId {
			return iwires[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (r *SRegion) GetVpcs(pageNumber, pageSize int) (vpcs []SVpc, total int, err error) {
	req := apis.NewDescribeVpcsRequestWithAllParams(r.ID, &pageNumber, &pageSize, nil)
	client := client.NewVpcClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeVpcs(req)
	if err != nil {
		return
	}
	if resp.Error.Code >= 400 {
		err = errors.Error(resp.Error.Message)
		return
	}
	total = resp.Result.TotalCount
	vpcs = make([]SVpc, 0, len(resp.Result.Vpcs))
	for i := range resp.Result.Vpcs {
		vpcs = append(vpcs, SVpc{
			Vpc:    resp.Result.Vpcs[i],
			region: r,
		})
	}
	return
}

func (r *SRegion) GetVpcById(id string) (*SVpc, error) {
	req := apis.NewDescribeVpcRequest(r.ID, id)
	client := client.NewVpcClient(r.getCredential())
	client.Logger = Logger{debug: r.client.debug}
	resp, err := client.DescribeVpc(req)
	if err != nil {
		return nil, err
	}
	if resp.Error.Code >= 400 {
		return nil, fmt.Errorf(resp.Error.Message)
	}
	return &SVpc{
		region: r,
		Vpc:    resp.Result.Vpc,
	}, nil
}
