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
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/apis/cloudid"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	cloudid_modules "yunion.io/x/onecloud/pkg/mcclient/modules/cloudid"
)

func (account *SCloudaccount) GetDetailsSaml(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.GetCloudaccountSamlOutput, error) {
	output := api.GetCloudaccountSamlOutput{}

	if account.SAMLAuth.IsFalse() {
		return output, httperrors.NewNotSupportedError("account %s not enable saml auth", account.Name)
	}

	provider, err := account.GetProvider(ctx)
	if err != nil {
		return output, errors.Wrap(err, "GetProviderFactory")
	}

	output.EntityId = provider.GetSamlEntityId()
	if len(output.EntityId) == 0 {
		return output, errors.Wrap(httperrors.ErrNotSupported, "SAML login not supported")
	}
	output.RedirectLoginUrl = httputils.JoinPath(options.Options.ApiServer, cloudid.SAML_IDP_PREFIX, "redirect/login", account.Id)
	output.RedirectLogoutUrl = httputils.JoinPath(options.Options.ApiServer, cloudid.SAML_IDP_PREFIX, "redirect/logout", account.Id)
	output.MetadataUrl = httputils.JoinPath(options.Options.ApiServer, cloudid.SAML_IDP_PREFIX, "metadata", account.Id)
	s := auth.GetAdminSession(ctx, options.Options.Region)
	params := map[string]string{
		"scope":           "system",
		"cloudaccount_id": account.Id,
		"status":          cloudid.SAML_PROVIDER_STATUS_AVAILABLE,
	}
	samlproviders, _ := cloudid_modules.SAMLProviders.List(s, jsonutils.Marshal(params))
	for _, sp := range samlproviders.Data {
		authUrl, _ := sp.GetString("auth_url")
		if len(authUrl) > 0 {
			output.InitLoginUrl = authUrl
			break
		}
	}
	return output, nil
}
