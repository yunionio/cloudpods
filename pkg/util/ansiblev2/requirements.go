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

package ansiblev2

import "gopkg.in/yaml.v2"

type RoleSource struct {
	Name    string
	Src     string
	Version string
}

func (rs *RoleSource) MarshalYAML() (interface{}, error) {
	r := map[string]string{
		"src": rs.Src,
	}
	if len(rs.Name) > 0 {
		r["name"] = rs.Name
	}
	if len(rs.Version) > 0 {
		r["version"] = rs.Version
	}
	return r, nil
}

type Requirements struct {
	RoleSources []RoleSource
}

func NewRequirements(rss ...RoleSource) *Requirements {
	return &Requirements{
		RoleSources: rss,
	}
}

func (rm *Requirements) AddRoleSource(rss ...RoleSource) {
	rm.RoleSources = append(rm.RoleSources, rss...)
}

func (rm *Requirements) MarshalYAML() (interface{}, error) {
	var err error
	r := make([]interface{}, len(rm.RoleSources))
	for i := range rm.RoleSources {
		r[i], err = rm.RoleSources[i].MarshalYAML()
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (rm *Requirements) String() string {
	r, err := yaml.Marshal(rm)
	if err != nil {
		panic(err)
	}
	return string(r)
}
