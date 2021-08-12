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
	"database/sql"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SHuaweiSAMLDriver) GetIdpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccountId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLIdpInitiatedLoginData, error) {
	// not supported
	data := samlutils.SSAMLIdpInitiatedLoginData{}

	return data, httperrors.ErrNotSupported
}

func (d *SHuaweiSAMLDriver) GetSpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccountId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
	data := samlutils.SSAMLSpInitiatedLoginData{}

	_account, err := CloudaccountManager.FetchById(cloudAccountId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return data, httperrors.NewResourceNotFoundError2("cloudaccount", cloudAccountId)
		}
		return data, httperrors.NewGeneralError(err)
	}
	account := _account.(*SCloudaccount)
	if account.Provider != api.CLOUD_PROVIDER_HUAWEI && account.Provider != api.CLOUD_PROVIDER_HUAWEI_CLOUD_STACK {
		return data, httperrors.NewClientError("cloudaccount %s is %s not %s", account.Id, account.Provider, api.CLOUD_PROVIDER_HUAWEI)
	}
	if account.SAMLAuth.IsFalse() {
		return data, httperrors.NewNotSupportedError("cloudaccount %s not open saml auth", account.Id)
	}

	_, valid := account.IsSAMLProviderValid()
	if !valid {
		return data, httperrors.NewResourceNotReadyError("SAMLProvider for account %s not ready", account.Id)
	}

	groups, err := account.GetUserCloudgroups(userCred.GetUserId())
	if err != nil {
		return data, httperrors.NewGeneralError(errors.Wrapf(err, "GetUserCloudgroups"))
	}
	if len(groups) == 0 {
		return data, httperrors.NewResourceNotFoundError("no available group found")
	}

	data.NameId = userCred.GetUserName()
	data.NameIdFormat = samlutils.NAME_ID_FORMAT_TRANSIENT
	data.AudienceRestriction = sp.GetEntityId()
	for k, v := range map[string][]string{
		"User":   {userCred.GetUserName()},
		"Groups": groups,
	} {
		data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
			Name: k, FriendlyName: k,
			NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values:     v,
		})
	}

	return data, nil
}
