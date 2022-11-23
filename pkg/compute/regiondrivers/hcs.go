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

package regiondrivers

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SHCSRegionDriver struct {
	SHuaWeiRegionDriver
}

func init() {
	driver := SHCSRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SHCSRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_HCS
}

func (self *SHCSRegionDriver) IsSupportedElasticcache() bool {
	return true
}

func (self *SHCSRegionDriver) IsSecurityGroupBelongVpc() bool {
	return true
}

func (self *SHCSRegionDriver) IsSupportedElasticcacheSecgroup() bool {
	return false
}

func (self *SHCSRegionDriver) GetMaxElasticcacheSecurityGroupCount() int {
	return 1
}
