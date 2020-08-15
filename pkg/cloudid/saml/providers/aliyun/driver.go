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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SAliyunSAMLDriver) GetIdpInitiatedLoginData(cloudAccoutId string, userId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLIdpInitiatedLoginData, error) {
	// TODO
	data := samlutils.SSAMLIdpInitiatedLoginData{}
	data.NameId = "ecsossreadonly"
	data.NameIdFormat = samlutils.NAME_ID_FORMAT_PERSISTENT
	data.AudienceRestriction = sp.GetEntityId()
	for k, v := range map[string]string{
		"https://www.aliyun.com/SAML-Role/Attributes/Role":            "acs:ram::1123247935774897:role/administrator,acs:ram::1123247935774897:saml-provider/saml.yunion.io",
		"https://www.aliyun.com/SAML-Role/Attributes/RoleSessionName": "ecsossreadonly",
		"https://www.aliyun.com/SAML-Role/Attributes/SessionDuration": "1800",
	} {
		data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
			Name:   k,
			Values: []string{v},
		})
	}
	data.RelayState = "https://homenew.console.aliyun.com/"
	return data, nil
}

func (d *SAliyunSAMLDriver) GetSpInitiatedLoginData(cloudAccoutId string, userId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
	// not supported
	return samlutils.SSAMLSpInitiatedLoginData{}, errors.ErrNotSupported
}
