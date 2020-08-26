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

	"yunion.io/x/pkg/errors"
)

func ParseMetadata(data []byte) (EntityDescriptor, error) {
	ed := EntityDescriptor{}
	err := xml.Unmarshal(data, &ed)
	if err != nil {
		return ed, errors.Wrap(err, "xml.Unmarshal")
	}
	return ed, nil
}

type SSAMLIdpMetadataInput struct {
	EntityId          string
	CertString        string
	RedirectLoginUrl  string
	RedirectLogoutUrl string
}

func NewIdpMetadata(input SSAMLIdpMetadataInput) EntityDescriptor {
	desc := EntityDescriptor{
		XMLName: xml.Name{
			Space: XMLNS_MD,
			Local: "EntityDescriptor",
		},
		EntityId: input.EntityId,
		IDPSSODescriptor: &SSODescriptor{
			XMLName: xml.Name{
				Space: XMLNS_MD,
				Local: "IDPSSODescriptor",
			},
			ProtocolSupportEnumeration: PROTOCOL_SAML2,
			KeyDescriptors: []KeyDescriptor{
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "KeyDescriptor",
					},
					Use: KEY_USE_SIGNING,
					KeyInfo: KeyInfo{
						XMLName: xml.Name{
							Space: XMLNS_DS,
							Local: "KeyInfo",
						},
						X509Data: &X509Data{
							XMLName: xml.Name{
								Space: XMLNS_DS,
								Local: "X509Data",
							},
							X509Certificate: X509Certificate{
								XMLName: xml.Name{
									Space: XMLNS_DS,
									Local: "X509Certificate",
								},
								Cert: input.CertString,
							},
						},
					},
				},
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "KeyDescriptor",
					},
					Use: KEY_USE_ENCRYPTION,
					KeyInfo: KeyInfo{
						XMLName: xml.Name{
							Space: XMLNS_DS,
							Local: "KeyInfo",
						},
						X509Data: &X509Data{
							XMLName: xml.Name{
								Space: XMLNS_DS,
								Local: "X509Data",
							},
							X509Certificate: X509Certificate{
								XMLName: xml.Name{
									Space: XMLNS_DS,
									Local: "X509Certificate",
								},
								Cert: input.CertString,
							},
						},
					},
				},
			},
			SingleLogoutServices: []SSAMLService{
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "SingleLogoutService",
					},
					Binding:  BINDING_HTTP_REDIRECT,
					Location: input.RedirectLogoutUrl,
				},
			},
			NameIDFormat: []SSAMLNameIDFormat{
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "NameIDFormat",
					},
					Format: NAME_ID_FORMAT_TRANSIENT,
				},
			},
			SingleSignOnServices: []SSAMLService{
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "SingleSignOnService",
					},
					Binding:  BINDING_HTTP_REDIRECT,
					Location: input.RedirectLoginUrl,
				},
			},
		},
	}
	return desc
}

type SSAMLSpMetadataInput struct {
	EntityId             string
	CertString           string
	AssertionConsumerUrl string
	ServiceName          string
	RequestedAttributes  []RequestedAttribute
}

func NewSpMetadata(input SSAMLSpMetadataInput) EntityDescriptor {
	strTrue := "true"
	strIndex := "1"

	reqAttrs := make([]RequestedAttribute, len(input.RequestedAttributes))
	for i := range input.RequestedAttributes {
		reqAttrs[i] = RequestedAttribute{
			XMLName: xml.Name{
				Space: XMLNS_MD,
				Local: "RequestedAttribute",
			},
			IsRequired:   input.RequestedAttributes[i].IsRequired,
			Name:         input.RequestedAttributes[i].Name,
			FriendlyName: input.RequestedAttributes[i].FriendlyName,
		}
	}

	desc := EntityDescriptor{
		XMLName: xml.Name{
			Space: XMLNS_MD,
			Local: "EntityDescriptor",
		},
		EntityId: input.EntityId,
		SPSSODescriptor: &SSODescriptor{
			XMLName: xml.Name{
				Space: XMLNS_MD,
				Local: "SPSSODescriptor",
			},
			ProtocolSupportEnumeration: PROTOCOL_SAML2,
			WantAssertionsSigned:       &strTrue,
			KeyDescriptors: []KeyDescriptor{
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "KeyDescriptor",
					},
					Use: KEY_USE_SIGNING,
					KeyInfo: KeyInfo{
						XMLName: xml.Name{
							Space: XMLNS_DS,
							Local: "KeyInfo",
						},
						X509Data: &X509Data{
							XMLName: xml.Name{
								Space: XMLNS_DS,
								Local: "X509Data",
							},
							X509Certificate: X509Certificate{
								XMLName: xml.Name{
									Space: XMLNS_DS,
									Local: "X509Certificate",
								},
								Cert: input.CertString,
							},
						},
					},
				},
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "KeyDescriptor",
					},
					Use: KEY_USE_ENCRYPTION,
					KeyInfo: KeyInfo{
						XMLName: xml.Name{
							Space: XMLNS_DS,
							Local: "KeyInfo",
						},
						X509Data: &X509Data{
							XMLName: xml.Name{
								Space: XMLNS_DS,
								Local: "X509Data",
							},
							X509Certificate: X509Certificate{
								XMLName: xml.Name{
									Space: XMLNS_DS,
									Local: "X509Certificate",
								},
								Cert: input.CertString,
							},
						},
					},
				},
			},
			NameIDFormat: []SSAMLNameIDFormat{
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "NameIDFormat",
					},
					Format: NAME_ID_FORMAT_TRANSIENT,
				},
			},
			AssertionConsumerServices: []SSAMLService{
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "AssertionConsumerService",
					},
					Binding:   BINDING_HTTP_POST,
					Location:  input.AssertionConsumerUrl,
					Index:     &strIndex,
					IsDefault: &strTrue,
				},
			},
			AttributeConsumingServices: []AttributeConsumingService{
				{
					XMLName: xml.Name{
						Space: XMLNS_MD,
						Local: "AttributeConsumingService",
					},
					Index: strIndex,
					ServiceName: SXMLText{
						XMLName: xml.Name{
							Space: XMLNS_MD,
							Local: "ServiceName",
						},
						Lang: "en",
						Text: input.ServiceName,
					},
					RequestedAttributes: reqAttrs,
				},
			},
		},
	}
	return desc
}
