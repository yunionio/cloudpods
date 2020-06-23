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

package samlutils

import (
	"encoding/xml"
	"time"

	"yunion.io/x/pkg/util/timeutils"
)

type SSAMLRequestInput struct {
	AssertionConsumerServiceURL string
	Destination                 string
	RequestID                   string
	EntityID                    string
}

func NewRequest(input SSAMLRequestInput) AuthnRequest {
	nowStr := timeutils.IsoTime(time.Now().UTC())
	req := AuthnRequest{
		XMLName: xml.Name{
			Space: XMLNS_PROTO,
			Local: "AuthnRequest",
		},
		AssertionConsumerServiceURL: input.AssertionConsumerServiceURL,
		Destination:                 input.Destination,
		ForceAuthn:                  "false",
		ID:                          input.RequestID,
		IsPassive:                   "false",
		IssueInstant:                nowStr,
		ProtocolBinding:             BINDING_HTTP_POST,
		Version:                     SAML2_VERSION,
		Issuer: Issuer{
			XMLName: xml.Name{
				Space: XMLNS_ASSERT,
				Local: "Issuer",
			},
			Issuer: input.EntityID,
		},
		NameIDPolicy: NameIDPolicy{
			XMLName: xml.Name{
				Space: XMLNS_PROTO,
				Local: "NameIDPolicy",
			},
			AllowCreate: "true",
			Format:      NAME_ID_FORMAT_TRANSIENT,
			// SPNameQualifier: input.EntityID,
		},
	}
	return req
}
