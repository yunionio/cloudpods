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

package apsara

import (
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SDBInstanceParameter struct {
	instance *SDBInstance

	ParameterDescription string
	ParameterValue       string
	ParameterName        string
}

func (param *SDBInstanceParameter) GetGlobalId() string {
	return param.ParameterName
}

func (param *SDBInstanceParameter) GetKey() string {
	return param.ParameterName
}

func (param *SDBInstanceParameter) GetValue() string {
	return param.ParameterValue
}

func (param *SDBInstanceParameter) GetDescription() string {
	return param.ParameterDescription
}

func (region *SRegion) GetDBInstanceParameters(instanceId string) ([]SDBInstanceParameter, error) {
	params := map[string]string{
		"RegionId":     region.RegionId,
		"DBInstanceId": instanceId,
		"ClientToken":  utils.GenRequestId(20),
	}

	body, err := region.rdsRequest("DescribeParameters", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeParameters")
	}
	parameters1 := []SDBInstanceParameter{}
	err = body.Unmarshal(&parameters1, "ConfigParameters", "DBInstanceParameter")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal.ConfigParameters")
	}
	parameters2 := []SDBInstanceParameter{}
	err = body.Unmarshal(&parameters1, "RunningParameters", "DBInstanceParameter")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal.RunningParameters")
	}
	return append(parameters1, parameters2...), nil
}
