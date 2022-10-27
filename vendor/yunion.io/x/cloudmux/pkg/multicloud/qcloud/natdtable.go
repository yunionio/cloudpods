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

package qcloud

import (
	"fmt"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDTable struct {
	multicloud.SResourceBase
	QcloudTags
	nat *SNatGateway

	Eip         string
	NatId       string
	Description string
	UniqVpcId   string
	Proto       string
	Pport       int
	Eport       int
	Owner       string
	VpcId       int
	PipType     int
	Pip         string
	UniqNatId   string
	CreateTime  time.Time
}

func (table *SDTable) GetName() string {
	if len(table.Description) > 0 {
		return table.Description
	}
	return fmt.Sprintf("%s/%s/%d", table.Eip, table.Proto, table.Eport)
}

func (table *SDTable) GetId() string {
	return fmt.Sprintf("%s/%s/%d", table.NatId, table.Eip, table.Eport)
}

func (table *SDTable) GetGlobalId() string {
	return table.GetId()
}

func (table *SDTable) GetStatus() string {
	return api.NAT_STAUTS_AVAILABLE
}

func (table *SDTable) GetExternalIp() string {
	return table.Eip
}

func (table *SDTable) GetExternalPort() int {
	return table.Eport
}

func (table *SDTable) GetInternalIp() string {
	return table.Pip
}

func (table *SDTable) GetInternalPort() int {
	return table.Pport
}

func (table *SDTable) GetIpProtocol() string {
	return table.Proto
}

func (table *SDTable) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetDTables(natId, vpcId string) ([]SDTable, error) {
	param := map[string]string{}
	param["vpcId"] = vpcId
	param["natId"] = natId

	body, err := region.vpc2017Request("GetDnaptRule", param)
	if err != nil {
		return nil, err
	}
	tables := []SDTable{}
	err = body.Unmarshal(&tables, "data", "detail")
	if err != nil {
		return nil, err
	}
	return tables, nil
}
