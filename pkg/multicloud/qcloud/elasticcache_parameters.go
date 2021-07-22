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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheParameters struct {
	multicloud.SElasticcacheParameterBase
	multicloud.QcloudTags

	cacheDB *SElasticcache

	InstanceEnumParam    []SElasticcacheParameter `json:"InstanceEnumParam"`
	InstanceIntegerParam []SElasticcacheParameter `json:"InstanceIntegerParam"`
	InstanceTextParam    []SElasticcacheParameter `json:"InstanceTextParam"`
	RequestID            string                   `json:"RequestId"`
	TotalCount           int64                    `json:"TotalCount"`
}

type SElasticcacheParameter struct {
	multicloud.SElasticcacheParameterBase
	multicloud.QcloudTags

	cacheDB *SElasticcache

	CurrentValue string   `json:"CurrentValue"`
	DefaultValue string   `json:"DefaultValue"`
	EnumValue    []string `json:"EnumValue,omitempty"`
	NeedRestart  string   `json:"NeedRestart"`
	ParamName    string   `json:"ParamName"`
	Tips         string   `json:"Tips"`
	ValueType    string   `json:"ValueType"`
	Max          *string  `json:"Max,omitempty"`
	Min          *string  `json:"Min,omitempty"`
	TextValue    []string `json:"TextValue,omitempty"`
}

func (self *SElasticcacheParameter) GetId() string {
	return fmt.Sprintf("%s/%s", self.cacheDB.InstanceID, self.ParamName)
}

func (self *SElasticcacheParameter) GetName() string {
	return self.ParamName
}

func (self *SElasticcacheParameter) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheParameter) GetStatus() string {
	return api.ELASTIC_CACHE_PARAMETER_STATUS_AVAILABLE
}

func (self *SElasticcacheParameter) GetParameterKey() string {
	return self.ParamName
}

func (self *SElasticcacheParameter) GetParameterValue() string {
	return self.CurrentValue
}

func (self *SElasticcacheParameter) GetParameterValueRange() string {
	return fmt.Sprintf("%s", self.EnumValue)
}

func (self *SElasticcacheParameter) GetDescription() string {
	return self.Tips
}

func (self *SElasticcacheParameter) GetModifiable() bool {
	return true
}

func (self *SElasticcacheParameter) GetForceRestart() bool {
	return self.NeedRestart == "true"
}
