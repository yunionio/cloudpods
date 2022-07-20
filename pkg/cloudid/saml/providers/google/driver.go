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

package google

import (
	"context"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SGoogleSAMLDriver) GetIdpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccountId string, sp *idp.SSAMLServiceProvider, redirectUrl string) (samlutils.SSAMLIdpInitiatedLoginData, error) {
	// not supported
	data := samlutils.SSAMLIdpInitiatedLoginData{}

	return data, httperrors.ErrNotSupported
}

func (d *SGoogleSAMLDriver) GetSpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccountId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
	// TODO
	data := samlutils.SSAMLSpInitiatedLoginData{}

	data.NameId = "qiujian@yunion-hk.com"
	data.NameIdFormat = samlutils.NAME_ID_FORMAT_EMAIL
	data.AudienceRestriction = sp.GetEntityId()
	for k, v := range map[string]string{
		"user.email": "qiujian@yunion-hk.com",
	} {
		data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
			Name: k, FriendlyName: k,
			NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
			Values:     []string{v},
		})
	}

	return data, nil
}
