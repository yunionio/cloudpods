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

	"github.com/aws/aws-sdk-go/service/elasticache"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

func (region *SRegion) DescribeCacheParameters(parameterGroupId string) ([]*elasticache.Parameter, error) {
	ecClient, err := region.getAwsElasticacheClient()
	if err != nil {
		return nil, errors.Wrap(err, "client.getAwsElasticacheClient")
	}

	input := elasticache.DescribeCacheParametersInput{}
	if len(parameterGroupId) > 0 {
		input.CacheParameterGroupName = &parameterGroupId
	}

	marker := ""
	maxrecords := (int64)(50)
	input.MaxRecords = &maxrecords

	parameters := []*elasticache.Parameter{}
	for {
		if len(marker) >= 0 {
			input.Marker = &marker
		}
		out, err := ecClient.DescribeCacheParameters(&input)
		if err != nil {
			return nil, errors.Wrap(err, "ecClient.DescribeCacheParameters")
		}
		parameters = append(parameters, out.Parameters...)

		if out.Marker != nil && len(*out.Marker) > 0 {
			marker = *out.Marker
		} else {
			break
		}
	}

	return parameters, nil
}

type SElasticacheParameter struct {
	multicloud.SElasticcacheParameterBase
	AwsTags
	parameterGroup string
	parameter      *elasticache.Parameter
}

func (self *SElasticacheParameter) GetId() string {
	return fmt.Sprintf("%s/%s", self.parameterGroup, *self.parameter.ParameterName)
}

func (self *SElasticacheParameter) GetName() string {
	return *self.parameter.ParameterName
}

func (self *SElasticacheParameter) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticacheParameter) GetStatus() string {
	return api.ELASTIC_CACHE_PARAMETER_STATUS_AVAILABLE
}

func (self *SElasticacheParameter) GetParameterKey() string {
	if self.parameter == nil || self.parameter.ParameterName == nil {
		return ""
	}
	return *self.parameter.ParameterName
}

func (self *SElasticacheParameter) GetParameterValue() string {
	if self.parameter == nil || self.parameter.ParameterValue == nil {
		return ""
	}
	return *self.parameter.ParameterValue
}

func (self *SElasticacheParameter) GetParameterValueRange() string {
	if self.parameter == nil || self.parameter.AllowedValues == nil {
		return ""
	}
	return *self.parameter.AllowedValues
}

func (self *SElasticacheParameter) GetDescription() string {
	if self.parameter == nil || self.parameter.Description == nil {
		return ""
	}
	return *self.parameter.Description
}

func (self *SElasticacheParameter) GetModifiable() bool {
	if self.parameter == nil || self.parameter.IsModifiable == nil {
		return *self.parameter.IsModifiable
	}
	return false
}

func (self *SElasticacheParameter) GetForceRestart() bool {
	if self.parameter == nil || self.parameter.ChangeType == nil {
		return false
	}
	if *self.parameter.ChangeType == "requires-reboot" {
		return true
	}
	return false
}
