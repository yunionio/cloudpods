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

package aws

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

func (self *SRegion) GetCacheParameters(name string) ([]SElasticacheParameter, error) {
	params := map[string]string{}
	if len(name) > 0 {
		params["CacheParameterGroupName"] = name
	}
	ret := []SElasticacheParameter{}
	for {
		result := struct {
			Marker     string                  `xml:"Marker"`
			Parameters []SElasticacheParameter `xml:"CacheNodeTypeSpecificParameters>CacheNodeTypeSpecificParameter"`
		}{}
		err := self.redisRequest("DescribeCacheParameters", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeCacheParameters")
		}
		ret = append(ret, result.Parameters...)
		if len(result.Marker) == 0 || len(result.Parameters) == 0 {
			break
		}
		params["Marker"] = result.Marker
	}
	return ret, nil
}

type SElasticacheParameter struct {
	multicloud.SElasticcacheParameterBase
	multicloud.AwsTags
	parameterGroup string

	AllowedValues        string `xml:"AllowedValues"`
	ChangeType           string `xml:"ChangeType"`
	DataType             string `xml:"DataType"`
	Description          string `xml:"Description"`
	IsModifiable         bool   `xml:"IsModifiable"`
	MinimumEngineVersion string `xml:"MinimumEngineVersion"`
	ParameterName        string `xml:"ParameterName"`
	ParameterValue       string `xml:"ParameterValue"`
	Source               string `xml:"Source"`
}

func (self *SElasticacheParameter) GetId() string {
	return fmt.Sprintf("%s/%s", self.parameterGroup, self.ParameterName)
}

func (self *SElasticacheParameter) GetName() string {
	return self.ParameterName
}

func (self *SElasticacheParameter) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticacheParameter) GetStatus() string {
	return api.ELASTIC_CACHE_PARAMETER_STATUS_AVAILABLE
}

func (self *SElasticacheParameter) GetParameterKey() string {
	return self.ParameterName
}

func (self *SElasticacheParameter) GetParameterValue() string {
	return self.ParameterValue
}

func (self *SElasticacheParameter) GetParameterValueRange() string {
	return self.AllowedValues
}

func (self *SElasticacheParameter) GetDescription() string {
	return self.Description
}

func (self *SElasticacheParameter) GetModifiable() bool {
	return self.IsModifiable
}

func (self *SElasticacheParameter) GetForceRestart() bool {
	return self.ChangeType == "requires-reboot"
}
