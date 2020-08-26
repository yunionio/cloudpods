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

package aws

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

func (d *SAWSSAMLDriver) GetIdpInitiatedLoginData(cloudAccoutId string, userId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLIdpInitiatedLoginData, error) {
	// TODO
	data := samlutils.SSAMLIdpInitiatedLoginData{}

	data.NameId = "ec2s3readonly"
	data.NameIdFormat = samlutils.NAME_ID_FORMAT_PERSISTENT
	data.AudienceRestriction = "https://signin.aws.amazon.com/saml"
	for _, v := range []struct {
		name         string
		friendlyName string
		value        string
	}{
		{
			name:         "https://aws.amazon.com/SAML/Attributes/Role",
			friendlyName: "RoleEntitlement",
			value:        "arn:aws:iam::285906155448:role/ec2s3readonly,arn:aws:iam::285906155448:saml-provider/saml.yunion.cn",
		},
		{
			name:         "https://aws.amazon.com/SAML/Attributes/RoleSessionName",
			friendlyName: "RoleSessionName",
			value:        "ec2s3readonly",
		},
		{
			name:         "urn:oid:1.3.6.1.4.1.5923.1.1.1.3",
			friendlyName: "eduPersonOrgDN",
			value:        "ec2s3readonly",
		},
	} {
		data.Attributes = append(data.Attributes, samlutils.SSAMLResponseAttribute{
			Name:         v.name,
			FriendlyName: v.friendlyName,
			Values:       []string{v.value},
		})
	}
	data.RelayState = "https://console.aws.amazon.com/"

	return data, nil
}

func (d *SAWSSAMLDriver) GetSpInitiatedLoginData(cloudAccoutId string, userId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error) {
	// not supported
	return samlutils.SSAMLSpInitiatedLoginData{}, errors.ErrNotSupported
}
