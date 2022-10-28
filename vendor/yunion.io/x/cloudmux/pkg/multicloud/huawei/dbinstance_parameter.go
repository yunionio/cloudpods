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

package huawei

type SDBInstanceParameter struct {
	instance *SDBInstance

	Name            string
	Value           string
	RestartRequired bool
	Readonly        bool
	ValueRange      string
	Type            string
	Description     string
}

func (region *SRegion) GetDBInstanceParameters(dbinstanceId string) ([]SDBInstanceParameter, error) {
	params := map[string]string{
		"instance_id": dbinstanceId,
	}
	paramters := []SDBInstanceParameter{}
	err := doListAll(region.ecsClient.DBInstance.ListParameters, params, &paramters)
	if err != nil {
		return nil, err
	}
	return paramters, nil
}

func (param *SDBInstanceParameter) GetGlobalId() string {
	return param.Name
}

func (param *SDBInstanceParameter) GetKey() string {
	return param.Name
}

func (param *SDBInstanceParameter) GetValue() string {
	return param.Value
}

func (param *SDBInstanceParameter) GetDescription() string {
	return param.Description
}
