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
)

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
	if len(input.Users) == 0 || len(input.Groups) == 0 {
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
