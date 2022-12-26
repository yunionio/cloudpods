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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
)

type SSAMLResponseAttribute struct {
	Name         string
	NameFormat   string
	FriendlyName string
	Values       []string
}

type SSAMLSpInitiatedLoginData struct {
	NameId       string
	NameIdFormat string

	AudienceRestriction string

	Attributes []SSAMLResponseAttribute

	Form string
}

type SSAMLIdpInitiatedLoginData struct {
	SSAMLSpInitiatedLoginData

	RelayState string
}

type SSAMLResponseInput struct {
	IssuerEntityId string

	RequestEntityId string
	RequestID       string

	AssertionConsumerServiceURL string

	IssuerCertString string

	SSAMLSpInitiatedLoginData
}

func NewResponse(input SSAMLResponseInput) Response {
	// since := timeutils.IsoTime(time.Now().UTC().Add(-time.Minute * 60 * 24))
	until := timeutils.IsoTime(time.Now().UTC().Add(time.Minute * 5))

	respId := GenerateSAMLId()
	assertId := GenerateSAMLId()
	now := timeutils.IsoTime(time.Now().UTC())
	issuerFormat := NAME_ID_FORMAT_ENTITY

	issuer := Issuer{
		XMLName: xml.Name{
			Space: XMLNS_ASSERT,
			Local: "Issuer",
		},
		Format: &issuerFormat,
		Issuer: input.IssuerEntityId,
	}

	var responseTo *string
	if len(input.RequestID) > 0 {
		responseTo = &input.RequestID
	}

	resp := Response{
		XMLName: xml.Name{
			Space: XMLNS_PROTO,
			Local: "Response",
		},
		ID:           respId,
		InResponseTo: responseTo,
		Version:      SAML2_VERSION,
		IssueInstant: now,
		Destination:  input.AssertionConsumerServiceURL,
		Issuer:       issuer,
		Status: Status{
			XMLName: xml.Name{
				Space: XMLNS_PROTO,
				Local: "Status",
			},
			StatusCode: StatusCode{
				XMLName: xml.Name{
					Space: XMLNS_PROTO,
					Local: "StatusCode",
				},
				Value: STATUS_SUCCESS,
			},
			StatusMessage: &StatusMessage{
				XMLName: xml.Name{
					Space: XMLNS_PROTO,
					Local: "StatusMessage",
				},
				Message: STATUS_SUCCESS,
			},
		},
		Assertion: &Assertion{
			XMLName: xml.Name{
				Space: XMLNS_ASSERT,
				Local: "Assertion",
			},
			ID:           assertId,
			Version:      SAML2_VERSION,
			IssueInstant: now,
			Issuer:       issuer,
			Signature: &Signature{
				XMLName: xml.Name{
					Space: XMLNS_DS,
					Local: "Signature",
				},
				SignedInfo: SignedInfo{
					XMLName: xml.Name{
						Space: XMLNS_DS,
						Local: "SignedInfo",
					},
					CanonicalizationMethod: EncryptionMethod{
						XMLName: xml.Name{
							Space: XMLNS_DS,
							Local: "CanonicalizationMethod",
						},
						Algorithm: "http://www.w3.org/2001/10/xml-exc-c14n#",
					},
					SignatureMethod: EncryptionMethod{
						XMLName: xml.Name{
							Space: XMLNS_DS,
							Local: "SignatureMethod",
						},
						Algorithm: "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
					},
					Reference: Reference{
						XMLName: xml.Name{
							Space: XMLNS_DS,
							Local: "Reference",
						},
						URI: "#" + assertId,
						Transforms: Transforms{
							XMLName: xml.Name{
								Space: XMLNS_DS,
								Local: "Transforms",
							},
							Transforms: []EncryptionMethod{
								{
									XMLName: xml.Name{
										Space: XMLNS_DS,
										Local: "Transform",
									},
									Algorithm: "http://www.w3.org/2000/09/xmldsig#enveloped-signature",
								},
								{
									XMLName: xml.Name{
										Space: XMLNS_DS,
										Local: "Transform",
									},
									Algorithm: "http://www.w3.org/2001/10/xml-exc-c14n#",
								},
							},
						},
						DigestMethod: EncryptionMethod{
							XMLName: xml.Name{
								Space: XMLNS_DS,
								Local: "DigestMethod",
							},
							Algorithm: "http://www.w3.org/2001/04/xmlenc#sha256",
						},
						DigestValue: SSAMLValue{
							XMLName: xml.Name{
								Space: XMLNS_DS,
								Local: "DigestValue",
							},
							Value: "",
						},
					},
				},
				SignatureValue: SSAMLValue{
					XMLName: xml.Name{
						Space: XMLNS_DS,
						Local: "SignatureValue",
					},
					Value: "",
				},
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
							Cert: input.IssuerCertString,
						},
					},
				},
			},
			Subject: Subject{
				XMLName: xml.Name{
					Space: XMLNS_ASSERT,
					Local: "Subject",
				},
				NameID: NameID{
					XMLName: xml.Name{
						Space: XMLNS_ASSERT,
						Local: "NameID",
					},
					Format:        input.NameIdFormat,
					NameQualifier: &input.RequestEntityId,
					Value:         input.NameId,
				},
				SubjectConfirmation: SubjectConfirmation{
					XMLName: xml.Name{
						Space: XMLNS_ASSERT,
						Local: "SubjectConfirmation",
					},
					Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
					SubjectConfirmationData: SubjectConfirmationData{
						XMLName: xml.Name{
							Space: XMLNS_ASSERT,
							Local: "SubjectConfirmationData",
						},
						InResponseTo: responseTo,
						Recipient:    input.AssertionConsumerServiceURL,
						NotOnOrAfter: until,
					},
				},
			},
			Conditions: Conditions{
				XMLName: xml.Name{
					Space: XMLNS_ASSERT,
					Local: "Conditions",
				},
				NotBefore:            &now,
				NotOnOrAfter:         until,
				AudienceRestrictions: []AudienceRestriction{},
			},
			AttributeStatement: &AttributeStatement{
				XMLName: xml.Name{
					Space: XMLNS_ASSERT,
					Local: "AttributeStatement",
				},
				Attributes: []Attribute{},
			},
			AuthnStatement: AuthnStatement{
				XMLName: xml.Name{
					Space: XMLNS_ASSERT,
					Local: "AuthnStatement",
				},
				AuthnInstant: now,
				SessionIndex: assertId,
				SubjectLocality: &SubjectLocality{
					XMLName: xml.Name{
						Space: XMLNS_ASSERT,
						Local: "SubjectLocality",
					},
					Address: input.RequestEntityId,
				},
				AuthnContext: AuthnContext{
					XMLName: xml.Name{
						Space: XMLNS_ASSERT,
						Local: "AuthnContext",
					},
					AuthnContextClassRef: AuthnContextClassRef{
						XMLName: xml.Name{
							Space: XMLNS_ASSERT,
							Local: "AuthnContextClassRef",
						},
						// Value: "urn:oasis:names:tc:SAML:2.0:ac:classes:unspecified",
						Value: "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
					},
				},
			},
		},
	}

	if len(input.AudienceRestriction) > 0 {
		resp.AddAudienceRestriction(input.AudienceRestriction)
	}

	for _, attr := range input.Attributes {
		resp.AddAttribute(attr.Name, attr.FriendlyName, attr.NameFormat, attr.Values)
	}

	return resp
}

// AddAttribute add strong attribute to the Response
func (r *Response) AddAttribute(name string, friendlyName string, nameFormat string, values []string) {
	attr := Attribute{
		XMLName: xml.Name{
			Space: XMLNS_ASSERT,
			Local: "Attribute",
		},
		Name:            name,
		AttributeValues: []AttributeValue{},
	}
	if len(friendlyName) > 0 {
		attr.FriendlyName = &friendlyName
	}
	if len(nameFormat) > 0 {
		attr.NameFormat = &nameFormat
	}
	for _, value := range values {
		attrValue := AttributeValue{
			XMLName: xml.Name{
				Space: XMLNS_ASSERT,
				Local: "AttributeValue",
			},
			Type:  "xs:string",
			Value: value,
		}
		attr.AttributeValues = append(attr.AttributeValues, attrValue)
	}
	r.Assertion.AttributeStatement.Attributes = append(r.Assertion.AttributeStatement.Attributes, attr)
}

func (r *Response) AddAudienceRestriction(value string) {
	restrict := AudienceRestriction{
		XMLName: xml.Name{
			Space: XMLNS_ASSERT,
			Local: "AudienceRestriction",
		},
		Audience: Audience{
			XMLName: xml.Name{
				Space: XMLNS_ASSERT,
				Local: "Audience",
			},
			Value: value,
		},
	}
	r.Assertion.Conditions.AudienceRestrictions = append(r.Assertion.Conditions.AudienceRestrictions, restrict)
}

func (saml *SSAMLInstance) UnmarshalResponse(xmlText []byte) (*Response, error) {
	resp := Response{}
	err := xml.Unmarshal(xmlText, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "xml.Unmarshal response")
	}
	if resp.EncryptedAssertion != nil {
		asserText, err := resp.EncryptedAssertion.EncryptedData.decryptData(saml.privateKey)
		if err != nil {
			return nil, errors.Wrap(err, "EncryptedAssertion.EncryptedData.decryptData")
		}

		assertion := Assertion{}
		err = xml.Unmarshal(asserText, &assertion)
		if err != nil {
			return nil, errors.Wrap(err, "xml.Unmarshal assertion")
		}

		resp.Assertion = &assertion
	}

	return &resp, nil
}

func (samlResp Response) FetchAttribtues() map[string][]string {
	ret := make(map[string][]string)
	if samlResp.Assertion != nil && samlResp.Assertion.AttributeStatement != nil {
		for _, attr := range samlResp.Assertion.AttributeStatement.Attributes {
			values := make([]string, len(attr.AttributeValues))
			for i := range values {
				values[i] = attr.AttributeValues[i].Value
			}
			ret[attr.Name] = values
		}
	}
	return ret
}

func (samlResp Response) IsSuccess() bool {
	return samlResp.Status.StatusCode.Value == STATUS_SUCCESS
}
