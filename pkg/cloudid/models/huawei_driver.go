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
	"fmt"
	"net/url"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SHuaweiSAMLDriver) GetIdpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccountId string, sp *idp.SSAMLServiceProvider, redirectUrl string) (samlutils.SSAMLIdpInitiatedLoginData, error) {
	data := samlutils.SSAMLIdpInitiatedLoginData{}
	_account, err := CloudaccountManager.FetchById(cloudAccountId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return data, httperrors.NewResourceNotFoundError2("cloudaccount", cloudAccountId)
		}
		return data, httperrors.NewGeneralError(err)
	}
	account := _account.(*SCloudaccount)
	if account.Provider != api.CLOUD_PROVIDER_HUAWEI && account.Provider != api.CLOUD_PROVIDER_HCSO {
		return data, httperrors.NewClientError("cloudaccount %s is %s not %s", account.Id, account.Provider, api.CLOUD_PROVIDER_HUAWEI)
	}
	if account.SAMLAuth.IsFalse() {
		return data, httperrors.NewNotSupportedError("cloudaccount %s not open saml auth", account.Id)
	}

	saml, valid := account.IsSAMLProviderValid()
	if !valid {
		return data, httperrors.NewResourceNotReadyError("SAMLProvider for account %s not ready", account.Id)
	}

	uri := saml.AuthUrl
	if len(uri) == 0 {
		return data, httperrors.NewResourceNotReadyError("saml provider no auth url")
	}
	url, err := url.Parse(uri)
	if err != nil {
		return data, httperrors.NewInputParameterError("parse saml auth url %s error", uri)
	}

	domainId := url.Query().Get("domain_id")
	idpId := url.Query().Get("idp")
	if len(domainId) == 0 {
		return data, httperrors.NewInputParameterError("saml auth url %s missing domain_id", uri)
	}
	if len(idpId) == 0 {
		return data, httperrors.NewInputParameterError("saml auth url %s missing idp", uri)
	}
	groups, err := account.GetUserCloudgroups(userCred)
	if err != nil {
		return data, httperrors.NewGeneralError(errors.Wrapf(err, "GetUserCloudgroups"))
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
	data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
		Name: "IAM_SAML_Attributes_identityProviders", FriendlyName: "IAM_SAML_Attributes_identityProviders",
		NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
		Values:     []string{fmt.Sprintf("iam::%s:identityProvider:%s", domainId, idpId)},
	})

	if len(redirectUrl) > 0 {
		data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
			Name: "IAM_SAML_Attributes_redirect_url", FriendlyName: "IAM_SAML_Attributes_redirect_url",
			NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
			Values:     []string{redirectUrl},
		})
	}

	return data, nil
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
	if account.Provider != api.CLOUD_PROVIDER_HUAWEI && account.Provider != api.CLOUD_PROVIDER_HCSO {
		return data, httperrors.NewClientError("cloudaccount %s is %s not %s", account.Id, account.Provider, api.CLOUD_PROVIDER_HUAWEI)
	}
	if account.SAMLAuth.IsFalse() {
		return data, httperrors.NewNotSupportedError("cloudaccount %s not open saml auth", account.Id)
	}

	_, valid := account.IsSAMLProviderValid()
	if !valid {
		return data, httperrors.NewResourceNotReadyError("SAMLProvider for account %s not ready", account.Id)
	}

	groups, err := account.GetUserCloudgroups(userCred)
	if err != nil {
		return data, httperrors.NewGeneralError(errors.Wrapf(err, "GetUserCloudgroups"))
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
