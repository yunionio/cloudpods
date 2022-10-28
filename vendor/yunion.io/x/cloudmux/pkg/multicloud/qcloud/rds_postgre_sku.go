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

type SpecItemInfoList struct {
	SpecCode    string
	Version     string
	VersionName string
	Cpu         int
	Memory      int
	MaxStorage  int
	MinStorage  int
	Qps         int
	Pid         int
	Type        string
}

type SpecInfoList struct {
	Region           string
	Zone             string
	SpecItemInfoList []SpecItemInfoList
}

func (self *SRegion) DescribeProductConfig() ([]SpecInfoList, error) {
	resp, err := self.postgresRequest("DescribeProductConfig", map[string]string{})
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeProductConfig")
	}
	products := []SpecInfoList{}
	err = resp.Unmarshal(&products, "SpecInfoList")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return products, nil
}

func (self *SRegion) ListPostgreSQLSkus() ([]SDBInstanceSku, error) {
	skus := []SDBInstanceSku{}
	products, err := self.DescribeProductConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeProductConfig")
	}
	for _, product := range products {
		sku := SDBInstanceSku{
			Region:   self.Region,
			Zone1:    product.Zone,
			Engine:   api.DBINSTANCE_TYPE_POSTGRESQL,
			Status:   api.DBINSTANCE_SKU_AVAILABLE,
			Category: "双机高可用",
		}

		for _, spec := range product.SpecItemInfoList {
			sku.EngineVersion = spec.Version
			sku.Qps = spec.Qps
			sku.Cpu = spec.Cpu
			sku.MemoryMb = spec.Memory * 1024
			sku.StorageMax = spec.MaxStorage
			sku.StorageMin = spec.MinStorage
			sku.StorageStep = 10

			skus = append(skus, sku)
		}
	}
	return skus, nil
}
