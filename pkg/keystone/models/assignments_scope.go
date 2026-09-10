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

package models

import (
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func checkRoleAssignmentListInput(allowScope rbacscope.TRbacScope, userCred mcclient.TokenCredential, input *api.RoleAssignmentsInput) (string, error) {
	switch allowScope {
	case rbacscope.ScopeSystem:
		return "", nil
	case rbacscope.ScopeDomain:
		domainId := userCred.GetProjectDomainId()
		if err := rejectRoleAssignmentMismatch(input.ProjectDomainId, domainId); err != nil {
			return "", err
		}
		if err := rejectRoleAssignmentMismatch(input.Scope.Domain.Id, domainId); err != nil {
			return "", err
		}
		return domainId, nil
	case rbacscope.ScopeProject:
		projectId := userCred.GetProjectId()
		if err := rejectRoleAssignmentMismatch(input.Scope.Project.Id, projectId); err != nil {
			return "", err
		}
		input.Scope.Project.Id = projectId
		input.IncludePolicies = nil
		return "", nil
	case rbacscope.ScopeUser:
		userId := userCred.GetUserId()
		if err := rejectRoleAssignmentMismatch(input.User.Id, userId); err != nil {
			return "", err
		}
		if len(input.Group.Id) > 0 || len(input.Groups) > 0 {
			return "", httperrors.NewForbiddenError("not allow to list role assignments")
		}
		input.User.Id = userId
		input.IncludePolicies = nil
		return "", nil
	default:
		return "", httperrors.NewForbiddenError("not allow to list role assignments")
	}
}

func rejectRoleAssignmentMismatch(requested, allowed string) error {
	if len(requested) > 0 && requested != allowed {
		return httperrors.NewForbiddenError("not allow to list role assignments")
	}
	return nil
}
