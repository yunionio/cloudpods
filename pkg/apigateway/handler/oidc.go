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
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/apigateway/clientman"
	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/oidcutils"
)

const (
	// OIDC code expires in 5 minutes
	OIDC_CODE_EXPIRE_SECONDS = 300
	// OIDC token expires in 2 hours
	OIDC_TOKEN_EXPIRE_SECONDS = 7200
)

func getLoginCallbackParam() string {
	if options.Options.LoginCallbackParam == "" {
		return "rf"
	}
	return options.Options.LoginCallbackParam
}

func addQuery(urlstr string, qs jsonutils.JSONObject) string {
	qsPos := strings.LastIndexByte(urlstr, '?')
	if qsPos < 0 {
		return fmt.Sprintf("%s?%s", urlstr, qs.QueryString())
	}
	oldQs, _ := jsonutils.ParseQueryString(urlstr[qsPos+1:])
	if oldQs != nil {
		oldQs.(*jsonutils.JSONDict).Update(qs)
		return fmt.Sprintf("%s?%s", urlstr[:qsPos], oldQs.QueryString())
	} else {
		return fmt.Sprintf("%s?%s", urlstr[:qsPos], qs.QueryString())
	}
}

func handleOIDCAuth(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	ctx, err := fetchAndSetAuthContext(ctx, w, req)
	if err != nil {
		// not login redirect to login page
		qs := jsonutils.NewDict()
		oUrl := req.URL.String()
		if !strings.HasPrefix(oUrl, "http") {
			oUrl = httputils.JoinPath(options.Options.ApiServer, oUrl)
		}
		qs.Set(getLoginCallbackParam(), jsonutils.NewString(oUrl))
		loginUrl := addQuery(getSsoAuthCallbackUrl(), qs)
		appsrv.SendRedirect(w, loginUrl)
		return
	}
	query, _ := jsonutils.ParseQueryString(req.URL.RawQuery)
	auth, code, err := doOIDCAuth(ctx, req, query)
	if err != nil {
		qs := jsonutils.NewDict()
		qs.Set("error", jsonutils.NewString(errors.Cause(err).Error()))
		qs.Set("error_description", jsonutils.NewString(err.Error()))
		errorUrl := addQuery(auth.RedirectUri, qs)
		appsrv.SendRedirect(w, errorUrl)
		return
	}
	qs := jsonutils.NewDict()
	qs.Set("code", jsonutils.NewString(code))
	qs.Set("state", jsonutils.NewString(auth.State))
	redirUrl := addQuery(auth.RedirectUri, qs)
	appsrv.DisableClientCache(w)
	appsrv.SendRedirect(w, redirUrl)
}

func fetchOIDCCredential(ctx context.Context, req *http.Request, clientId string) (modules.SOpenIDConnectCredential, error) {
	var oidcSecret modules.SOpenIDConnectCredential
	s := auth.GetAdminSession(ctx, FetchRegion(req), "")
	secret, err := modules.Credentials.GetById(s, clientId, nil)
	if err != nil {
		return oidcSecret, errors.Wrap(err, "Request Credential")
	}
	oidcSecret, err = modules.DecodeOIDCSecret(secret)
	if err != nil {
		return oidcSecret, errors.Wrap(err, "DecodeOIDCSecret")
	}
	return oidcSecret, nil
}

func doOIDCAuth(ctx context.Context, req *http.Request, query jsonutils.JSONObject) (oidcutils.SOIDCAuthRequest, string, error) {
	oidcAuth := oidcutils.SOIDCAuthRequest{}
	if query == nil {
		return oidcAuth, "", errors.Wrap(httperrors.ErrInputParameter, "empty query string")
	}
	err := query.Unmarshal(&oidcAuth)
	if err != nil {
		return oidcAuth, "", errors.Wrap(httperrors.ErrInputParameter, "unmarshal request parameter fail")
	}

	if oidcAuth.ResponseType != oidcutils.OIDC_RESPONSE_TYPE_CODE {
		return oidcAuth, "", errors.Wrapf(httperrors.ErrInputParameter, "invalid resposne type %s", oidcAuth.ResponseType)
	}
	oidcSecret, err := fetchOIDCCredential(ctx, req, oidcAuth.ClientId)
	if err != nil {
		return oidcAuth, "", errors.Wrap(err, "fetchOIDCCredential")
	}
	if oidcSecret.RedirectUri != oidcAuth.RedirectUri {
		return oidcAuth, "", errors.Wrap(httperrors.ErrInvalidCredential, "redirect uri not match")
	}

	token := AppContextToken(ctx)

	cliIp := netutils2.GetHttpRequestIp(req)
	codeInfo := newOIDCClientInfo(token, cliIp, FetchRegion(req))
	code := clientman.EncryptString(codeInfo.toBytes())

	return oidcAuth, code, nil
}

func handleOIDCToken(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	resp, err := validateOIDCToken(ctx, req)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(resp))
	return
}

type SOIDCClientInfo struct {
	Timestamp int64
	Ip        netutils.IPV4Addr
	UserId    string
	ProjectId string
	Region    string
}

func (i SOIDCClientInfo) toBytes() []byte {
	enc := make([]byte, 12+1+len(i.UserId)+1+len(i.ProjectId)+len(i.Region))
	binary.LittleEndian.PutUint64(enc, uint64(i.Timestamp))
	binary.LittleEndian.PutUint32(enc[8:], uint32(i.Ip))
	enc[12] = byte(len(i.UserId))
	enc[13] = byte(len(i.ProjectId))
	copy(enc[14:], i.UserId)
	copy(enc[14+len(i.UserId):], i.ProjectId)
	copy(enc[14+len(i.UserId)+len(i.ProjectId):], i.Region)
	return enc
}

func (i SOIDCClientInfo) isExpired() bool {
	if time.Now().UnixNano()-i.Timestamp > OIDC_CODE_EXPIRE_SECONDS*1000000000 {
		return true
	}
	return false
}

func (i SOIDCClientInfo) expiresAt(secs int) time.Time {
	expires := i.Timestamp + int64(secs)*int64(time.Second)
	esecs := expires / int64(time.Second)
	nsecs := expires - esecs*int64(time.Second)
	return time.Unix(esecs, nsecs)
}

func decodeOIDCClientInfo(enc []byte) (SOIDCClientInfo, error) {
	info := SOIDCClientInfo{}
	if len(enc) < 8+4+1 {
		return info, errors.Wrap(httperrors.ErrInvalidCredential, "code byte length must be 12")
	}
	info.Timestamp = int64(binary.LittleEndian.Uint64(enc))
	info.Ip = netutils.IPV4Addr(binary.LittleEndian.Uint32(enc[8:]))
	info.UserId = string(enc[14 : 14+int(enc[12])])
	info.ProjectId = string(enc[14+int(enc[12]) : 14+int(enc[12])+int(enc[13])])
	info.Region = string(enc[14+int(enc[12])+int(enc[13]):])
	return info, nil
}

func newOIDCClientInfo(token mcclient.TokenCredential, ipstr string, region string) SOIDCClientInfo {
	info := SOIDCClientInfo{}
	info.Timestamp = time.Now().UnixNano()
	info.Ip, _ = netutils.NewIPV4Addr(ipstr)
	info.UserId = token.GetUserId()
	info.ProjectId = token.GetProjectId()
	info.Region = region
	return info
}

type SOIDCClientToken struct {
	Info SOIDCClientInfo
}

func (t SOIDCClientToken) encode() string {
	json := jsonutils.NewDict()
	json.Add(jsonutils.NewString(string(t.Info.toBytes())), "info")
	return clientman.EncryptString([]byte(json.String()))
}

func decodeOIDCClientToken(token string) (SOIDCClientToken, error) {
	ret := SOIDCClientToken{}
	tBytes, err := clientman.DecryptString(token)
	if err != nil {
		return ret, errors.Wrap(err, "DecryptString")
	}
	json, err := jsonutils.Parse(tBytes)
	if err != nil {
		return ret, errors.Wrap(err, "json.Parse")
	}
	info, err := json.GetString("info")
	if err != nil {
		return ret, errors.Wrap(err, "getString(info)")
	}
	ret.Info, err = decodeOIDCClientInfo([]byte(info))
	if err != nil {
		return ret, errors.Wrap(err, "decodeOIDCClientInfo")
	}
	return ret, nil
}

func validateOIDCToken(ctx context.Context, req *http.Request) (oidcutils.SOIDCAccessTokenResponse, error) {
	var tokenResp oidcutils.SOIDCAccessTokenResponse
	bodyBytes, err := appsrv.Fetch(req)
	if err != nil {
		return tokenResp, errors.Wrap(err, "Fetch Body")
	}
	log.Debugf("validateOIDCToken body: %s", string(bodyBytes))
	bodyJson, err := jsonutils.ParseQueryString(string(bodyBytes))
	if err != nil {
		return tokenResp, errors.Wrap(err, "Decode body form data")
	}
	authReq := oidcutils.SOIDCAccessTokenRequest{}
	err = bodyJson.Unmarshal(&authReq)
	if err != nil {
		return tokenResp, errors.Wrap(err, "Unmarshal Access Token Request")
	}
	if authReq.GrantType != oidcutils.OIDC_REQUEST_GRANT_TYPE {
		return tokenResp, errors.Wrapf(httperrors.ErrInvalidCredential, "invalid grant type %s", authReq.GrantType)
	}

	codeTimeBytes, err := clientman.DecryptString(authReq.Code)
	if err != nil {
		return tokenResp, errors.Wrapf(httperrors.ErrInvalidCredential, "invalid code %s", authReq.Code)
	}
	codeInfo, err := decodeOIDCClientInfo(codeTimeBytes)
	if err != nil {
		return tokenResp, errors.Wrap(httperrors.ErrInvalidCredential, "fail to decode code")
	}
	if codeInfo.isExpired() {
		return tokenResp, errors.Wrapf(httperrors.ErrInvalidCredential, "code expires")
	}

	authStr := req.Header.Get("Authorization")
	log.Debugf("Authorization: %s", authStr)
	authParts := strings.Split(string(authStr), " ")
	if len(authParts) != 2 {
		return tokenResp, errors.Wrap(httperrors.ErrInvalidCredential, "illegal authorization header")
	}
	if authParts[0] != "Basic" {
		return tokenResp, errors.Wrapf(httperrors.ErrInvalidCredential, "unsupport auth method %s, only Basic supported", authParts)
	}
	authBytes, err := base64.StdEncoding.DecodeString(authParts[1])
	if err != nil {
		return tokenResp, errors.Wrap(err, "Decode Authorization Header")
	}
	log.Debugf("Authorization basic: %s", string(authBytes))
	authParts = strings.Split(string(authBytes), ":")
	if len(authParts) != 2 {
		return tokenResp, errors.Wrap(httperrors.ErrInvalidCredential, "illegal authorization header")
	}
	clientId, _ := url.QueryUnescape(authParts[0])
	clientSecret, _ := url.QueryUnescape(authParts[1])
	log.Debugf("clientId %s clientSecret: %s authReq.ClientId %s", clientId, clientSecret, authReq.ClientId)

	oidcSecret, err := fetchOIDCCredential(ctx, req, clientId)
	if err != nil {
		return tokenResp, errors.Wrap(err, "fetchOIDCCredential")
	}
	if oidcSecret.RedirectUri != authReq.RedirectUri {
		return tokenResp, errors.Wrap(httperrors.ErrInvalidCredential, "redirect uri not match")
	}
	if oidcSecret.Secret != clientSecret {
		return tokenResp, errors.Wrap(httperrors.ErrInvalidCredential, "client secret not match")
	}

	token := SOIDCClientToken{
		Info: codeInfo,
	}

	tokenResp = token2AccessTokenResponse(token, clientId)
	return tokenResp, nil
}

func token2AccessTokenResponse(token SOIDCClientToken, clientId string) oidcutils.SOIDCAccessTokenResponse {
	resp := oidcutils.SOIDCAccessTokenResponse{}
	resp.AccessToken = token.encode()
	resp.TokenType = oidcutils.OIDC_BEARER_TOKEN_TYPE
	resp.IdToken, _ = token2IdToken(token, clientId)
	resp.ExpiresIn = int(token.Info.expiresAt(OIDC_TOKEN_EXPIRE_SECONDS).Unix() - time.Now().Unix())
	return resp
}

func token2IdToken(token SOIDCClientToken, clientId string) (string, error) {
	jwtToken := jwt.New()
	jwtToken.Set(jwt.IssuerKey, options.Options.ApiServer)
	jwtToken.Set(jwt.SubjectKey, token.Info.UserId)
	jwtToken.Set(jwt.AudienceKey, clientId)
	jwtToken.Set(jwt.ExpirationKey, token.Info.expiresAt(OIDC_TOKEN_EXPIRE_SECONDS).Unix())
	jwtToken.Set(jwt.IssuedAtKey, time.Now().Unix())
	return clientman.SignJWT(jwtToken)
}

func handleOIDCConfiguration(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	authUrl := httputils.JoinPath(options.Options.ApiServer, "api/v1/auth/oidc/auth")
	tokenUrl := httputils.JoinPath(options.Options.ApiServer, "api/v1/auth/oidc/token")
	userinfoUrl := httputils.JoinPath(options.Options.ApiServer, "api/v1/auth/oidc/user")
	jwksUrl := httputils.JoinPath(options.Options.ApiServer, "api/v1/auth/oidc/keys")
	conf := oidcutils.SOIDCConfiguration{
		Issuer:                httputils.JoinPath(options.Options.ApiServer, "api/v1/auth/oidc"),
		AuthorizationEndpoint: authUrl,
		TokenEndpoint:         tokenUrl,
		UserinfoEndpoint:      userinfoUrl,
		JwksUri:               jwksUrl,
		ResponseTypesSupported: []string{
			oidcutils.OIDC_RESPONSE_TYPE_CODE,
		},
		SubjectTypesSupported: []string{
			"public",
		},
		IdTokenSigningAlgValuesSupported: []string{
			string(jwa.RS256),
		},
		ScopesSupported: []string{
			"user",
			"profile",
		},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_basic",
		},
		ClaimsSupported: []string{
			jwt.IssuerKey,
			jwt.SubjectKey,
			jwt.AudienceKey,
			jwt.ExpirationKey,
			jwt.IssuedAtKey,
		},
	}
	appsrv.SendJSON(w, jsonutils.Marshal(conf))
}

func handleOIDCJWKeys(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	keyJson, err := clientman.GetJWKs(ctx)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, keyJson)
}

func handleOIDCUserInfo(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	tokenHdr := getAuthToken(req)
	if len(tokenHdr) == 0 {
		httperrors.InvalidCredentialError(ctx, w, "No token in header")
		return
	}
	token, err := decodeOIDCClientToken(tokenHdr)
	if err != nil {
		log.Errorf("decodeOIDCClientToken %s fail %s", tokenHdr, err)
		httperrors.InvalidCredentialError(ctx, w, "Token in header invalid")
		return
	}
	if token.Info.expiresAt(OIDC_TOKEN_EXPIRE_SECONDS).Before(time.Now()) {
		httperrors.InvalidCredentialError(ctx, w, "Token expired")
		return
	}

	s := auth.GetAdminSession(ctx, token.Info.Region, "")
	data, err := getUserInfo2(s, token.Info.UserId, token.Info.ProjectId, token.Info.Ip.String())
	if err != nil {
		httperrors.NotFoundError(ctx, w, "%v", err)
		return
	}
	appsrv.SendJSON(w, data)
}
