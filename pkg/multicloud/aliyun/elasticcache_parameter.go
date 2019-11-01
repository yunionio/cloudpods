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

package aliyun

import (
	"fmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheParameter struct {
	multicloud.SElasticcacheParameterBase

	cacheDB *SElasticcache

	ParameterDescription string `json:"ParameterDescription"`
	ParameterValue       string `json:"ParameterValue"`
	ForceRestart         string `json:"ForceRestart"`
	CheckingCode         string `json:"CheckingCode"`
	ModifiableStatus     string `json:"ModifiableStatus"`
	ParameterName        string `json:"ParameterName"`
}

func (self *SElasticcacheParameter) GetId() string {
	return fmt.Sprintf("%s/%s", self.cacheDB.InstanceID, self.ParameterName)
}

func (self *SElasticcacheParameter) GetName() string {
	return self.ParameterName
}

func (self *SElasticcacheParameter) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheParameter) GetStatus() string {
	return api.ELASTIC_CACHE_PARAMETER_STATUS_AVAILABLE
}

func (self *SElasticcacheParameter) GetParameterKey() string {
	return self.ParameterName
}

func (self *SElasticcacheParameter) GetParameterValue() string {
	return self.ParameterValue
}

func (self *SElasticcacheParameter) GetParameterValueRange() string {
	return self.CheckingCode
}

func (self *SElasticcacheParameter) GetDescription() string {
	return self.ParameterDescription
}

func (self *SElasticcacheParameter) GetModifiable() bool {
	switch self.ModifiableStatus {
	case "true":
		return true
	default:
		return false
	}
}

func (self *SElasticcacheParameter) GetForceRestart() bool {
	switch self.ForceRestart {
	case "true":
		return true
	default:
		return false
	}
}
