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

package aliyun

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SAliyunSAMLDriver) GetIdpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, managerId string, sp *idp.SSAMLServiceProvider, redirectUrl string) (samlutils.SSAMLIdpInitiatedLoginData, error) {
	data := samlutils.SSAMLIdpInitiatedLoginData{}

	provider, err := models.CloudproviderManager.FetchProvier(managerId)
	if err != nil {
		return data, err
	}

	role, err := provider.GetRole(ctx, userCred.GetUserId())
	if err != nil {
		return data, err
	}

	samlProvider, err := provider.GetSamlProvider()
	if err != nil {
		return data, err
	}

	data.NameId = userCred.GetUserName()
	data.NameIdFormat = samlutils.NAME_ID_FORMAT_PERSISTENT
	data.AudienceRestriction = sp.GetEntityId()
	for k, v := range map[string]string{
		"https://www.aliyun.com/SAML-Role/Attributes/Role":            fmt.Sprintf("%s,%s", role.ExternalId, samlProvider.ExternalId),
		"https://www.aliyun.com/SAML-Role/Attributes/RoleSessionName": userCred.GetUserId(),
		"https://www.aliyun.com/SAML-Role/Attributes/SessionDuration": "1800",
	} {
		data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
			Name:   k,
			Values: []string{v},
		})
	}
	if len(redirectUrl) == 0 {
		redirectUrl = "https://homenew.console.aliyun.com/"
	}
	data.RelayState = redirectUrl
	return data, nil
}

func (d *SAliyunSAMLDriver) GetSpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccountId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
	// not supported
	return samlutils.SSAMLSpInitiatedLoginData{}, errors.ErrNotSupported
}
