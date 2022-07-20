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
	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/samlutils"
)

const (
	IDP_ID_KEY = "<idp_id>"

	langTemplateKey = "lang_template_key"
)

type OnSpInitiatedLogin func(ctx context.Context, idpId string, sp *SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error)
type OnIdpInitiatedLogin func(ctx context.Context, sp *SSAMLServiceProvider, IdpId, redirectUrl string) (samlutils.SSAMLIdpInitiatedLoginData, error)
type OnLogout func(ctx context.Context, idpId string) string

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

	htmlTemplate i18n.Table
}

func NewIdpInstance(saml *samlutils.SSAMLInstance, spLoginFunc OnSpInitiatedLogin, idpLoginFunc OnIdpInitiatedLogin, logoutFunc OnLogout) *SSAMLIdpInstance {
	return &SSAMLIdpInstance{
		saml:                saml,
		onSpInitiatedLogin:  spLoginFunc,
		onIdpInitiatedLogin: idpLoginFunc,
		onLogout:            logoutFunc,
		htmlTemplate:        i18n.Table{},
	}
}

func (idp *SSAMLIdpInstance) AddHandlers(app *appsrv.Application, prefix string, middleware appsrv.TMiddleware) {
	idp.metadataPath = httputils.JoinPath(prefix, "metadata/"+IDP_ID_KEY)
	idp.redirectLoginPath = httputils.JoinPath(prefix, "redirect/login/"+IDP_ID_KEY)
	idp.redirectLogoutPath = httputils.JoinPath(prefix, "redirect/logout/"+IDP_ID_KEY)
	idp.idpInitiatedSSOPath = httputils.JoinPath(prefix, "sso")

	app.AddHandler("GET", idp.metadataPath, idp.metadataHandler)
	handler := idp.redirectLoginHandler
	if middleware != nil {
		handler = middleware(handler)
	}
	app.AddHandler("POST", idp.redirectLoginPath, handler)
	app.AddHandler("GET", idp.redirectLoginPath, handler)
	handler = idp.redirectLogoutHandler
	if middleware != nil {
		handler = middleware(handler)
	}
	app.AddHandler("GET", idp.redirectLogoutPath, handler)
	handler = idp.idpInitiatedSSOHandler
	if middleware != nil {
		handler = middleware(handler)
	}
	app.AddHandler("GET", idp.idpInitiatedSSOPath, handler)

	log.Infof("IDP metadata: %s", idp.getMetadataUrl(IDP_ID_KEY))
	log.Infof("IDP redirect login: %s", idp.getRedirectLoginUrl(IDP_ID_KEY))
	log.Infof("IDP redirect logout: %s", idp.getRedirectLogoutUrl(IDP_ID_KEY))
	log.Infof("IDP initated SSO: %s", idp.getIdpInitiatedSSOUrl())
}

func (idp *SSAMLIdpInstance) SetHtmlTemplate(entry i18n.TableEntry) error {
	for _, tmp := range entry {
		if strings.Index(tmp, samlutils.HTML_SAML_FORM_TOKEN) < 0 {
			return errors.Wrapf(httperrors.ErrInvalidFormat, "no %s found", samlutils.HTML_SAML_FORM_TOKEN)
		}
	}
	idp.htmlTemplate.Set(langTemplateKey, entry)
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

func (idp *SSAMLIdpInstance) getMetadataUrl(idpId string) string {
	return strings.Replace(httputils.JoinPath(idp.saml.GetEntityId(), idp.metadataPath), IDP_ID_KEY, idpId, 1)
}

func (idp *SSAMLIdpInstance) getRedirectLoginUrl(idpId string) string {
	return strings.Replace(httputils.JoinPath(idp.saml.GetEntityId(), idp.redirectLoginPath), IDP_ID_KEY, idpId, 1)
}

func (idp *SSAMLIdpInstance) getRedirectLogoutUrl(idpId string) string {
	return strings.Replace(httputils.JoinPath(idp.saml.GetEntityId(), idp.redirectLogoutPath), IDP_ID_KEY, idpId, 1)
}

func (idp *SSAMLIdpInstance) getIdpInitiatedSSOUrl() string {
	return httputils.JoinPath(idp.saml.GetEntityId(), idp.idpInitiatedSSOPath)
}

func (idp *SSAMLIdpInstance) metadataHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := appctx.AppContextParams(ctx)
	idpId := params[IDP_ID_KEY]
	desc := idp.GetMetadata(idpId)
	appsrv.SendXmlWithIndent(w, nil, desc, true)
}

func (idp *SSAMLIdpInstance) redirectLoginHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params, query, _ := appsrv.FetchEnv(ctx, w, r)
	idpId := params[IDP_ID_KEY]
	input := samlutils.SIdpRedirectLoginInput{}
	err := query.Unmarshal(&input)
	if err != nil {
		httperrors.InputParameterError(ctx, w, "query.Unmarshal error %s", err)
		return
	}
	log.Debugf("recv input %s", input)
	respHtml, err := idp.processLoginRequest(ctx, idpId, input)
	if err != nil {
		httperrors.InputParameterError(ctx, w, "parse parameter error %s", err)
		return
	}
	appsrv.SendHTML(w, respHtml)
}

func (idp *SSAMLIdpInstance) redirectLogoutHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := appctx.AppContextParams(ctx)
	idpId := params[IDP_ID_KEY]
	log.Debugf("logout: %s", r.Header)
	html := idp.onLogout(ctx, idpId)
	appsrv.SendHTML(w, html)
}

func (idp *SSAMLIdpInstance) idpInitiatedSSOHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	input := samlutils.SIdpInitiatedLoginInput{}
	err := query.Unmarshal(&input)
	if err != nil {
		httperrors.InputParameterError(ctx, w, "unmarshal input fail %s", err)
		return
	}
	respHtml, err := idp.processIdpInitiatedLogin(ctx, input)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendHTML(w, respHtml)
}

func (idp *SSAMLIdpInstance) GetMetadata(idpId string) samlutils.EntityDescriptor {
	input := samlutils.SSAMLIdpMetadataInput{
		EntityId:          idp.saml.GetEntityId(),
		CertString:        idp.saml.GetCertString(),
		RedirectLoginUrl:  idp.getRedirectLoginUrl(idpId),
		RedirectLogoutUrl: idp.getRedirectLogoutUrl(idpId),
	}
	return samlutils.NewIdpMetadata(input)
}

func (idp *SSAMLIdpInstance) processLoginRequest(ctx context.Context, idpId string, input samlutils.SIdpRedirectLoginInput) (string, error) {
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

	if len(authReq.Destination) > 0 && authReq.Destination != idp.getRedirectLoginUrl(idpId) {
		return "", errors.Wrapf(httperrors.ErrInputParameter, "Destination not match: get %s want %s", authReq.Destination, idp.getRedirectLoginUrl(idpId))
	}

	if len(authReq.AssertionConsumerServiceURL) > 0 && authReq.AssertionConsumerServiceURL != sp.GetPostAssertionConsumerServiceUrl() {
		return "", errors.Wrapf(httperrors.ErrInputParameter, "AssertionConsumerServiceURL not match: get %s want %s", authReq.AssertionConsumerServiceURL, sp.GetPostAssertionConsumerServiceUrl())
	}

	sp.Username = input.Username
	resp, err := idp.getLoginResponse(ctx, authReq, idpId, sp)
	if err != nil {
		return "", errors.Wrap(err, "getLoginResponse")
	}

	form, err := idp.samlResponse2Form(ctx, sp.GetPostAssertionConsumerServiceUrl(), resp, input.RelayState)
	if err != nil {
		return "", errors.Wrap(err, "samlResponse2Form")
	}

	return form, nil
}

func (idp *SSAMLIdpInstance) samlResponse2Form(ctx context.Context, url string, resp *samlutils.Response, state string) (string, error) {
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
	_temp := idp.htmlTemplate.Lookup(ctx, langTemplateKey)
	if _temp != langTemplateKey {
		template = _temp
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

func (idp *SSAMLIdpInstance) getLoginResponse(ctx context.Context, req samlutils.AuthnRequest, idpId string, sp *SSAMLServiceProvider) (*samlutils.Response, error) {
	data, err := idp.onSpInitiatedLogin(ctx, idpId, sp)
	if err != nil {
		return nil, errors.Wrap(err, "idp.onSpInitiatedLogin")
	}
	input := samlutils.SSAMLResponseInput{
		IssuerCertString:            idp.saml.GetCertString(),
		IssuerEntityId:              idp.saml.GetEntityId(),
		RequestID:                   req.ID,
		RequestEntityId:             req.Issuer.Issuer,
		AssertionConsumerServiceURL: sp.GetPostAssertionConsumerServiceUrl(),
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
	data, err := idp.onIdpInitiatedLogin(ctx, sp, input.IdpId, input.RedirectUrl)
	if err != nil {
		return "", errors.Wrap(err, "idp.onIdpInitiatedLogin")
	}
	if len(data.Form) > 0 {
		return data.Form, nil
	}
	respInput := samlutils.SSAMLResponseInput{
		IssuerCertString:            idp.saml.GetCertString(),
		IssuerEntityId:              idp.saml.GetEntityId(),
		RequestID:                   "",
		RequestEntityId:             sp.GetEntityId(),
		AssertionConsumerServiceURL: sp.GetPostAssertionConsumerServiceUrl(),
		SSAMLSpInitiatedLoginData:   data.SSAMLSpInitiatedLoginData,
	}
	resp := samlutils.NewResponse(respInput)
	form, err := idp.samlResponse2Form(ctx, sp.GetPostAssertionConsumerServiceUrl(), &resp, data.RelayState)
	if err != nil {
		return "", errors.Wrap(err, "samlResponse2Form")
	}
	return form, nil
}
