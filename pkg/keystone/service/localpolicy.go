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

package service

import (
	"context"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func localPolicyFetcher(ctx context.Context, token mcclient.TokenCredential) (*mcclient.SFetchMatchPoliciesOutput, error) {
	names, groups, err := models.RolePolicyManager.GetMatchPolicyGroupByCred(ctx, token, time.Now(), false)
	if err != nil {
		return nil, errors.Wrap(err, "GetMatchPolicyGroup")
	}

	output := mcclient.SFetchMatchPoliciesOutput{}
	output.Names = names
	output.Policies = groups

	return &output, nil
}
