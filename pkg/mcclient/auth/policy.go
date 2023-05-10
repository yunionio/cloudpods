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

package auth

import (
	"context"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func (a *authManager) fetchMatchPolicies(ctx context.Context, token mcclient.TokenCredential) (*mcclient.SFetchMatchPoliciesOutput, error) {
	return a.client.FetchMatchPolicies(ctx, token)
}

func (a *authManager) checkMatchPolicies(ctx context.Context, input mcclient.SCheckPoliciesInput) (*mcclient.SFetchMatchPoliciesOutput, error) {
	return a.client.CheckMatchPolicies(ctx, a.adminCredential, input)
}

func FetchMatchPolicies(ctx context.Context, token mcclient.TokenCredential) (*mcclient.SFetchMatchPoliciesOutput, error) {
	if len(token.GetTokenString()) > 0 && !IsGuestToken(token) {
		return manager.fetchMatchPolicies(ctx, token)
	} else {
		input := mcclient.SCheckPoliciesInput{
			UserId:    token.GetUserId(),
			ProjectId: token.GetProjectId(),
			LoginIp:   token.GetLoginIp(),
		}
		return manager.checkMatchPolicies(ctx, input)
	}
}
