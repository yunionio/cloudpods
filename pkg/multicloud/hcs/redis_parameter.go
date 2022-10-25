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
	"fmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423027.html
type SElasticcacheParameter struct {
	multicloud.SElasticcacheParameterBase
	multicloud.HuaweiTags

	cacheDB *SElasticcache

	Description  string `json:"description"`
	ParamId      int64  `json:"param_id"`
	ParamName    string `json:"param_name"`
	ParamValue   string `json:"param_value"`
	DefaultValue string `json:"default_value"`
	ValueType    string `json:"value_type"`
	ValueRange   string `json:"value_range"`
}

func (self *SElasticcacheParameter) GetId() string {
	return fmt.Sprintf("%d", self.ParamId)
}

func (self *SElasticcacheParameter) GetName() string {
	return self.ParamName
}

func (self *SElasticcacheParameter) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.cacheDB.InstanceId, self.GetId())
}

func (self *SElasticcacheParameter) GetStatus() string {
	return api.ELASTIC_CACHE_PARAMETER_STATUS_AVAILABLE
}

func (self *SElasticcacheParameter) GetParameterKey() string {
	return self.ParamName
}

func (self *SElasticcacheParameter) GetParameterValue() string {
	return self.ParamValue
}

func (self *SElasticcacheParameter) GetParameterValueRange() string {
	return self.Description
}

func (self *SElasticcacheParameter) GetDescription() string {
	return self.ValueRange
}

func (self *SElasticcacheParameter) GetModifiable() bool {
	return true
}

func (self *SElasticcacheParameter) GetForceRestart() bool {
	return false
}
