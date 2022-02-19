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

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func fetchTokenPolicies(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := policy.FetchUserCredential(ctx)
	names, group, err := models.RolePolicyManager.GetMatchPolicyGroup(token, time.Now(), false)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	output := mcclient.SFetchMatchPoliciesOutput{}
	output.Names = names
	output.Policies = group
	appsrv.SendJSON(w, output.Encode())
}
