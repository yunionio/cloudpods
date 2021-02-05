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

import "yunion.io/x/pkg/errors"

type SDBInstanceParameter struct {
	instance *SDBInstance

	AllowedValues  string `xml:"AllowedValues"`
	ApplyType      string `xml:"ApplyType"`
	DataType       string `xml:"DataType"`
	Description    string `xml:"Description"`
	ApplyMethod    string `xml:"ApplyMethod"`
	ParameterName  string `xml:"ParameterName"`
	Source         string `xml:"Source"`
	IsModifiable   bool   `xml:"IsModifiable"`
	ParameterValue string `xml:"ParameterValue"`
}

type SDBInstanceParameters struct {
	Parameters []SDBInstanceParameter `xml:"Parameters>Parameter"`
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
	return param.Description
}

func (region *SRegion) GetDBInstanceParameters(name string) ([]SDBInstanceParameter, error) {
	param := map[string]string{"DBParameterGroupName": name}
	parameters := SDBInstanceParameters{}
	err := region.rdsRequest("DescribeDBParameters", param, &parameters)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeDBParameters")
	}
	return parameters.Parameters, nil
}
