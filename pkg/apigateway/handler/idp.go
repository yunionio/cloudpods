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
	"net/http"
	"net/url"
	"regexp"
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
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

func getSsoCallbackUrl() string {
	if options.Options.SsoRedirectUrl == "" {
		return httputils.JoinPath(options.Options.ApiServer, "api/v1/auth/ssologin")
	}
	return options.Options.SsoRedirectUrl
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

func (h *AuthHandlers) getIdpSsoRedirectUri(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	params := appctx.AppContextParams(ctx)
	idpId := params["<idp_id>"]
	query, _ := jsonutils.ParseQueryString(req.URL.RawQuery)
	var linkuser string
	if query != nil && query.Contains("linkuser") {
		t, _, _ := fetchAuthInfo(ctx, req)
		if t == nil {
			httperrors.InvalidCredentialError(w, "invalid credential")
			return
		}
		linkuser = t.GetUserId()
	}

	referer := req.Header.Get(http.CanonicalHeaderKey("referer"))

	state := utils.GenRequestId(16)
	redirectUri := getSsoCallbackUrl()
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	input := api.GetIdpSsoRedirectUriInput{
		RedirectUri: redirectUri,
		State:       state,
	}
	resp, err := modules.IdentityProviders.GetSpecific(s, idpId, "sso-redirect-uri", jsonutils.Marshal(input))
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	redirUrl, _ := resp.GetString("uri")
	idpDriver, _ := resp.GetString("driver")
	expires := time.Now().Add(time.Minute * 5)
	saveCookie(w, "idp_id", idpId, "", expires, true)
	saveCookie(w, "idp_state", state, "", expires, true)
	saveCookie(w, "idp_driver", idpDriver, "", expires, true)
	saveCookie(w, "idp_referer", referer, "", expires, true)
	saveCookie(w, "idp_link_user", linkuser, "", expires, true)

	appsrv.DisableClientCache(w)
	appsrv.SendRedirect(w, redirUrl)
}

func findExtUserId(input string) string {
	pattern := regexp.MustCompile(`idp.SyncOrCreateDomainAndUser: ([^:]+): UserNotFoundError`)
	matches := pattern.FindAllStringSubmatch(input, -1)
	log.Debugf("%#v", matches)
	if len(matches) > 0 && len(matches[0]) > 1 {
		return matches[0][1]
	}
	return ""
}

func (h *AuthHandlers) handleSsoLogin(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	idpId := getCookie(req, "idp_id")
	idpDriver := getCookie(req, "idp_driver")
	idpState := getCookie(req, "idp_state")
	idpReferer := getCookie(req, "idp_referer")
	idpLinkUser := getCookie(req, "idp_link_user")
	if len(idpId) == 0 || len(idpDriver) == 0 || len(idpState) == 0 || len(idpReferer) == 0 {
		httperrors.TimeoutError(w, "session expires")
		return
	}

	for _, k := range []string{"idp_id", "idp_driver", "idp_state", "idp_referer", "idp_link_user"} {
		clearCookie(w, k, "")
	}

	var body jsonutils.JSONObject
	var err error
	switch req.Method {
	case "GET":
		body, err = jsonutils.ParseQueryString(req.URL.RawQuery)
		if err != nil {
			httperrors.InputParameterError(w, "parse query string error: %s", err)
			return
		}
	case "POST":
		formData, err := appsrv.Fetch(req)
		if err != nil {
			httperrors.InputParameterError(w, "fetch formdata error: %s", err)
		}
		body, err = jsonutils.ParseQueryString(string(formData))
		if err != nil {
			httperrors.InputParameterError(w, "parse form data error: %s", err)
			return
		}
	default:
		httperrors.InputParameterError(w, "invalid request")
		return
	}

	body.(*jsonutils.JSONDict).Set("idp_id", jsonutils.NewString(idpId))
	body.(*jsonutils.JSONDict).Set("idp_driver", jsonutils.NewString(idpDriver))
	body.(*jsonutils.JSONDict).Set("idp_state", jsonutils.NewString(idpState))

	appsrv.DisableClientCache(w)

	if len(idpLinkUser) > 0 {
		// link with existing user
		err := linkWithExistingUser(ctx, req, idpId, idpLinkUser, body)
		referer := getSsoLinkCallbackUrl()
		refererUrl, _ := url.Parse(referer)
		if refererUrl == nil {
			refererUrl, _ = url.Parse(idpReferer)
		}
		if err != nil {
			log.Debugf("error: %s", err)
		}
		redirUrl := generateRedirectUrl(refererUrl, err, "", "")
		// success, do redirect
		appsrv.SendRedirect(w, redirUrl)
	} else {
		// ordinary login
		referer := getSsoAuthCallbackUrl()
		refererUrl, _ := url.Parse(referer)
		if refererUrl == nil {
			refererUrl, _ = url.Parse(idpReferer)
		}
		var idpUserId string
		err = h.doLogin(ctx, w, req, body)
		if err != nil {
			if errors.Cause(err) == httperrors.ErrUserNotFound {
				idpUserId = findExtUserId(err.Error())
				if len(idpUserId) == 0 {
					err = httputils.NewJsonClientError(400, string(httperrors.ErrInputParameter), "empty external user id")
				}
			}
			log.Debugf("error: %s", err)
		}
		redirUrl := generateRedirectUrl(refererUrl, err, idpId, idpUserId)
		appsrv.SendRedirect(w, redirUrl)
	}
}

func generateRedirectUrl(originUrl *url.URL, err error, idpId, idpUserId string) string {
	var qs jsonutils.JSONObject
	if len(originUrl.RawQuery) > 0 {
		qs, _ = jsonutils.ParseQueryString(originUrl.RawQuery)
	} else {
		qs = jsonutils.NewDict()
	}
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
		qs.(*jsonutils.JSONDict).Add(jsonutils.NewString(errCls), "error_class")
		qs.(*jsonutils.JSONDict).Add(jsonutils.NewString(errDetails), "error_details")
		qs.(*jsonutils.JSONDict).Add(jsonutils.NewString("error"), "result")
		if len(idpUserId) > 0 {
			qs.(*jsonutils.JSONDict).Add(jsonutils.NewString(idpId), "idp_id")
			qs.(*jsonutils.JSONDict).Add(jsonutils.NewString(idpUserId), "idp_user_id")
		}
	} else {
		qs.(*jsonutils.JSONDict).Add(jsonutils.NewString("success"), "result")
	}
	originUrl.RawQuery = qs.QueryString()
	return originUrl.String()
}

func processSsoLoginData(body jsonutils.JSONObject, cliIp string) (mcclient.TokenCredential, error) {
	var token mcclient.TokenCredential
	var err error
	idpDriver, _ := body.GetString("idp_driver")
	idpId, _ := body.GetString("idp_id")
	idpState, _ := body.GetString("idp_state")
	switch idpDriver {
	case api.IdentityDriverCAS:
		redirectUri := getSsoCallbackUrl()
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
		redirectUri := getSsoCallbackUrl()
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
	ntoken, err := processSsoLoginData(body, cliIp)
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
		s := auth.GetAdminSession(ctx, FetchRegion(req), "")
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
		httperrors.GeneralServerError(w, err)
		return
	}

	idpId, _ := body.GetString("idp_id")
	idpEntityId, _ := body.GetString("idp_entity_id")

	if len(idpId) == 0 || len(idpEntityId) == 0 {
		httperrors.InputParameterError(w, "empty idp_id or idp_entity_id")
		return
	}

	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	input := api.UserUnlinkIdpInput{
		IdpId:       idpId,
		IdpEntityId: idpEntityId,
	}
	_, err = modules.UsersV3.PerformAction(s, t.GetUserId(), "unlink-idp", jsonutils.Marshal(input))
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.Send(w, "")
}

func fetchIdpBasicConfig(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	params := appctx.AppContextParams(ctx)
	idpId := params["<idp_id>"]
	info, err := getIdpBasicConfig(s, idpId)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, info)
}

func getIdpBasicConfig(s *mcclient.ClientSession, idpId string) (jsonutils.JSONObject, error) {
	idp, err := modules.IdentityProviders.Get(s, idpId, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Fetch")
	}
	info := jsonutils.NewDict()
	idpDriver, _ := idp.GetString("driver")
	switch idpDriver {
	case api.IdentityDriverSQL:
	case api.IdentityDriverLDAP:
	case api.IdentityDriverCAS:
		info.Add(jsonutils.NewString(getSsoCallbackUrl()), "redirect_uri")
	case api.IdentityDriverSAML:
		info.Add(jsonutils.NewString(options.Options.ApiServer), "entity_id")
		info.Add(jsonutils.NewString(getSsoCallbackUrl()), "redirect_uri")
	case api.IdentityDriverOIDC:
		info.Add(jsonutils.NewString(getSsoCallbackUrl()), "redirect_uri")
	case api.IdentityDriverOAuth2:
		info.Add(jsonutils.NewString(getSsoCallbackUrl()), "redirect_uri")
	default:
	}
	return info, nil
}

func fetchIdpSAMLMetadata(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	params := appctx.AppContextParams(ctx)
	idpId := params["<idp_id>"]
	query := jsonutils.NewDict()
	query.Set("redirect_uri", jsonutils.NewString(getSsoCallbackUrl()))
	md, err := modules.IdentityProviders.GetSpecific(s, idpId, "saml-metadata", query)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, md)
}
