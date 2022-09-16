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

package sp

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"os"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/samlutils"
)

type SSAMLAttribute struct {
	Name         string
	FriendlyName string
	Values       []string
}
type SSAMLAssertionConsumeResult struct {
	SSAMLSpInitiatedLoginRequest
	Attributes []SSAMLAttribute
}

type SSAMLSpInitiatedLoginRequest struct {
	RequestID  string
	RelayState string
}

type OnSAMLAssertionConsume func(ctx context.Context, w http.ResponseWriter, idp *SSAMLIdentityProvider, result SSAMLAssertionConsumeResult) error
type OnSAMLSpInitiatedLogin func(ctx context.Context, idp *SSAMLIdentityProvider) (SSAMLSpInitiatedLoginRequest, error)

type SSAMLSpInstance struct {
	saml *samlutils.SSAMLInstance

	serviceName string

	metadataPath          string
	assertionConsumerPath string
	spInitiatedSSOPath    string

	assertionConsumerUri string

	identityProviders []*SSAMLIdentityProvider

	onSAMLAssertionConsume OnSAMLAssertionConsume
	onSAMLSpInitiatedLogin OnSAMLSpInitiatedLogin

	htmlTemplate string
}

func NewSpInstance(saml *samlutils.SSAMLInstance, serviceName string, consumeFunc OnSAMLAssertionConsume, loginFunc OnSAMLSpInitiatedLogin) *SSAMLSpInstance {
	return &SSAMLSpInstance{
		saml:                   saml,
		serviceName:            serviceName,
		onSAMLAssertionConsume: consumeFunc,
		onSAMLSpInitiatedLogin: loginFunc,
	}
}

func (sp *SSAMLSpInstance) GetIdentityProviders() []*SSAMLIdentityProvider {
	return sp.identityProviders
}

func (sp *SSAMLSpInstance) AddIdpMetadataFile(filename string) error {
	metadata, err := os.ReadFile(filename)
	if err != nil {
		return errors.Wrap(err, "os.ReadFile")
	}
	return sp.AddIdpMetadata(metadata)
}

func (sp *SSAMLSpInstance) AddIdpMetadata(metadata []byte) error {
	ed, err := samlutils.ParseMetadata(metadata)
	if err != nil {
		return errors.Wrap(err, "samlutils.ParseMetadata")
	}
	idp, err := NewSAMLIdpFromDescriptor(ed)
	if err != nil {
		return errors.Wrap(err, "NewSAMLIdpFromDescriptor")
	}
	err = idp.IsValid()
	if err != nil {
		return errors.Wrap(err, "Invalid SAMLIdentityProvider")
	}
	log.Debugf("Register Idp metadata: %s", idp.GetEntityId())
	sp.identityProviders = append(sp.identityProviders, idp)
	return nil
}

func (sp *SSAMLSpInstance) AddIdp(entityId, redirectSsoUrl string) error {
	idp := NewSAMLIdp(entityId, redirectSsoUrl)
	err := idp.IsValid()
	if err != nil {
		return errors.Wrap(err, "Invalid SAMLIdentityProvider")
	}
	log.Debugf("Register Idp metadata: %s", idp.GetEntityId())
	sp.identityProviders = append(sp.identityProviders, idp)
	return nil
}

func (sp *SSAMLSpInstance) AddHandlers(app *appsrv.Application, prefix string) {
	sp.metadataPath = httputils.JoinPath(prefix, "metadata")
	sp.assertionConsumerPath = httputils.JoinPath(prefix, "acs")
	sp.spInitiatedSSOPath = httputils.JoinPath(prefix, "sso")

	app.AddHandler("GET", sp.metadataPath, sp.metadataHandler)
	app.AddHandler("POST", sp.assertionConsumerPath, sp.assertionConsumeHandler)
	app.AddHandler("GET", sp.spInitiatedSSOPath, sp.spInitiatedSSOHandler)

	log.Infof("SP metadata: %s", sp.getMetadataUrl())
	log.Infof("SP assertion consumer: %s", sp.getAssertionConsumerUrl())
	log.Infof("SP initated SSO: %s", sp.getSpInitiatedSSOUrl())
}

func (sp *SSAMLSpInstance) SetAssertionConsumerUri(uri string) {
	sp.assertionConsumerUri = uri
}

func (sp *SSAMLSpInstance) getMetadataUrl() string {
	return httputils.JoinPath(sp.saml.GetEntityId(), sp.metadataPath)
}

func (sp *SSAMLSpInstance) getAssertionConsumerUrl() string {
	if len(sp.assertionConsumerUri) > 0 {
		return sp.assertionConsumerUri
	}
	return httputils.JoinPath(sp.saml.GetEntityId(), sp.assertionConsumerPath)
}

func (sp *SSAMLSpInstance) getSpInitiatedSSOUrl() string {
	return httputils.JoinPath(sp.saml.GetEntityId(), sp.spInitiatedSSOPath)
}

func (sp *SSAMLSpInstance) metadataHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	desc := sp.GetMetadata()
	appsrv.SendXmlWithIndent(w, nil, desc, true)
}

func (sp *SSAMLSpInstance) assertionConsumeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	samlResponse := r.FormValue("SAMLResponse")
	relayState := r.FormValue("RelayState")

	err := sp.processAssertionConsumer(ctx, w, samlResponse, relayState)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
}

func (sp *SSAMLSpInstance) spInitiatedSSOHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	input := samlutils.SSpInitiatedLoginInput{}
	err := query.Unmarshal(&input)
	if err != nil {
		httperrors.InputParameterError(ctx, w, "unmarshal input fail %s", err)
		return
	}
	redirectUrl, err := sp.ProcessSpInitiatedLogin(ctx, input)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendRedirect(w, redirectUrl)
}

func (sp *SSAMLSpInstance) GetMetadata() samlutils.EntityDescriptor {
	input := samlutils.SSAMLSpMetadataInput{
		EntityId:    sp.saml.GetEntityId(),
		CertString:  sp.saml.GetCertString(),
		ServiceName: sp.serviceName,

		AssertionConsumerUrl: sp.getAssertionConsumerUrl(),
		RequestedAttributes: []samlutils.RequestedAttribute{
			{
				IsRequired:   "false",
				Name:         "userId",
				FriendlyName: "userId",
			},
			{
				IsRequired:   "false",
				Name:         "projectId",
				FriendlyName: "projectId",
			},
			{
				IsRequired:   "false",
				Name:         "roleId",
				FriendlyName: "roleId",
			},
		},
	}
	return samlutils.NewSpMetadata(input)
}

func (sp *SSAMLSpInstance) getIdentityProvider(eId string) *SSAMLIdentityProvider {
	for _, sp := range sp.identityProviders {
		if sp.GetEntityId() == eId {
			return sp
		}
	}
	return nil
}

func (sp *SSAMLSpInstance) processAssertionConsumer(ctx context.Context, w http.ResponseWriter, samlResponse string, relayState string) error {
	samlRespBytes, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return errors.Wrap(err, "base64.StdEncoding.DecodeString")
	}
	log.Debugf("samlResponse: %s", string(samlRespBytes))

	samlResp, err := sp.saml.UnmarshalResponse(samlRespBytes)
	if err != nil {
		return errors.Wrap(err, "saml.UnmarshalResponse")
	}

	/*_, err = samlutils.ValidateXML(string(samlRespBytes))
	if err != nil {
		return errors.Wrap(err, "ValidateXML")
	}*/

	if !samlResp.IsSuccess() {
		return errors.Wrapf(httperrors.ErrInvalidCredential, "SAML authenticate fail: %s", samlResp.Status.StatusCode.Value)
	}

	idp := sp.getIdentityProvider(samlResp.Issuer.Issuer)
	if idp == nil {
		return errors.Wrapf(httperrors.ErrResourceNotFound, "issuer %s not found", samlResp.Issuer.Issuer)
	}

	result := SSAMLAssertionConsumeResult{}

	if samlResp.InResponseTo != nil {
		result.RequestID = *samlResp.InResponseTo
	}
	result.RelayState = relayState

	if samlResp.Assertion != nil && samlResp.Assertion.AttributeStatement != nil {
		result.Attributes = make([]SSAMLAttribute, len(samlResp.Assertion.AttributeStatement.Attributes))
		for i, attr := range samlResp.Assertion.AttributeStatement.Attributes {
			values := make([]string, len(attr.AttributeValues))
			for i := range values {
				values[i] = attr.AttributeValues[i].Value
			}
			result.Attributes[i].Name = attr.Name
			if attr.FriendlyName != nil {
				result.Attributes[i].FriendlyName = *attr.FriendlyName
			}
			result.Attributes[i].Values = values
		}
	}

	err = sp.onSAMLAssertionConsume(ctx, w, idp, result)
	if err != nil {
		return errors.Wrap(err, "onSAMLAssertionConsume")
	}

	return nil
}

func (sp *SSAMLSpInstance) ProcessSpInitiatedLogin(ctx context.Context, input samlutils.SSpInitiatedLoginInput) (string, error) {
	idp := sp.getIdentityProvider(input.EntityID)
	if idp == nil {
		return "", errors.Wrapf(httperrors.ErrResourceNotFound, "issuer %s not found", input.EntityID)
	}
	loginReq, err := sp.onSAMLSpInitiatedLogin(ctx, idp)
	if err != nil {
		return "", errors.Wrap(err, "onSAMLSpInitiatedLogin")
	}
	reqInput := samlutils.SSAMLRequestInput{
		AssertionConsumerServiceURL: sp.getAssertionConsumerUrl(),
		Destination:                 idp.getRedirectSSOUrl(),
		RequestID:                   loginReq.RequestID,
		EntityID:                    sp.saml.GetEntityId(),
	}
	samlRequest := samlutils.NewRequest(reqInput)
	samlRequestXml, err := xml.Marshal(samlRequest)
	if err != nil {
		return "", errors.Wrap(err, "xml.Marshal")
	}
	reqStr, err := samlutils.SAMLEncode(samlRequestXml)
	if err != nil {
		return "", errors.Wrap(err, "SAMLEncode")
	}
	queryInput := samlutils.SIdpRedirectLoginInput{
		SAMLRequest: reqStr,
		RelayState:  loginReq.RelayState,
	}
	queryStr := jsonutils.Marshal(queryInput).QueryString()
	redirectUrl := idp.getRedirectSSOUrl()
	if strings.IndexByte(redirectUrl, '?') > 0 {
		// non-empty query string
		redirectUrl += "&" + queryStr
	} else {
		redirectUrl += "?" + queryStr
	}
	return redirectUrl, nil
}
