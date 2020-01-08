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

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
)

type IdentityBaseResourceCreateInput struct {
	apis.StandaloneResourceCreateInput
}

type EnabledIdentityBaseResourceCreateInput struct {
	IdentityBaseResourceCreateInput

	Enabled *bool `json:"enabled"`
}

type IdentityBaseResourceListInput struct {
	apis.StandaloneResourceListInput
	apis.DomainizedResourceListInput
}

type EnabledIdentityBaseResourceListInput struct {
	IdentityBaseResourceListInput

	// filter by enabled status
	Enabled *bool `json:"enabled"`
}

type RoleListInput struct {
	IdentityBaseResourceListInput

	// filter by project_id
	ProjectId string `json:"project_id"`
	// filter by user_id
	UserId string `json:"user_id"`
	// filter by group_id
	GroupId string `json:"group_id"`
}

type GroupListInput struct {
	IdentityBaseResourceListInput

	// filter by user_id
	UserId string `json:"user_id"`
	// filter by project_id
	ProjectId string `json:"project_id"`
	// deprecated: true
	// filter by tenant_id
	TenantId string `json:"tenant_id"`
}

func (input GroupListInput) ProjectIdStr() string {
	if len(input.ProjectId) > 0 {
		return input.ProjectId
	}
	if len(input.TenantId) > 0 {
		return input.TenantId
	}
	return ""
}

type ProjectListInput struct {
	IdentityBaseResourceListInput

	// filter by user_id
	UserId string `json:"user_id"`
	// filter by group_id
	GroupId string `json:"group_id"`
}

type DomainListInput struct {
	apis.StandaloneResourceListInput
}

type UserListInput struct {
	EnabledIdentityBaseResourceListInput

	// filter by group_id
	GroupId string `json:"group_id"`
	// deprecated: true
	// filter by group
	Group string `json:"group"`

	// filter by project
	Project string `json:"project"`
	// filter by project_id
	ProjectId string `json:"project_id"`
	// filter by tenant
	Tenant string `json:"tenant"`
	// filter by tenant_id
	TenantId string `json:"tenant_id"`

	// filter by role
	Role string `json:"role"`
	// deprecated: true
	// filter by role_id
	RoleId string `json:"role_id"`
}

func (input UserListInput) GroupStr() string {
	if len(input.GroupId) > 0 {
		return input.GroupId
	}
	if len(input.Group) > 0 {
		return input.Group
	}
	return ""
}

func (input UserListInput) ProjectStr() string {
	if len(input.Project) > 0 {
		return input.Project
	}
	if len(input.ProjectId) > 0 {
		return input.ProjectId
	}
	if len(input.Tenant) > 0 {
		return input.Tenant
	}
	if len(input.TenantId) > 0 {
		return input.TenantId
	}
	return ""
}

func (input UserListInput) RoleStr() string {
	if len(input.Role) > 0 {
		return input.Role
	}
	if len(input.RoleId) > 0 {
		return input.RoleId
	}
	return ""
}

type EndpointListInput struct {
	apis.StandaloneResourceListInput

	// filter by service, either id or name
	Service string `json:"service"`
	// deprecated: true
	// filter by service_id
	ServiceId string `json:"service_id"`
}

func (input EndpointListInput) ServiceStr() string {
	if len(input.Service) > 0 {
		return input.Service
	}
	if len(input.ServiceId) > 0 {
		return input.ServiceId
	}
	return ""
}

type SJoinProjectsInput struct {
	Projects []string
	Roles    []string
}

func (input SJoinProjectsInput) Validate() error {
	if len(input.Projects) == 0 {
		return errors.Error("empty projects")
	}
	if len(input.Roles) == 0 {
		return errors.Error("empty roles")
	}
	return nil
}

type SProjectRole struct {
	Project string
	Role    string
}
type SLeaveProjectsInput struct {
	ProjectRoles []SProjectRole
}

func (input SLeaveProjectsInput) Validate() error {
	if len(input.ProjectRoles) == 0 {
		return errors.Error("empty project_roles")
	}
	for i := range input.ProjectRoles {
		if len(input.ProjectRoles[i].Project) == 0 {
			return errors.Error("no project in project_roles")
		}
		if len(input.ProjectRoles[i].Role) == 0 {
			return errors.Error("no role in project_roles")
		}
	}
	return nil
}

type SProjectAddUserGroupInput struct {
	Users  []string
	Groups []string
	Roles  []string
}

func (input SProjectAddUserGroupInput) Validate() error {
	if len(input.Users) == 0 && len(input.Groups) == 0 {
		return errors.Error("empty user and group")
	}
	if len(input.Roles) == 0 {
		return errors.Error("invalid roles")
	}
	return nil
}

type SUserRole struct {
	User string
	Role string
}
type SGroupRole struct {
	Group string
	Role  string
}

type SProjectRemoveUserGroupInput struct {
	UserRoles  []SUserRole
	GroupRoles []SGroupRole
}

func (input SProjectRemoveUserGroupInput) Validate() error {
	if len(input.UserRoles) == 0 && len(input.GroupRoles) == 0 {
		return errors.Error("empty input")
	}
	for i := range input.UserRoles {
		if len(input.UserRoles[i].User) == 0 {
			return errors.Error("empty user")
		}
		if len(input.UserRoles[i].Role) == 0 {
			return errors.Error("empty role")
		}
	}
	for i := range input.GroupRoles {
		if len(input.GroupRoles[i].Group) == 0 {
			return errors.Error("empty group")
		}
		if len(input.GroupRoles[i].Role) == 0 {
			return errors.Error("empty role")
		}
	}
	return nil
}
