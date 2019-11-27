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

package identity

type SIdentityObject struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type SDomainObject struct {
	SIdentityObject
	Domain SIdentityObject `json:"domain"`
}

type SFetchDomainObject struct {
	SIdentityObject
	Domain   string `json:"domain"`
	DomainId string `json:"domain_id"`
}

type SRoleAssignment struct {
	Scope struct {
		Domain  SIdentityObject `json:"domain"`
		Project SDomainObject   `json:"project"`
	} `json:"scope"`
	User  SDomainObject `json:"user"`
	Group SDomainObject `json:"group"`
	Role  SDomainObject `json:"role"`

	Policies struct {
		Project []string `json:"project"`
		Domain  []string `json:"domain"`
		System  []string `json:"system"`
	} `json:"policies"`
}

// rbacutils.IRbacIdentity interfaces
func (ra *SRoleAssignment) GetProjectDomainId() string {
	return ra.Scope.Project.Domain.Id
}

func (ra *SRoleAssignment) GetProjectName() string {
	return ra.Scope.Project.Name
}

func (ra *SRoleAssignment) GetRoles() []string {
	return []string{ra.Role.Name}
}

func (ra *SRoleAssignment) GetLoginIp() string {
	return ""
}
