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

package azure

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SAzureSAMLDriver) GetIdpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccountId string, sp *idp.SSAMLServiceProvider, redirectUrl string) (samlutils.SSAMLIdpInitiatedLoginData, error) {
	data := samlutils.SSAMLIdpInitiatedLoginData{}

	_account, err := models.CloudaccountManager.FetchById(cloudAccountId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return data, httperrors.NewResourceNotFoundError2("cloudaccount", cloudAccountId)
		}
		return data, httperrors.NewGeneralError(err)
	}
	account := _account.(*models.SCloudaccount)
	if account.Provider != api.CLOUD_PROVIDER_AZURE {
		return data, httperrors.NewClientError("cloudaccount %s is %s not %s", account.Id, account.Provider, api.CLOUD_PROVIDER_AWS)
	}
	if account.SAMLAuth.IsFalse() {
		return data, httperrors.NewNotSupportedError("cloudaccount %s not open saml auth", account.Id)
	}

	samlProvider, valid := account.IsSAMLProviderValid()
	if !valid {
		return data, httperrors.NewResourceNotReadyError("SAMLProvider for account %s not ready", account.Id)
	}

	inviteUrl, err := account.InviteAzureUser(ctx, userCred, samlProvider.ExternalId)
	if err != nil {
		return data, httperrors.NewGeneralError(errors.Wrapf(err, "InviteAzureUser"))
	}

	data.Form = fmt.Sprintf(`<!DOCTYPE html>
							<html lang="en">
							<head>
							    <meta charset="UTF-8">
							    <meta http-equiv="refresh" content="0; url=%s"> 
							    <title>waiting...</title>
							</head>
							<body>
							    waiting...
							</body>
							</html>`, inviteUrl)
	return data, nil
}

func (d *SAzureSAMLDriver) GetSpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccountId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
	data := samlutils.SSAMLSpInitiatedLoginData{}

	_account, err := models.CloudaccountManager.FetchById(cloudAccountId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return data, httperrors.NewResourceNotFoundError2("cloudaccount", cloudAccountId)
		}
		return data, httperrors.NewGeneralError(err)
	}
	account := _account.(*models.SCloudaccount)
	if account.Provider != api.CLOUD_PROVIDER_AZURE {
		return data, httperrors.NewClientError("cloudaccount %s is %s not %s", account.Id, account.Provider, api.CLOUD_PROVIDER_AWS)
	}
	if account.SAMLAuth.IsFalse() {
		return data, httperrors.NewNotSupportedError("cloudaccount %s not open saml auth", account.Id)
	}

	samlUsers, err := account.GetSamlusers()
	if err != nil {
		return data, httperrors.NewGeneralError(errors.Wrapf(err, "GetSamlusers"))
	}

	for i := range samlUsers {
		if samlUsers[i].OwnerId == userCred.GetUserId() && strings.ToLower(samlUsers[i].Email) == strings.ToLower(sp.Username) {

			err := samlUsers[i].SyncAzureGroup()
			if err != nil {
				return data, errors.Wrapf(err, "SyncAzureGroup")
			}

			data.NameId = sp.Username

			data.NameIdFormat = samlutils.NAME_ID_FORMAT_PERSISTENT
			data.AudienceRestriction = sp.GetEntityId()
			for _, v := range []struct {
				name         string
				friendlyName string
				value        string
			}{
				{
					name:  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
					value: data.NameId,
				},
			} {
				data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
					Name:         v.name,
					FriendlyName: v.friendlyName,
					Values:       []string{v.value},
				})
			}
			return data, nil

		}
	}

	return data, httperrors.NewResourceNotFoundError("not found any saml user for %s -> %s", userCred.GetUserName(), sp.Username)
}
