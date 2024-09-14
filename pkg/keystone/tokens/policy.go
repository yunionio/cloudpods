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
	"net/http"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func fetchTokenPolicies(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := policy.FetchUserCredential(ctx)
	names, group, err := models.RolePolicyManager.GetMatchPolicyGroupByCred(ctx, token, time.Now(), false)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	output := mcclient.SFetchMatchPoliciesOutput{}
	output.Names = names
	output.Policies = group
	appsrv.SendJSON(w, output.Encode())
}

func postTokenPolicies(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)
	if body == nil {
		httperrors.InvalidInputError(ctx, w, "empty request body")
		return
	}
	input := mcclient.SCheckPoliciesInput{}
	err := body.Unmarshal(&input)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	output, err := doCheckPolicies(ctx, input)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, output.Encode())
}

func doCheckPolicies(ctx context.Context, input mcclient.SCheckPoliciesInput) (*mcclient.SFetchMatchPoliciesOutput, error) {
	adminToken := policy.FetchUserCredential(ctx)
	if adminToken == nil {
		return nil, httperrors.NewForbiddenError("missing auth token")
	}
	if policy.PolicyManager.Allow(rbacscope.ScopeSystem, adminToken, api.SERVICE_TYPE, "tokens", "perform", "check_policies").Result.IsDeny() {
		return nil, httperrors.NewForbiddenError("%s not allow to check policies", adminToken.GetUserName())
	}
	log.Debugf("doCheckPolicies userId: %s projectId: %s", input.UserId, input.ProjectId)
	names, group, err := models.RolePolicyManager.GetMatchPolicyGroupByInput(ctx, input.UserId, input.ProjectId, time.Now(), false)
	if err != nil {
		return nil, errors.Wrap(err, "GetMatchPolicyGroupByInput")
	}
	output := mcclient.SFetchMatchPoliciesOutput{}
	output.Names = names
	output.Policies = group
	return &output, nil
}
