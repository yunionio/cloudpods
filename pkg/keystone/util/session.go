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

package util

import (
	"context"
	"time"

	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/tokens"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	authToken   *tokens.SAuthToken
	simpleToken *mcclient.SSimpleToken
)

func getDefaultAdminCredWithToken() (mcclient.TokenCredential, error) {
	if simpleToken == nil {
		simpleToken = models.GetDefaultAdminSSimpleToken()
	}
	var err error
	if now := time.Now(); authToken == nil || authToken.ExpiresAt.Sub(now) < time.Duration(3600) {
		authTokenTmp := &tokens.SAuthToken{
			UserId:    simpleToken.GetUserId(),
			Method:    api.AUTH_METHOD_TOKEN,
			ProjectId: simpleToken.GetProjectId(),
			ExpiresAt: now.Add(24 * time.Hour),
			AuditIds:  []string{utils.GenRequestId(16)},
		}
		simpleToken.Token, err = authTokenTmp.EncodeFernetToken()
		if err != nil {
			return nil, err
		}
		authToken = authTokenTmp
	}
	return simpleToken, nil
}

func GetDefaulAdminSession(ctx context.Context, region, apiVersion string) (*mcclient.ClientSession, error) {
	cred, err := getDefaultAdminCredWithToken()
	if err != nil {
		return nil, err
	}
	return models.GetDefaultClientSession(ctx, cred, region, apiVersion), nil
}
