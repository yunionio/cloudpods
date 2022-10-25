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

package hcs

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	HUAWEI_NAT_SKU_SMALL  = "Small"
	HUAWEI_NAT_SKU_MIDDLE = "Middle"
	HUAWEI_NAT_SKU_LARGE  = "Large"
	HUAWEI_NAT_SKU_XLARGE = "XLarge"
)

type SNatSku struct {
	spec string
}

func (self *SNatSku) GetName() string {
	return self.spec
}

func (self *SNatSku) GetPps() int {
	return 1000000
}

func (self *SNatSku) GetConns() int {
	switch self.spec {
	case HUAWEI_NAT_SKU_SMALL:
		return 10000
	case HUAWEI_NAT_SKU_MIDDLE:
		return 50000
	case HUAWEI_NAT_SKU_LARGE:
		return 200000
	case HUAWEI_NAT_SKU_XLARGE:
		return 1000000
	}
	return 0
}

func (self *SNatSku) GetThroughput() int {
	return 10
}

func (self *SNatSku) GetDesc() string {
	switch self.spec {
	case HUAWEI_NAT_SKU_SMALL:
		return "小型"
	case HUAWEI_NAT_SKU_MIDDLE:
		return "中型"
	case HUAWEI_NAT_SKU_LARGE:
		return "大型"
	case HUAWEI_NAT_SKU_XLARGE:
		return "超大型"
	}
	return ""
}

func (self *SNatSku) GetPostpaidStatus() string {
	return api.NAT_SKU_AVAILABLE
}

func (self *SNatSku) GetPrepaidStatus() string {
	return api.NAT_SKU_SOLDOUT
}

func (self *SNatSku) GetGlobalId() string {
	return self.spec
}

func (self *SRegion) GetICloudNatSkus() ([]cloudprovider.ICloudNatSku, error) {
	ret := []cloudprovider.ICloudNatSku{}
	for _, spec := range []string{
		HUAWEI_NAT_SKU_SMALL,
		HUAWEI_NAT_SKU_MIDDLE,
		HUAWEI_NAT_SKU_LARGE,
		HUAWEI_NAT_SKU_XLARGE,
	} {
		sku := &SNatSku{spec: spec}
		ret = append(ret, sku)
	}
	return ret, nil
}
