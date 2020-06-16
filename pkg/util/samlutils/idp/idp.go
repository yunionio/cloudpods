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

package idp

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/samlutils"
)

type OnSpInitiatedLogin func(ctx context.Context, sp *SSAMLServiceProvider) samlutils.SSAMLSpInitiatedLoginData
type OnIdpInitiatedLogin func(ctx context.Context, sp *SSAMLServiceProvider, state string) samlutils.SSAMLIdpInitiatedLoginData
type OnLogout func(ctx context.Context) string

type SSAMLIdpInstance struct {
	saml *samlutils.SSAMLInstance

	metadataPath        string
	redirectLoginPath   string
	redirectLogoutPath  string
	idpInitiatedSSOPath string

	serviceProviders []*SSAMLServiceProvider

	onSpInitiatedLogin  OnSpInitiatedLogin
	onIdpInitiatedLogin OnIdpInitiatedLogin
	onLogout            OnLogout

	htmlTemplate string
}

func NewIdpInstance(saml *samlutils.SSAMLInstance, spLoginFunc OnSpInitiatedLogin, idpLoginFunc OnIdpInitiatedLogin, logoutFunc OnLogout) *SSAMLIdpInstance {
	return &SSAMLIdpInstance{
		saml:                saml,
		onSpInitiatedLogin:  spLoginFunc,
		onIdpInitiatedLogin: idpLoginFunc,
		onLogout:            logoutFunc,
	}
}

func (idp *SSAMLIdpInstance) AddHandlers(app *appsrv.Application, prefix string) {
	idp.metadataPath = httputils.JoinPath(prefix, "metadata")
	idp.redirectLoginPath = httputils.JoinPath(prefix, "redirect/login")
	idp.redirectLogoutPath = httputils.JoinPath(prefix, "redirect/logout")
	idp.idpInitiatedSSOPath = httputils.JoinPath(prefix, "sso")

	app.AddHandler("GET", idp.metadataPath, idp.metadataHandler)
	app.AddHandler("GET", idp.redirectLoginPath, idp.redirectLoginHandler)
	app.AddHandler("GET", idp.redirectLogoutPath, idp.redirectLogoutHandler)
	app.AddHandler("GET", idp.idpInitiatedSSOPath, idp.idpInitiatedSSOHandler)

	log.Infof("IDP metadata: %s", idp.getMetadataUrl())
	log.Infof("IDP redirect login: %s", idp.getRedirectLoginUrl())
	log.Infof("IDP redirect logout: %s", idp.getRedirectLogoutUrl())
	log.Infof("IDP initated SSO: %s", idp.getIdpInitiatedSSOUrl())
}

func (idp *SSAMLIdpInstance) SetHtmlTemplate(tmp string) error {
	if strings.Index(tmp, samlutils.HTML_SAML_FORM_TOKEN) < 0 {
		return errors.Wrapf(httperrors.ErrInvalidFormat, "no %s found", samlutils.HTML_SAML_FORM_TOKEN)
	}
	idp.htmlTemplate = tmp
	return nil
}

func (idp *SSAMLIdpInstance) AddSPMetadataFile(filename string) error {
	metadata, err := ioutil.ReadFile(filename)
	if err != nil {
		return errors.Wrap(err, "ioutil.ReadFile")
	}
	return idp.AddSPMetadata(metadata)
}

func (idp *SSAMLIdpInstance) AddSPMetadata(metadata []byte) error {
	ed, err := samlutils.ParseMetadata(metadata)
	if err != nil {
		return errors.Wrap(err, "samlutils.ParseMetadata")
	}
	sp := &SSAMLServiceProvider{desc: ed}
	err = sp.IsValid()
	if err != nil {
		return errors.Wrap(err, "NewSAMLServiceProvider")
	}
	log.Debugf("Register SP metadata: %s", sp.GetEntityId())
	idp.serviceProviders = append(idp.serviceProviders, sp)
	return nil
}

func (idp *SSAMLIdpInstance) getMetadataUrl() string {
	return httputils.JoinPath(idp.saml.GetEntityId(), idp.metadataPath)
}

func (idp *SSAMLIdpInstance) getRedirectLoginUrl() string {
	return httputils.JoinPath(idp.saml.GetEntityId(), idp.redirectLoginPath)
}

func (idp *SSAMLIdpInstance) getRedirectLogoutUrl() string {
	return httputils.JoinPath(idp.saml.GetEntityId(), idp.redirectLogoutPath)
}

func (idp *SSAMLIdpInstance) getIdpInitiatedSSOUrl() string {
	return httputils.JoinPath(idp.saml.GetEntityId(), idp.idpInitiatedSSOPath)
}

func (idp *SSAMLIdpInstance) metadataHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	desc := idp.getMetadata(ctx)
	appsrv.SendXmlWithIndent(w, nil, desc, true)
}

func (idp *SSAMLIdpInstance) redirectLoginHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	input := samlutils.SIdpRedirectLoginInput{}
	err := query.Unmarshal(&input)
	if err != nil {
		httperrors.InputParameterError(w, "query.Unmarshal error %s", err)
		return
	}
	log.Debugf("recv input %s", input)
	respHtml, err := idp.processLoginRequest(ctx, input)
	if err != nil {
		httperrors.InputParameterError(w, "parse parameter error %s", err)
		return
	}
	appsrv.SendHTML(w, respHtml)
}

func (idp *SSAMLIdpInstance) redirectLogoutHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log.Debugf("logout: %s", r.Header)
	html := idp.onLogout(ctx)
	appsrv.SendHTML(w, html)
}

func (idp *SSAMLIdpInstance) idpInitiatedSSOHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	input := samlutils.SIdpInitiatedLoginInput{}
	err := query.Unmarshal(&input)
	if err != nil {
		httperrors.InputParameterError(w, "unmarshal input fail %s", err)
		return
	}
	respHtml, err := idp.processIdpInitiatedLogin(ctx, input)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendHTML(w, respHtml)
}

func (idp *SSAMLIdpInstance) getMetadata(ctx context.Context) samlutils.EntityDescriptor {
	input := samlutils.SSAMLIdpMetadataInput{
		EntityId:          idp.saml.GetEntityId(),
		CertString:        idp.saml.GetCertString(),
		RedirectLoginUrl:  idp.getRedirectLoginUrl(),
		RedirectLogoutUrl: idp.getRedirectLogoutUrl(),
	}
	hostId := appctx.AppContextHostId(ctx)
	return samlutils.NewIdpMetadata(hostId, input)
}

func (idp *SSAMLIdpInstance) processLoginRequest(ctx context.Context, input samlutils.SIdpRedirectLoginInput) (string, error) {
	plainText, err := samlutils.SAMLDecode(input.SAMLRequest)
	if err != nil {
		return "", errors.Wrap(err, "samlutils.SAMLDecode")
	}

	log.Debugf("AuthnRequest: %s", string(plainText))

	authReq := samlutils.AuthnRequest{}
	err = xml.Unmarshal(plainText, &authReq)
	if err != nil {
		return "", errors.Wrap(err, "xml.Unmarshal")
	}

	sp := idp.getServiceProvider(authReq.Issuer.Issuer)
	if sp == nil {
		return "", errors.Wrapf(httperrors.ErrResourceNotFound, "issuer %s not found", authReq.Issuer.Issuer)
	}

	if len(authReq.Destination) > 0 && authReq.Destination != idp.getRedirectLoginUrl() {
		return "", errors.Wrapf(httperrors.ErrInputParameter, "Destination not match: get %s want %s", authReq.Destination, idp.getRedirectLoginUrl())
	}

	if authReq.AssertionConsumerServiceURL != sp.GetPostAssertionConsumerServiceUrl() {
		return "", errors.Wrapf(httperrors.ErrInputParameter, "AssertionConsumerServiceURL not match: get %s want %s", authReq.AssertionConsumerServiceURL, sp.GetPostAssertionConsumerServiceUrl())
	}

	resp, err := idp.getLoginResponse(ctx, authReq, sp)
	if err != nil {
		return "", errors.Wrap(err, "getLoginResponse")
	}

	form, err := idp.samlResponse2Form(authReq.AssertionConsumerServiceURL, resp, input.RelayState)
	if err != nil {
		return "", errors.Wrap(err, "samlResponse2Form")
	}

	return form, nil
}

func (idp *SSAMLIdpInstance) samlResponse2Form(url string, resp *samlutils.Response, state string) (string, error) {
	respXml, err := xml.Marshal(resp)
	if err != nil {
		return "", errors.Wrap(err, "xml.Marshal")
	}

	signed, err := idp.saml.SignXML(string(respXml))
	if err != nil {
		return "", errors.Wrap(err, "saml.SignXML")
	}

	log.Debugf("ResponseXML: %s", signed)

	samlResp := base64.StdEncoding.EncodeToString([]byte(signed))

	form := samlutils.SAMLForm(url, map[string]string{
		"SAMLResponse": samlResp,
		"RelayState":   state,
	})
	template := samlutils.DEFAULT_HTML_TEMPLATE
	if len(idp.htmlTemplate) > 0 {
		template = idp.htmlTemplate
	}
	form = strings.Replace(template, samlutils.HTML_SAML_FORM_TOKEN, form, 1)
	return form, nil
}

func (idp *SSAMLIdpInstance) getServiceProvider(eId string) *SSAMLServiceProvider {
	for _, sp := range idp.serviceProviders {
		if sp.GetEntityId() == eId {
			return sp
		}
	}
	return nil
}

func (idp *SSAMLIdpInstance) getLoginResponse(ctx context.Context, req samlutils.AuthnRequest, sp *SSAMLServiceProvider) (*samlutils.Response, error) {
	data := idp.onSpInitiatedLogin(ctx, sp)
	input := samlutils.SSAMLResponseInput{
		IssuerCertString:            idp.saml.GetCertString(),
		IssuerEntityId:              idp.saml.GetEntityId(),
		RequestID:                   req.ID,
		RequestEntityId:             req.Issuer.Issuer,
		AssertionConsumerServiceURL: req.AssertionConsumerServiceURL,
		SSAMLSpInitiatedLoginData:   data,
	}
	resp := samlutils.NewResponse(input)
	return &resp, nil
}

func (idp *SSAMLIdpInstance) processIdpInitiatedLogin(ctx context.Context, input samlutils.SIdpInitiatedLoginInput) (string, error) {
	sp := idp.getServiceProvider(input.EntityID)
	if sp == nil {
		return "", errors.Wrapf(httperrors.ErrResourceNotFound, "issuer %s not found", input.EntityID)
	}
	var state []byte
	if len(input.State) > 0 {
		var err error
		state, err = samlutils.SAMLDecode(input.State)
		if err != nil {
			return "", errors.Wrapf(httperrors.ErrInputParameter, "invalid state %s: %s", input.State, err)
		}
	}
	data := idp.onIdpInitiatedLogin(ctx, sp, string(state))
	respInput := samlutils.SSAMLResponseInput{
		IssuerCertString:            idp.saml.GetCertString(),
		IssuerEntityId:              idp.saml.GetEntityId(),
		RequestID:                   "",
		RequestEntityId:             sp.GetEntityId(),
		AssertionConsumerServiceURL: sp.GetPostAssertionConsumerServiceUrl(),
		SSAMLSpInitiatedLoginData:   data.SSAMLSpInitiatedLoginData,
	}
	resp := samlutils.NewResponse(respInput)
	form, err := idp.samlResponse2Form(sp.GetPostAssertionConsumerServiceUrl(), &resp, data.RelayState)
	if err != nil {
		return "", errors.Wrap(err, "samlResponse2Form")
	}
	return form, nil
}
