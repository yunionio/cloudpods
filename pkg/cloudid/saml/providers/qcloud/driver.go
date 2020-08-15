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
	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SQcloudSAMLDriver) GetIdpInitiatedLoginData(cloudAccoutId string, userId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLIdpInitiatedLoginData, error) {
	// TODO
	data := samlutils.SSAMLIdpInitiatedLoginData{}

	data.NameId = "cvmcosreadonly"
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
			value:        "qcs::cam::uin/100008182714:roleName/cvmcosreadonly,qcs::cam::uin/100008182714:saml-provider/saml.yunion.io",
		},
		{
			name:         "https://cloud.tencent.com/SAML/Attributes/RoleSessionName",
			friendlyName: "RoleSessionName",
			value:        "cvmcosreadonly",
		},
	} {
		data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
			Name:         v.name,
			FriendlyName: v.friendlyName,
			Values:       []string{v.value},
		})
	}
	data.RelayState = "https://console.cloud.tencent.com/"

	return data, nil
}

func (d *SQcloudSAMLDriver) GetSpInitiatedLoginData(cloudAccoutId string, userId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
	// not supported
	data := samlutils.SSAMLSpInitiatedLoginData{}

	data.NameId = "cvmcosreadonly"
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
			value:        "qcs::cam::uin/100008182714:roleName/cvmcosreadonly,qcs::cam::uin/100008182714:saml-provider/saml.yunion.io",
		},
		{
			name:         "https://cloud.tencent.com/SAML/Attributes/RoleSessionName",
			friendlyName: "RoleSessionName",
			value:        "cvmcosreadonly",
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
