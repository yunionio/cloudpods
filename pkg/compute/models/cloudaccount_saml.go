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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/cloudid"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/samlutils"
)

func (account *SCloudaccount) AllowGetDetailsSaml(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowGetSpec(userCred, account, "saml")
}

func (account *SCloudaccount) GetDetailsSaml(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.GetCloudaccountSamlOutput, error) {
	output := api.GetCloudaccountSamlOutput{}

	provider, err := account.GetProvider()
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
	// XXXXX
	// TODO, find idpName for this cloudaccount
	// XXXXX
	idpName := "saml.yunion.io"
	output.InitLoginUrl = provider.GetSamlSpInitiatedLoginUrl(idpName)
	if len(output.InitLoginUrl) == 0 {
		input := samlutils.SIdpInitiatedLoginInput{
			EntityID: output.EntityId,
			IdpId:    account.Id,
		}
		output.InitLoginUrl = httputils.JoinPath(options.Options.ApiServer, cloudid.SAML_IDP_PREFIX, fmt.Sprintf("sso?%s", jsonutils.Marshal(input).QueryString()))
	}

	return output, nil
}
