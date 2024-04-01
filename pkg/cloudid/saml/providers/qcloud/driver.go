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

package qcloud

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SQcloudSAMLDriver) GetIdpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, managerId string, sp *idp.SSAMLServiceProvider, redirectUrl string) (samlutils.SSAMLIdpInitiatedLoginData, error) {
	data := samlutils.SSAMLIdpInitiatedLoginData{}

	provider, err := models.CloudproviderManager.FetchProvier(managerId)
	if err != nil {
		return data, err
	}

	account, err := provider.GetCloudaccount()
	if err != nil {
		return data, errors.Wrapf(err, "GetCloudaccount")
	}

	role, err := provider.GetRole(ctx, userCred.GetUserId())
	if err != nil {
		return data, err
	}

	samlProvider, err := provider.GetSamlProvider()
	if err != nil {
		return data, err
	}

	roleStr := fmt.Sprintf("qcs::cam::uin/%s:roleName/%s,qcs::cam::uin/%s:saml-provider/%s", account.AccountId, role.ExternalId, account.AccountId, samlProvider.ExternalId)

	data.NameId = role.Name
	data.NameIdFormat = samlutils.NAME_ID_FORMAT_TRANSIENT
	data.AudienceRestriction = "https://cloud.tencent.com"
	for _, v := range []struct {
		name         string
		friendlyName string
		value        string
	}{
		{
			name:         "https://cloud.tencent.com/SAML/Attributes/Role",
			friendlyName: "RoleEntitlement",
			value:        roleStr,
		},
		{
			name:         "https://cloud.tencent.com/SAML/Attributes/RoleSessionName",
			friendlyName: "RoleSessionName",
			value:        data.NameId,
		},
	} {
		data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
			Name:         v.name,
			FriendlyName: v.friendlyName,
			Values:       []string{v.value},
		})
	}
	if len(redirectUrl) == 0 {
		redirectUrl = "https://console.cloud.tencent.com/"
	}
	data.RelayState = redirectUrl

	return data, nil
}

func (d *SQcloudSAMLDriver) GetSpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, managerId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
	data := samlutils.SSAMLSpInitiatedLoginData{}

	provider, err := models.CloudproviderManager.FetchProvier(managerId)
	if err != nil {
		return data, err
	}

	account, err := provider.GetCloudaccount()
	if err != nil {
		return data, errors.Wrapf(err, "GetCloudaccount")
	}

	role, err := provider.GetRole(ctx, userCred.GetUserId())
	if err != nil {
		return data, err
	}

	samlProvider, err := provider.GetSamlProvider()
	if err != nil {
		return data, err
	}

	roleStr := fmt.Sprintf("qcs::cam::uin/%s:roleName/%s,qcs::cam::uin/%s:saml-provider/%s", account.AccountId, role.ExternalId, account.AccountId, samlProvider.ExternalId)

	data.NameId = role.Name
	data.NameIdFormat = samlutils.NAME_ID_FORMAT_TRANSIENT
	data.AudienceRestriction = "https://cloud.tencent.com"
	for _, v := range []struct {
		name         string
		friendlyName string
		value        string
	}{
		{
			name:         "https://cloud.tencent.com/SAML/Attributes/Role",
			friendlyName: "RoleEntitlement",
			value:        roleStr,
		},
		{
			name:         "https://cloud.tencent.com/SAML/Attributes/RoleSessionName",
			friendlyName: "RoleSessionName",
			value:        role.Name,
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
