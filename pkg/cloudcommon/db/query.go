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

package db

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func FetchQueryDomain(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (string, error) {
	var domainId string
	domainInfo, err := FetchDomainInfo(ctx, query)
	if err != nil {
		return "", httperrors.NewGeneralError(err)
	}
	if domainInfo != nil {
		domainId = domainInfo.GetProjectDomainId()
	}
	scopeStr, _ := query.GetString("scope")
	queryScope := rbacscope.String2ScopeDefault(scopeStr, rbacscope.ScopeDomain)
	if queryScope != rbacscope.ScopeSystem && len(domainId) == 0 {
		domainId = userCred.GetProjectDomainId()
	}
	return domainId, nil
}
