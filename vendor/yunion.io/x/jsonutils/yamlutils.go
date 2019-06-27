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

package jsonutils

import "github.com/ghodss/yaml"

func ParseYAML(str string) (JSONObject, error) {
	jsonBytes, err := yaml.YAMLToJSON([]byte(str))
	if err != nil {
		return nil, err
	}
	return Parse(jsonBytes)
}

func yamlString(obj JSONObject) string {
	yamlBytes, _ := yaml.JSONToYAML([]byte(obj.String()))
	return string(yamlBytes)
}

func (this *JSONValue) YAMLString() string {
	return yamlString(this)
}

func (this *JSONString) YAMLString() string {
	return yamlString(this)
}

func (this *JSONInt) YAMLString() string {
	return yamlString(this)
}

func (this *JSONFloat) YAMLString() string {
	return yamlString(this)
}

func (this *JSONBool) YAMLString() string {
	return yamlString(this)
}

func (this *JSONArray) YAMLString() string {
	return yamlString(this)
}

func (this *JSONDict) YAMLString() string {
	return yamlString(this)
}
