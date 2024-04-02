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

package volcengine

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SVolcEngineSAMLDriver) GetIdpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, managerId string, sp *idp.SSAMLServiceProvider, redirectUrl string) (samlutils.SSAMLIdpInitiatedLoginData, error) {
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

	// trn:iam::${AccountID}:role/${RoleName},trn:iam::${AccountID}:saml-provider/${SAMLProviderName}
	roleStr := fmt.Sprintf("trn:iam::%s:role/%s,trn:iam::%s:saml-provider/%s", account.AccountId, role.ExternalId, account.AccountId, samlProvider.ExternalId)

	data.NameId = role.Name
	data.NameIdFormat = samlutils.NAME_ID_FORMAT_TRANSIENT
	data.AudienceRestriction = "https://console.volcengine.com"
	for _, v := range []struct {
		name         string
		friendlyName string
		value        string
	}{
		{
			name:         "https://www.volcengine.com/SAML/Attributes/Identity",
			friendlyName: "RoleEntitlement",
			value:        roleStr,
		},
		{
			name:         "https://www.volcengine.com/SAML/Attributes/SessionName",
			friendlyName: "SessionName",
			value:        role.Name,
		},
		{
			name:         "https://www.volcengine.com/SAML/Attributes/SessionDuration",
			friendlyName: "SessionDuration",
			value:        "7200",
		},
	} {
		data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
			Name:         v.name,
			FriendlyName: v.friendlyName,
			Values:       []string{v.value},
		})
	}
	if len(redirectUrl) == 0 {
		redirectUrl = "https://console.volcengine.com"
	}
	data.RelayState = redirectUrl

	return data, nil
}

func (d *SVolcEngineSAMLDriver) GetSpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, managerId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
	return samlutils.SSAMLSpInitiatedLoginData{}, errors.ErrNotSupported
}
