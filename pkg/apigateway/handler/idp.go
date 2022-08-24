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

package handler

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

func getSsoBaseCallbackUrl() string {
	if options.Options.SsoRedirectUrl == "" {
		return httputils.JoinPath(options.Options.ApiServer, "api/v1/auth/ssologin")
	}
	return options.Options.SsoRedirectUrl
}

func getSsoCallbackUrl(ctx context.Context, req *http.Request, idpId string) string {
	baseUrl := getSsoBaseCallbackUrl()
	s := auth.GetAdminSession(ctx, FetchRegion(req))
	input := api.GetIdpSsoCallbackUriInput{
		RedirectUri: baseUrl,
	}
	resp, err := modules.IdentityProviders.GetSpecific(s, idpId, "sso-callback-uri", jsonutils.Marshal(input))
	if err != nil {
		return baseUrl
	}
	ret, err := resp.GetString("redirect_uri")
	if err != nil {
		return baseUrl
	}
	return ret
}

func getSsoAuthCallbackUrl() string {
	if options.Options.SsoAuthCallbackUrl == "" {
		return httputils.JoinPath(options.Options.ApiServer, "auth")
	}
	return options.Options.SsoAuthCallbackUrl
}

func getSsoLinkCallbackUrl() string {
	if options.Options.SsoLinkCallbackUrl == "" {
		return httputils.JoinPath(options.Options.ApiServer, "user")
	}
	return options.Options.SsoLinkCallbackUrl
}

func getSsoUserNotFoundCallbackUrl() string {
	if options.Options.SsoUserNotFoundCallbackUrl == "" {
		return getSsoAuthCallbackUrl()
	}
	return options.Options.SsoUserNotFoundCallbackUrl
}

func (h *AuthHandlers) getIdpSsoRedirectUri(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	expires := time.Now().Add(time.Minute * 5)
	params := appctx.AppContextParams(ctx)
	idpId := params["<idp_id>"]
	query, _ := jsonutils.ParseQueryString(req.URL.RawQuery)
	var linkuser string
	if query != nil && query.Contains("linkuser") {
		t, _, _ := fetchAuthInfo(ctx, req)
		if t == nil {
			httperrors.InvalidCredentialError(ctx, w, "invalid credential")
			return
		}
		linkuser = t.GetUserId()
		// authCookie := authToken.GetAuthCookie(t)
		// saveCookie(w, constants.YUNION_AUTH_COOKIE, authCookie, "", expires, true)
	}

	referer := req.Header.Get(http.CanonicalHeaderKey("referer"))

	if query == nil {
		query = jsonutils.NewDict()
	}
	query.(*jsonutils.JSONDict).Set("idp_nonce", jsonutils.NewString(utils.GenRequestId(4)))
	state := base64.URLEncoding.EncodeToString([]byte(query.String()))
	redirectUri := getSsoCallbackUrl(ctx, req, idpId)
	s := auth.GetAdminSession(ctx, FetchRegion(req))
	input := api.GetIdpSsoRedirectUriInput{
		RedirectUri: redirectUri,
		State:       state,
	}
	resp, err := modules.IdentityProviders.GetSpecific(s, idpId, "sso-redirect-uri", jsonutils.Marshal(input))
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	redirUrl, _ := resp.GetString("uri")
	idpDriver, _ := resp.GetString("driver")
	saveCookie(w, "idp_id", idpId, "", expires, true)
	saveCookie(w, "idp_state", state, "", expires, true)
	saveCookie(w, "idp_driver", idpDriver, "", expires, true)
	saveCookie(w, "idp_referer", referer, "", expires, true)
	saveCookie(w, "idp_link_user", linkuser, "", expires, true)

	appsrv.DisableClientCache(w)
	appsrv.SendRedirect(w, redirUrl)
}

func findExtUserId(input string) string {
	pattern := regexp.MustCompile(`idp.SyncOrCreateDomainAndUser: ([^:]+): UserNotFound`)
	matches := pattern.FindAllStringSubmatch(input, -1)
	log.Debugf("%#v", matches)
	if len(matches) > 0 && len(matches[0]) > 1 {
		return matches[0][1]
	}
	return ""
}

func (h *AuthHandlers) handleIdpInitSsoLogin(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	params := appctx.AppContextParams(ctx)
	idpId := params["<idp_id>"]
	s := auth.GetAdminSession(ctx, FetchRegion(req))
	resp, err := modules.IdentityProviders.Get(s, idpId, nil)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	idpDriver, _ := resp.GetString("driver")
	h.internalSsoLogin(ctx, w, req, idpId, idpDriver)
}

func (h *AuthHandlers) handleSsoLogin(ctx context.Context, w http.ResponseWriter, req *http.Request) {

	h.internalSsoLogin(ctx, w, req, "", "")
}

func (h *AuthHandlers) internalSsoLogin(ctx context.Context, w http.ResponseWriter, req *http.Request, idpId, idpDriver string) {
	idpIdC := getCookie(req, "idp_id")
	idpDriverC := getCookie(req, "idp_driver")
	idpState := getCookie(req, "idp_state")
	idpReferer := getCookie(req, "idp_referer")
	idpLinkUser := getCookie(req, "idp_link_user")

	if len(idpIdC) > 0 {
		idpId = idpIdC
	}
	if len(idpDriverC) > 0 {
		idpDriver = idpDriverC
	}

	for _, k := range []string{"idp_id", "idp_driver", "idp_state", "idp_referer", "idp_link_user"} {
		clearCookie(w, k, "")
	}

	missing := make([]string, 0)
	if len(idpId) == 0 {
		missing = append(missing, "idp_id")
	}
	if len(idpDriver) == 0 {
		missing = append(missing, "idp_driver")
	}
	/*if len(idpState) == 0 {
		missing = append(missing, "idp_state")
	}
	if len(idpReferer) == 0 {
		missing = append(missing, "idp_referer")
	}*/
	if len(missing) > 0 {
		httperrors.TimeoutError(ctx, w, "session expires, missing %s", strings.Join(missing, ","))
		return
	}

	var idpStateQs jsonutils.JSONObject
	if len(idpState) > 0 {
		idpStateQsBytes, _ := base64.URLEncoding.DecodeString(idpState)
		idpStateQs, _ = jsonutils.Parse(idpStateQsBytes)
		log.Debugf("state query sting: %s", idpStateQs)
	}

	var body jsonutils.JSONObject
	var err error
	switch req.Method {
	case "GET":
		body, err = jsonutils.ParseQueryString(req.URL.RawQuery)
		if err != nil {
			httperrors.InputParameterError(ctx, w, "parse query string error: %s", err)
			return
		}
	case "POST":
		formData, err := appsrv.Fetch(req)
		if err != nil {
			httperrors.InputParameterError(ctx, w, "fetch form data error: %s", err)
		}
		body, err = jsonutils.ParseQueryString(string(formData))
		if err != nil {
			httperrors.InputParameterError(ctx, w, "parse form data error: %s", err)
			return
		}
	default:
		httperrors.InputParameterError(ctx, w, "invalid request")
		return
	}

	body.(*jsonutils.JSONDict).Set("idp_id", jsonutils.NewString(idpId))
	body.(*jsonutils.JSONDict).Set("idp_driver", jsonutils.NewString(idpDriver))
	body.(*jsonutils.JSONDict).Set("idp_state", jsonutils.NewString(idpState))

	appsrv.DisableClientCache(w)

	var referer string
	var idpUserId string
	if len(idpLinkUser) > 0 {
		// link with existing user
		err = linkWithExistingUser(ctx, req, idpId, idpLinkUser, body)
		referer = getSsoLinkCallbackUrl()
	} else {
		// ordinary login
		err = h.doLogin(ctx, w, req, body)
		if err != nil {
			if errors.Cause(err) == httperrors.ErrUserNotFound {
				idpUserId = findExtUserId(err.Error())
				if len(idpUserId) == 0 {
					err = httputils.NewJsonClientError(400, string(httperrors.ErrInputParameter), "empty external user id")
				} else {
					referer = getSsoUserNotFoundCallbackUrl()
				}
			}
		}
		if referer == "" {
			referer = getSsoAuthCallbackUrl()
		}
	}
	refererUrl, _ := url.Parse(referer)
	if refererUrl == nil && len(idpReferer) > 0 {
		refererUrl, _ = url.Parse(idpReferer)
	}
	if refererUrl == nil {
		httperrors.InvalidInputError(ctx, w, "empty referer link")
		return
	}
	redirUrl := generateRedirectUrl(refererUrl, idpStateQs, err, idpId, idpUserId)
	appsrv.SendRedirect(w, redirUrl)
}

func generateRedirectUrl(originUrl *url.URL, stateQs jsonutils.JSONObject, err error, idpId, idpUserId string) string {
	var qs jsonutils.JSONObject
	if len(originUrl.RawQuery) > 0 {
		qs, _ = jsonutils.ParseQueryString(originUrl.RawQuery)
	} else {
		qs = jsonutils.NewDict()
	}
	qs.(*jsonutils.JSONDict).Update(stateQs)
	if err != nil {
		var errCls, errDetails string
		switch je := err.(type) {
		case *httputils.JSONClientError:
			errCls = je.Class
			errDetails = je.Details
		default:
			errCls = errors.Cause(err).Error()
			errDetails = err.Error()
		}
		msgLen := 100
		if len(errDetails) > msgLen {
			errDetails = errDetails[:msgLen] + "..."
		}
		qs.(*jsonutils.JSONDict).Add(jsonutils.NewString(errCls), "error_class")
		qs.(*jsonutils.JSONDict).Add(jsonutils.NewString(errDetails), "error_details")
		qs.(*jsonutils.JSONDict).Add(jsonutils.NewString("error"), "result")
		if len(idpUserId) > 0 {
			qs.(*jsonutils.JSONDict).Add(jsonutils.NewString(idpId), "idp_id")
			qs.(*jsonutils.JSONDict).Add(jsonutils.NewString(idpUserId), "idp_entity_id")
		}
	} else {
		qs.(*jsonutils.JSONDict).Add(jsonutils.NewString("success"), "result")
	}
	originUrl.RawQuery = qs.QueryString()
	return originUrl.String()
}

func processSsoLoginData(body jsonutils.JSONObject, cliIp string, redirectUri string) (mcclient.TokenCredential, error) {
	var token mcclient.TokenCredential
	var err error
	idpDriver, _ := body.GetString("idp_driver")
	idpId, _ := body.GetString("idp_id")
	idpState, _ := body.GetString("idp_state")
	switch idpDriver {
	case api.IdentityDriverCAS:
		ticket, _ := body.GetString("ticket")
		if len(ticket) == 0 {
			return nil, httperrors.NewMissingParameterError("ticket")
		}
		token, err = auth.Client().AuthenticateCAS(idpId, ticket, redirectUri, "", "", "", cliIp)
	case api.IdentityDriverSAML:
		samlResp, _ := body.GetString("SAMLResponse")
		relayState, _ := body.GetString("RelayState")
		if relayState != idpState {
			return nil, errors.Wrap(httperrors.ErrInputParameter, "state inconsistent")
		}
		if len(samlResp) == 0 {
			return nil, errors.Wrap(httperrors.ErrMissingParameter, "SAMLResponse")
		}
		token, err = auth.Client().AuthenticateSAML(idpId, samlResp, "", "", "", cliIp)
	case api.IdentityDriverOIDC:
		code, _ := body.GetString("code")
		state, _ := body.GetString("state")
		if state != idpState {
			return nil, errors.Wrap(httperrors.ErrInputParameter, "state inconsistent")
		}
		if len(code) == 0 {
			return nil, errors.Wrap(httperrors.ErrMissingParameter, "code")
		}
		token, err = auth.Client().AuthenticateOIDC(idpId, code, redirectUri, "", "", "", cliIp)
	case api.IdentityDriverOAuth2:
		state, _ := body.GetString("state")
		if state != idpState {
			return nil, errors.Wrap(httperrors.ErrInputParameter, "state inconsistent")
		}
		code, _ := body.GetString("code")
		if len(code) == 0 {
			code, _ = body.GetString("auth_code")
			if len(code) == 0 {
				return nil, errors.Wrap(httperrors.ErrMissingParameter, "code")
			}
		}
		token, err = auth.Client().AuthenticateOAuth2(idpId, code, "", "", "", cliIp)
	default:
		return nil, errors.Wrapf(httperrors.ErrNotSupported, "SSO driver %s not supported", idpDriver)
	}
	return token, err
}

func linkWithExistingUser(ctx context.Context, req *http.Request, idpId, idpLinkUser string, body jsonutils.JSONObject) error {
	t, _, _ := fetchAuthInfo(ctx, req)
	if t == nil {
		return errors.Wrap(httperrors.ErrInvalidCredential, "invalid credential")
	}
	if t.GetUserId() != idpLinkUser {
		return errors.Wrap(httperrors.ErrConflict, "link user id inconsistent with credential")
	}
	cliIp := netutils2.GetHttpRequestIp(req)
	redirectUri := getSsoCallbackUrl(ctx, req, idpId)
	ntoken, err := processSsoLoginData(body, cliIp, redirectUri)
	if err != nil {
		if errors.Cause(err) != httperrors.ErrUserNotFound {
			return errors.Wrap(err, "invalid ssologin result")
		}
		log.Debugf("error: %s", err)
		// not linked, link with user
		// fetch userId
		jsonErr := err.(*httputils.JSONClientError)
		extUserId := findExtUserId(jsonErr.Details)
		if len(extUserId) == 0 {
			return errors.Wrap(httperrors.ErrInputParameter, "empty external user id")
		}
		linkInput := api.UserLinkIdpInput{
			IdpId:       idpId,
			IdpEntityId: extUserId,
		}
		s := auth.GetAdminSession(ctx, FetchRegion(req))
		_, err = modules.UsersV3.PerformAction(s, t.GetUserId(), "link-idp", jsonutils.Marshal(linkInput))
		if err != nil {
			return errors.Wrap(err, "link-idp")
		}
	} else {
		if ntoken.GetUserId() != t.GetUserId() {
			return errors.Wrap(httperrors.ErrConflict, "link user id inconsistent with credential")
		}
	}
	return nil
}

func handleUnlinkIdp(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	t := AppContextToken(ctx)

	body, err := appsrv.FetchJSON(req)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}

	idpId, _ := body.GetString("idp_id")
	idpEntityId, _ := body.GetString("idp_entity_id")

	if len(idpId) == 0 || len(idpEntityId) == 0 {
		httperrors.InputParameterError(ctx, w, "empty idp_id or idp_entity_id")
		return
	}

	s := auth.GetAdminSession(ctx, FetchRegion(req))
	input := api.UserUnlinkIdpInput{
		IdpId:       idpId,
		IdpEntityId: idpEntityId,
	}
	_, err = modules.UsersV3.PerformAction(s, t.GetUserId(), "unlink-idp", jsonutils.Marshal(input))
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.Send(w, "")
}

func fetchIdpBasicConfig(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	params := appctx.AppContextParams(ctx)
	idpId := params["<idp_id>"]
	info, err := getIdpBasicConfig(ctx, req, idpId)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, info)
}

func getIdpBasicConfig(ctx context.Context, req *http.Request, idpId string) (jsonutils.JSONObject, error) {
	s := auth.GetAdminSession(ctx, FetchRegion(req))
	baseUrl := getSsoBaseCallbackUrl()
	input := api.GetIdpSsoCallbackUriInput{
		RedirectUri: baseUrl,
	}
	resp, err := modules.IdentityProviders.GetSpecific(s, idpId, "sso-callback-uri", jsonutils.Marshal(input))
	if err != nil {
		return nil, errors.Wrap(err, "GetSpecific sso-callback-uri")
	}
	redir, _ := resp.GetString("redirect_uri")
	idpDriver, _ := resp.GetString("driver")
	info := jsonutils.NewDict()
	switch idpDriver {
	case api.IdentityDriverSQL:
	case api.IdentityDriverLDAP:
	case api.IdentityDriverCAS:
		info.Add(jsonutils.NewString(redir), "redirect_uri")
	case api.IdentityDriverSAML:
		info.Add(jsonutils.NewString(options.Options.ApiServer), "entity_id")
		info.Add(jsonutils.NewString(redir), "redirect_uri")
	case api.IdentityDriverOIDC:
		info.Add(jsonutils.NewString(redir), "redirect_uri")
	case api.IdentityDriverOAuth2:
		info.Add(jsonutils.NewString(redir), "redirect_uri")
	default:
	}
	return info, nil
}

func fetchIdpSAMLMetadata(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	s := auth.GetAdminSession(ctx, FetchRegion(req))
	params := appctx.AppContextParams(ctx)
	idpId := params["<idp_id>"]
	query := jsonutils.NewDict()
	query.Set("redirect_uri", jsonutils.NewString(getSsoCallbackUrl(ctx, req, idpId)))
	md, err := modules.IdentityProviders.GetSpecific(s, idpId, "saml-metadata", query)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, md)
}
