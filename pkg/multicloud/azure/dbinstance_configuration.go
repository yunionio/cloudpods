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

package azure

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"
)

type SDBInstanceConfiguration struct {
	Properties SDBInstanceConfigurationProperties `json:"properties"`
	ID         string                             `json:"id"`
	Name       string                             `json:"name"`
	Type       string                             `json:"type"`
}

type SDBInstanceConfigurationProperties struct {
	Value         string `json:"value"`
	Description   string `json:"description"`
	DefaultValue  string `json:"defaultValue"`
	DataType      string `json:"dataType"`
	AllowedValues string `json:"allowedValues"`
	Source        string `json:"source"`
}

func (self *SRegion) ListDBInstanceConfiguration(Id string) ([]SDBInstanceConfiguration, error) {
	type configs struct {
		Value []SDBInstanceConfiguration
	}
	result := configs{}
	err := self.get(fmt.Sprintf("%s/configurations", Id), url.Values{}, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "get(%s/configurations)", Id)
	}
	return result.Value, nil
}

func (param *SDBInstanceConfiguration) GetGlobalId() string {
	return strings.ToLower(param.ID)
}

func (param *SDBInstanceConfiguration) GetKey() string {
	return strings.ToLower(param.Name)
}

func (param *SDBInstanceConfiguration) GetValue() string {
	return param.Properties.Value
}

func (param *SDBInstanceConfiguration) GetDescription() string {
	return param.Properties.Description
}
