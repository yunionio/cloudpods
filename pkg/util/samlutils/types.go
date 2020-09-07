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

import "encoding/xml"

type DigestMethod struct {
	XMLName xml.Name

	Algorithm string `xml:"Algorithm,attr"`
}

type SigningMethod struct {
	XMLName xml.Name

	Algorithm string `xml:"Algorithm,attr"`
}

type RequestInitiator struct {
	XMLName xml.Name

	Binding  string `xml:"Binding,attr"`
	Location string `xml:"Location,attr"`
}

type SXMLText struct {
	XMLName xml.Name

	Lang string `xml:"xml:lang,attr"`
	Text string `xml:",innerxml"`
}

type SXMLLogo struct {
	XMLName xml.Name

	Height string `xml:"height,attr"`
	Width  string `xml:"width,attr"`
	URL    string `xml:",innerxml"`
}

type SSAMLUIInfo struct {
	XMLName xml.Name

	DisplayName SXMLText `xml:"DisplayName"`
	Description SXMLText `xml:"Description"`
	Logo        SXMLLogo `xml:"Logo"`
}

type SSAMLScope struct {
	XMLName xml.Name

	Regexp string `xml:"regexp,attr"`
	Scope  string `xml:",innerxml"`
}

type Extensions struct {
	XMLName xml.Name

	// Alg    string `xml:"alg,attr"`
	// MDAttr string `xml:"mdattr,attr"`
	// MDRPI  string `xml:"mdrpi,attr"`

	// EntityAttributes string `xml:"EntityAttributes"`

	SigningMethods []SigningMethod `xml:"SigningMethod"`
	DigestMethods  []DigestMethod  `xml:"DigestMethod"`

	RequestInitiator *RequestInitiator `xml:"RequestInitiator"`

	UIInfo *SSAMLUIInfo `xml:"UIInfo"`

	Scope *SSAMLScope `xml:"Scope"`
}

type X509Certificate struct {
	XMLName xml.Name

	Cert string `xml:",innerxml"`
}

type X509Data struct {
	XMLName xml.Name

	X509Certificate X509Certificate `xml:"X509Certificate"`
}

type KeyInfo struct {
	XMLName xml.Name

	X509Data     *X509Data     `xml:"X509Data"`
	EncryptedKey *EncryptedKey `xml:"EncryptedKey"`
}

type EncryptionMethod struct {
	XMLName xml.Name

	Algorithm string `xml:"Algorithm,attr"`

	DigestMethod *DigestMethod `xml:"DigestMethod"`
}

type KeyDescriptor struct {
	XMLName xml.Name

	Use string `xml:"use,attr"`

	KeyInfo KeyInfo `xml:"KeyInfo"`

	EncryptionMethods []EncryptionMethod `xml:"EncryptionMethod"`
}

type SSAMLService struct {
	XMLName xml.Name

	Binding   string  `xml:"Binding,attr"`
	Location  string  `xml:"Location,attr"`
	Index     *string `xml:"index,attr"`
	IsDefault *string `xml:"isDefault,attr"`
}

type SSAMLNameIDFormat struct {
	XMLName xml.Name

	Format string `xml:",innerxml"`
}

type RequestedAttribute struct {
	XMLName xml.Name

	IsRequired   string `xml:"isRequired,attr"`
	Name         string `xml:"Name,attr"`
	FriendlyName string `xml:"FriendlyName,attr"`
}

type AttributeConsumingService struct {
	XMLName xml.Name

	Index string `xml:"index,attr"`

	ServiceName SXMLText `xml:"ServiceName"`

	RequestedAttributes []RequestedAttribute `xml:"RequestedAttribute"`
}

type SSODescriptor struct {
	XMLName xml.Name

	AuthnRequestsSigned        *string `xml:"AuthnRequestsSigned,attr"`
	WantAssertionsSigned       *string `xml:"WantAssertionsSigned,attr"`
	ProtocolSupportEnumeration string  `xml:"protocolSupportEnumeration,attr"`

	Extensions *Extensions `xml:"Extensions"`

	KeyDescriptors []KeyDescriptor `xml:"KeyDescriptor"`

	ArtifactResolutionServices []SSAMLService `xml:"ArtifactResolutionService"`

	SingleLogoutServices []SSAMLService `xml:"SingleLogoutService"`
	ManageNameIDServices []SSAMLService `xml:"ManageNameIDService"`

	NameIDFormat         []SSAMLNameIDFormat `xml:"NameIDFormat"`
	SingleSignOnServices []SSAMLService      `xml:"SingleSignOnService"`

	AssertionConsumerServices []SSAMLService `xml:"AssertionConsumerService"`

	AttributeConsumingServices []AttributeConsumingService `xml:"AttributeConsumingService"`
}

type SSAMLValue struct {
	XMLName xml.Name

	Value string `xml:",innerxml"`
}

type Transforms struct {
	XMLName xml.Name

	Transforms []EncryptionMethod `xml:"Transform"`
}

type Reference struct {
	XMLName xml.Name

	URI string `xml:"URI,attr"`

	Transforms   Transforms       `xml:"Transforms"`
	DigestMethod EncryptionMethod `xml:"DigestMethod"`
	DigestValue  SSAMLValue       `xml:"DigestValue"`
}

type SignedInfo struct {
	XMLName xml.Name

	CanonicalizationMethod EncryptionMethod `xml:"CanonicalizationMethod"`
	SignatureMethod        EncryptionMethod `xml:"SignatureMethod"`

	Reference Reference `xml:"Reference"`
}

type Signature struct {
	XMLName xml.Name

	SignedInfo     SignedInfo `xml:"SignedInfo"`
	SignatureValue SSAMLValue `xml:"SignatureValue"`
	KeyInfo        KeyInfo    `xml:"KeyInfo"`
}

type Organization struct {
	XMLName xml.Name

	OrganizationName        SXMLText `xml:"OrganizationName"`
	OrganizationDisplayName SXMLText `xml:"OrganizationDisplayName"`
	OrganizationURL         SXMLText `xml:"OrganizationURL"`
}

type EntityDescriptor struct {
	XMLName xml.Name

	// Id *string `xml:"ID,attr"`
	EntityId string `xml:"entityID,attr"`

	Extensions *Extensions `xml:"Extensions"`
	Signature  *Signature  `xml:"Signature"`

	SPSSODescriptor  *SSODescriptor `xml:"SPSSODescriptor"`
	IDPSSODescriptor *SSODescriptor `xml:"IDPSSODescriptor"`

	Organization *Organization `xml:"Organization"`
}

func (ed EntityDescriptor) String() string {
	str, _ := xml.MarshalIndent(ed, "", "  ")
	return string(str)
}

type SIdpRedirectLoginInput struct {
	SAMLRequest string `json:"SAMLRequest,ignoreempty"`
	RelayState  string `json:"RelayState,ignoreempty"`
	SigAlg      string `json:"SigAlg,ignoreempty"`
	Signature   string `json:"Signature,ignoreempty"`
}

type SIdpInitiatedLoginInput struct {
	EntityID string `json:"EntityID"`
	IdpId    string `json:"IdpId"`
}

type Issuer struct {
	XMLName xml.Name

	Format *string `xml:"Format,attr"`

	Issuer string `xml:",innerxml"`
}

type NameIDPolicy struct {
	XMLName xml.Name

	AllowCreate     string  `xml:"AllowCreate,attr"`
	Format          string  `xml:"Format,attr"`
	SPNameQualifier *string `xml:"SPNameQualifier,attr"`
}

type AuthnRequest struct {
	XMLName xml.Name

	AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr"`
	Destination                 string `xml:"Destination,attr"`
	ForceAuthn                  string `xml:"ForceAuthn,attr"`
	ID                          string `xml:"ID,attr"`
	IsPassive                   string `xml:"IsPassive,attr"`
	IssueInstant                string `xml:"IssueInstant,attr"`
	ProtocolBinding             string `xml:"ProtocolBinding,attr"`
	Version                     string `xml:"Version,attr"`

	Issuer       Issuer       `xml:"Issuer"`
	NameIDPolicy NameIDPolicy `xml:"NameIDPolicy"`
}

type StatusCode struct {
	XMLName xml.Name

	Value string `xml:"Value,attr"`
}

type StatusMessage struct {
	XMLName xml.Name

	Message string `xml:",innerxml"`
}

type Status struct {
	XMLName xml.Name

	StatusCode    StatusCode     `xml:"StatusCode"`
	StatusMessage *StatusMessage `xml:"StatusMessage"`
}

type Response struct {
	XMLName xml.Name

	ID           string  `xml:"ID,attr"`
	InResponseTo *string `xml:"InResponseTo,attr"`
	Version      string  `xml:"Version,attr"`
	IssueInstant string  `xml:"IssueInstant,attr"`
	Destination  string  `xml:"Destination,attr"`

	Issuer Issuer `xml:"Issuer"`
	Status Status `xml:"Status"`

	Assertion          *Assertion          `xml:"Assertion"`
	EncryptedAssertion *EncryptedAssertion `xml:"EncryptedAssertion"`
}

type Assertion struct {
	XMLName xml.Name

	ID           string `xml:"ID,attr"`
	Version      string `xml:"Version,attr"`
	IssueInstant string `xml:"IssueInstant,attr"`

	Issuer             Issuer              `xml:"Issuer"`
	Signature          *Signature          `xml:"Signature"`
	Subject            Subject             `xml:"Subject"`
	Conditions         Conditions          `xml:"Conditions"`
	AttributeStatement *AttributeStatement `xml:"AttributeStatement"`
	AuthnStatement     AuthnStatement      `xml:"AuthnStatement"`
}

type Subject struct {
	XMLName xml.Name

	NameID NameID `xml:"NameID"`

	SubjectConfirmation SubjectConfirmation `xml:"SubjectConfirmation"`
}

type NameID struct {
	XMLName xml.Name

	Format        string  `xml:"Format,attr"`
	NameQualifier *string `xml:"NameQualifier,attr"`

	Value string `xml:",innerxml"`
}

type SubjectConfirmation struct {
	XMLName xml.Name

	Method string `xml:"Method,attr"`

	SubjectConfirmationData SubjectConfirmationData `xml:"SubjectConfirmationData"`
}

type SubjectConfirmationData struct {
	XMLName xml.Name

	InResponseTo *string `xml:"InResponseTo,attr"`
	Recipient    string  `xml:"Recipient,attr"`
	NotBefore    *string `xml:"NotBefore,attr"`
	NotOnOrAfter string  `xml:"NotOnOrAfter,attr"`
}

type Conditions struct {
	XMLName xml.Name

	NotBefore    *string `xml:"NotBefore,attr"`
	NotOnOrAfter string  `xml:"NotOnOrAfter,attr"`

	AudienceRestrictions []AudienceRestriction `xml:"AudienceRestriction"`
}

type AudienceRestriction struct {
	XMLName xml.Name

	Audience Audience `xml:"Audience"`
}

type Audience struct {
	XMLName xml.Name

	Value string `xml:",innerxml"`
}

type AttributeStatement struct {
	XMLName xml.Name

	Attributes []Attribute `xml:"Attribute"`
}

type Attribute struct {
	XMLName xml.Name

	FriendlyName *string `xml:"FriendlyName,attr"`
	Name         string  `xml:"Name,attr"`
	NameFormat   *string `xml:"NameFormat,attr"`

	AttributeValues []AttributeValue `xml:"AttributeValue"`
}

type AttributeValue struct {
	XMLName xml.Name

	Type string `xml:"type,attr"`

	Value string `xml:",innerxml"`
}

type AuthnStatement struct {
	XMLName xml.Name

	AuthnInstant string `xml:"AuthnInstant,attr"`
	SessionIndex string `xml:"SessionIndex,attr"`

	SubjectLocality *SubjectLocality `xml:"SubjectLocality"`

	AuthnContext AuthnContext `xml:"AuthnContext"`
}

type SubjectLocality struct {
	XMLName xml.Name

	Address string `xml:"Address,attr"`
}

type AuthnContext struct {
	XMLName xml.Name

	AuthnContextClassRef AuthnContextClassRef `xml:"AuthnContextClassRef"`
}

type AuthnContextClassRef struct {
	XMLName xml.Name

	Value string `xml:",innerxml"`
}

type SSpInitiatedLoginInput struct {
	EntityID string `json:"EntityID"`
}

type EncryptedAssertion struct {
	XMLName xml.Name

	EncryptedData EncryptedData `xml:"EncryptedData"`
}

type EncryptedData struct {
	XMLName xml.Name

	Id   string `xml:"Id,attr"`
	Type string `xml:"Type,attr"`

	EncryptionMethod EncryptionMethod `xml:"EncryptionMethod"`
	KeyInfo          KeyInfo          `xml:"KeyInfo"`
	CipherData       CipherData       `xml:"CipherData"`
}

type CipherData struct {
	XMLName xml.Name

	CipherValue CipherValue `xml:"CipherValue"`
}

type CipherValue struct {
	XMLName xml.Name

	Value string `xml:",innerxml"`
}

type EncryptedKey struct {
	XMLName xml.Name

	Id        string `xml:"Id,attr"`
	Recipient string `xml:"Recipient,attr"`

	EncryptionMethod EncryptionMethod `xml:"EncryptionMethod"`
	KeyInfo          KeyInfo          `xml:"KeyInfo"`
	CipherData       CipherData       `xml:"CipherData"`
}
