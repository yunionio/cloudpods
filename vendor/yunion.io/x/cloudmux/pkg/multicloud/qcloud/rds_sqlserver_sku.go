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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
)

type SqlServerSpecInfoList struct {
	Cpu             int
	MachineType     string
	MachineTypeName string
	MaxStorage      int
	Memory          int
	MinStorage      int
	PayModeStatus   string
	Pid             int
	PostPid         []int
	Qps             int
	RoPid           int
	SpecId          int
	SuitInfo        string
	Version         string
	VersionName     string
}

func (self *SRegion) DescribeSqlServerProductConfig(zoneId string) ([]SqlServerSpecInfoList, error) {
	resp, err := self.sqlserverRequest("DescribeProductConfig", map[string]string{"Zone": zoneId})
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeProductConfig")
	}
	specs := []SqlServerSpecInfoList{}
	err = resp.Unmarshal(&specs, "SpecInfoList")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return specs, nil
}

func (self *SRegion) ListSQLServerSkus() ([]SDBInstanceSku, error) {
	zones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIZones")
	}
	skus := []SDBInstanceSku{}
	for _, zone := range zones {
		products, err := self.DescribeSqlServerProductConfig(zone.GetId())
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeSqlServerProductConfig")
		}
		for _, product := range products {
			sku := SDBInstanceSku{
				Region:        self.Region,
				Zone1:         zone.GetId(),
				Engine:        api.DBINSTANCE_TYPE_SQLSERVER,
				EngineVersion: product.Version,
			}
			skus = append(skus, sku)
		}
	}
	return skus, nil
}
