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

package tokens

import (
	"context"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// authUserByAssume allows an admin token to login as any user without password
func authUserByAssume(ctx context.Context, input mcclient.SAuthenticationInputV3) (*api.SUserExtended, error) {
	// Validate admin token
	if len(input.Auth.Identity.Token.Id) == 0 {
		return nil, httperrors.NewInputParameterError("admin token is required for assume authentication")
	}

	// Decode and validate the admin token
	adminToken, err := TokenStrDecode(ctx, input.Auth.Identity.Token.Id)
	if err != nil {
		return nil, errors.Wrap(err, "decode admin token")
	}

	// Check if admin token is expired
	if adminToken.IsExpired() {
		return nil, ErrExpiredToken
	}

	scopedProject, err := models.ProjectManager.FetchProject(
		input.Auth.Scope.Project.Id,
		input.Auth.Scope.Project.Name,
		input.Auth.Scope.Project.Domain.Id,
		input.Auth.Scope.Project.Domain.Name,
	)
	if err != nil {
		return nil, errors.Wrap(err, "fetch scoped project")
	}

	adminTokenCred, err := adminToken.GetSimpleUserCred(input.Auth.Identity.Token.Id)
	if err != nil {
		return nil, errors.Wrap(err, "get admin token credential")
	}

	// Validate target user information
	assumeUser := input.Auth.Identity.Assume.User
	if len(assumeUser.Id) == 0 && len(assumeUser.Name) == 0 {
		return nil, httperrors.NewInputParameterError("target user id or name is required")
	}

	// Fetch target user
	targetUser, err := models.UserManager.FetchUserExtended(
		assumeUser.Id,
		assumeUser.Name,
		assumeUser.Domain.Id,
		assumeUser.Domain.Name,
	)
	if err != nil {
		return nil, errors.Wrap(err, "fetch target user")
	}

	if adminTokenCred.GetUserId() != targetUser.Id {
		if err := checkAssumeAllowed(adminTokenCred, targetUser.Id, scopedProject); err != nil {
			return nil, err
		}
	}

	// Set audit IDs to include both admin token and assume operation
	targetUser.AuditIds = []string{adminTokenCred.GetUserId()}

	return targetUser, nil
}

func checkAssumeAllowed(adminTokenCred mcclient.TokenCredential, targetUserId string, scopedProject *models.SProject) error {
	if policy.PolicyManager.Allow(rbacscope.ScopeSystem, adminTokenCred, api.SERVICE_TYPE, "tokens", "perform", "assume").Result.IsDeny() {
		return httperrors.NewForbiddenError("%s not allow to assume user in project %s", adminTokenCred.GetUserName(), scopedProject.Name)
	}
	roles, err := models.AssignmentManager.FetchUserProjectRoles(targetUserId, scopedProject.Id)
	if err != nil {
		return errors.Wrap(err, "fetch target user roles")
	}
	roleIds := make([]string, len(roles))
	for i := range roles {
		roleIds[i] = roles[i].Id
	}
	if err := models.ValidateJoinProjectRoles(adminTokenCred, scopedProject.Id, roleIds); err != nil {
		return errors.Wrap(err, "validate assume target roles")
	}
	return nil
}
