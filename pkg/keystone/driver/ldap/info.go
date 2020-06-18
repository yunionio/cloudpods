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

package ldap

type SDomainInfo struct {
	DN   string
	Id   string
	Name string
}

type SUserInfo struct {
	SDomainInfo
	Enabled bool
	Extra   map[string]string
}

type SGroupInfo struct {
	SDomainInfo
	Members []string
}

func (info SDomainInfo) isValid() bool {
	return len(info.DN) > 0 && len(info.Id) > 0 && len(info.Name) > 0
}

func (info SUserInfo) isValid() bool {
	if !info.SDomainInfo.isValid() {
		return false
	}
	// regarding disabled LDAP user as invalid
	if !info.Enabled {
		return false
	}
	return true
}
