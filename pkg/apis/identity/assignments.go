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
	Id   string
	Name string
}

type SDomainObject struct {
	SIdentityObject
	Domain SIdentityObject
}

type SFetchDomainObject struct {
	SIdentityObject
	Domain   string
	DomainId string
}

type SRoleAssignment struct {
	Scope struct {
		Domain  SIdentityObject
		Project SDomainObject
	}
	User  SDomainObject
	Group SDomainObject
	Role  SDomainObject

	Policies struct {
		Project []string
		Domain  []string
		System  []string
	}
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
