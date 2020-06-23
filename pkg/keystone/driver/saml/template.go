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

package saml

import api "yunion.io/x/onecloud/pkg/apis/identity"

var (
	SAMLTestTemplate = api.SSAMLIdpConfigOptions{
		EntityId:       "https://samltest.id/saml/idp",
		RedirectSSOUrl: "https://samltest.id/idp/profile/SAML2/Redirect/SSO",
		SIdpAttributeOptions: api.SIdpAttributeOptions{
			UserNameAttribute:        "urn:oid:0.9.2342.19200300.100.1.1",
			UserIdAttribute:          "urn:oid:0.9.2342.19200300.100.1.1",
			UserDisplaynameAttribtue: "urn:oid:2.16.840.1.113730.3.1.241",
			UserEmailAttribute:       "urn:oid:0.9.2342.19200300.100.1.3",
			UserMobileAttribute:      "urn:oid:2.5.4.20",
		},
	}

	AzureADTemplate = api.SSAMLIdpConfigOptions{
		SIdpAttributeOptions: api.SIdpAttributeOptions{
			UserNameAttribute:        "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			UserIdAttribute:          "http://schemas.microsoft.com/identity/claims/objectidentifier",
			UserDisplaynameAttribtue: "http://schemas.microsoft.com/identity/claims/displayname",
			UserEmailAttribute:       "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			UserMobileAttribute:      "",
		},
	}
)
