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

package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lestrrat/go-jwx/jwk"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/oidcutils"
)

type SOIDCClient struct {
	clientId   string
	secret     string
	timeout    time.Duration
	isDebug    bool
	config     oidcutils.SOIDCConfiguration
	httpclient *http.Client

	keySet *jwk.Set
}

func NewOIDCClient(clientId string, secret string, timeoutSeconds int, isDebug bool) *SOIDCClient {
	cli := SOIDCClient{
		clientId: clientId,
		secret:   secret,
		timeout:  time.Duration(timeoutSeconds) * time.Second,
		isDebug:  isDebug,
	}
	cli.httpclient = httputils.GetClient(true, cli.timeout)
	return &cli
}

const (
	WELL_KNOWN_OIDC_CONFIG_PATH = ".well-known/openid-configuration"
)

func (cli *SOIDCClient) FetchConfiguration(ctx context.Context, endpoint string) error {
	path := httputils.JoinPath(endpoint, WELL_KNOWN_OIDC_CONFIG_PATH)
	_, resp, err := httputils.JSONRequest(cli.httpclient, ctx, httputils.GET, path, nil, nil, cli.isDebug)
	if err != nil {
		return errors.Wrap(err, "fetch well-known oidc configuration")
	}
	err = resp.Unmarshal(&cli.config)
	if err != nil {
		return errors.Wrap(err, "unmarshal oidc configuration")
	}
	return nil
}

func (cli *SOIDCClient) SetConfig(authUrl, tokenUrl, userinfoUrl string, scopes []string) {
	cli.config = oidcutils.SOIDCConfiguration{
		AuthorizationEndpoint: authUrl,
		TokenEndpoint:         tokenUrl,
		UserinfoEndpoint:      userinfoUrl,
		ScopesSupported:       scopes,
	}
}

func (cli *SOIDCClient) GetConfig() oidcutils.SOIDCConfiguration {
	return cli.config
}

func (cli *SOIDCClient) FetchJWKS(ctx context.Context) error {
	if len(cli.config.JwksUri) == 0 {
		return errors.Wrap(httperrors.ErrInvalidStatus, "no valid jwks_uri")
	}
	_, resp, err := httputils.JSONRequest(cli.httpclient, ctx, httputils.GET, cli.config.JwksUri, nil, nil, cli.isDebug)
	if err != nil {
		return errors.Wrap(err, "fetch jwks_uri")
	}
	set, err := jwk.ParseString(resp.String())
	if err != nil {
		return errors.Wrap(err, "parse JWK")
	}
	cli.keySet = set
	return nil
}

func (cli *SOIDCClient) request(ctx context.Context, method httputils.THttpMethod, urlStr string, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	secret := fmt.Sprintf("%s:%s", url.QueryEscape(cli.clientId), url.QueryEscape(cli.secret))
	b64Secret := base64.StdEncoding.EncodeToString([]byte(secret))
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Basic %s", b64Secret))
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("Accept", "application/json")
	reqbody := strings.NewReader(data.QueryString())
	resp, err := httputils.Request(cli.httpclient, ctx, method, urlStr, header, reqbody, cli.isDebug)
	_, body, err := httputils.ParseJSONResponse(data.QueryString(), resp, err, cli.isDebug)
	return body, err
}

func (cli *SOIDCClient) FetchToken(ctx context.Context, code string, redirUri string) (*oidcutils.SOIDCAccessTokenResponse, error) {
	req := oidcutils.SOIDCAccessTokenRequest{
		GrantType:   oidcutils.OIDC_REQUEST_GRANT_TYPE,
		Code:        code,
		RedirectUri: redirUri,
		ClientId:    cli.clientId,
	}
	respJson, err := cli.request(ctx, "POST", cli.config.TokenEndpoint, jsonutils.Marshal(req))
	if err != nil {
		return nil, errors.Wrap(err, "request access token")
	}
	if respJson.Contains("data") && !respJson.Contains("access_token") {
		respJson, _ = respJson.Get("data")
	}
	log.Debugf("AccesToken response: %s", respJson)
	accessTokenResp := oidcutils.SOIDCAccessTokenResponse{}
	err = respJson.Unmarshal(&accessTokenResp)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal access token response")
	}
	/*tokenPayload, err := jws.VerifyWithJWKSet([]byte(accessTokenResp.IdToken), cli.keySet, nil)
	if err != nil {
		return errors.Wrap(err, "jws.VerifyWithJWKSet")
	}
	log.Debugf("verify %s", tokenPayload) */
	return &accessTokenResp, nil
}

func (cli *SOIDCClient) FetchUserInfo(ctx context.Context, accessToken string) (map[string]string, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+accessToken)
	url := cli.config.UserinfoEndpoint
	header, body, err := httputils.JSONRequest(cli.httpclient, ctx, httputils.GET, url, header, nil, cli.isDebug)
	if err != nil {
		return nil, errors.Wrap(err, "request userinfo")
	}
	if body.Contains("data") {
		body, _ = body.Get("data")
	}
	info := make(map[string]string)
	err = body.Unmarshal(&info)
	if err != nil {
		return nil, errors.Wrap(err, "json unmarshal")
	}
	return info, nil
}
