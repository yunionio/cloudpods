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
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

// idpJoinAllowsSystemRole is true only when both project and role come from IdP configuration defaults.
func idpJoinAllowsSystemRole(projectFromAttr, roleFromAttr bool) bool {
	return !projectFromAttr && !roleFromAttr
}

func validateIdpJoinPolicies(assignPolicies rbacutils.TPolicyGroup, allowSystem bool) error {
	if !allowSystem && assignPolicies.HighestScope() == rbacscope.ScopeSystem {
		return errors.Wrap(httperrors.ErrNotSufficientPrivilege, "assigning roles requires higher privilege scope")
	}
	return nil
}

func validateIdpJoinRole(project *SProject, role *SRole, allowSystem bool) error {
	_, assignPolicies, err := RolePolicyManager.GetMatchPolicyGroup2(false, []string{role.Id}, project.Id, "", time.Time{}, false)
	if err != nil {
		return errors.Wrap(err, "GetMatchPolicyGroup2")
	}
	if err := validateIdpJoinPolicies(assignPolicies, allowSystem); err != nil {
		return err
	}
	if options.Options.ThreeAdminRoleSystem {
		return threeMemberSystemValidatePolicies(GetDefaultAdminCred(), project.Id, assignPolicies)
	}
	return nil
}
