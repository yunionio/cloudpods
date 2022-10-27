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

package alipayclient

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

const (
	AlipayGatewayUrl = "https://openapi.alipay.com/gateway.do"
	AlipayFormat     = "json"
	AlipayCharset    = "UTF-8"
	// AlipayCharsetGBK = "GBK"
	AlipaySignType = "RSA2"
	AlipayVersion  = "1.0"
)

type SAlipayClient struct {
	url           string
	appId         string
	appPrivateKey *rsa.PrivateKey
	format        string // always json
	charset       string // always utf-8
	alipayPubKey  string
	signType      string // always RSA2
	version       string
	httpClient    *http.Client
	isDebug       bool
}

func NewDefaultAlipayClient(appId string, appPrivateKey string, alipayPubKey string, isDebug bool) (*SAlipayClient, error) {
	return NewAlipayClient(AlipayGatewayUrl, appId, appPrivateKey, AlipayFormat, AlipayCharset, alipayPubKey, AlipaySignType, isDebug)
}

func NewAlipayClient(url string, appId string, appPrivateKey string, format string, charset string, alipayPubKey string, signType string, isDebug bool) (*SAlipayClient, error) {
	privKey, err := seclib2.DecodePrivateKey([]byte(appPrivateKey))
	if err != nil {
		return nil, errors.Wrap(err, "Invalid appPrivateKey")
	}
	httpClient := httputils.GetClient(true, time.Second*15)
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	cli := &SAlipayClient{
		url:           url,
		appId:         appId,
		appPrivateKey: privKey,
		format:        format,
		charset:       charset,
		alipayPubKey:  alipayPubKey,
		signType:      signType,
		version:       AlipayVersion,
		httpClient:    httpClient,
		isDebug:       isDebug,
	}
	return cli, nil
}

type SCommonRequestParameters struct {
	AppId        string `json:"app_id"`
	Method       string `json:"method"`
	Format       string `json:"format"`
	Charset      string `json:"charset"`
	SignType     string `json:"sign_type"`
	Timestamp    string `json:"timestamp"`
	Version      string `json:"version"`
	AppAuthToken string `json:"app_auth_token"`
}

func (c *SAlipayClient) getCommonParams(method string) SCommonRequestParameters {
	return SCommonRequestParameters{
		AppId:     c.appId,
		Method:    method,
		Format:    c.format,
		Charset:   c.charset,
		SignType:  c.signType,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Version:   c.version,
	}
}

func (c *SAlipayClient) execute(ctx context.Context, method string, params map[string]string) (jsonutils.JSONObject, error) {
	req := jsonutils.Marshal(params).(*jsonutils.JSONDict)
	req.Update(jsonutils.Marshal(c.getCommonParams(method)))
	signedReq, err := c.sign(req)
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}
	urlstr := fmt.Sprintf("%s?%s", c.url, signedReq.QueryString())
	_, resp, err := httputils.JSONRequest(c.httpClient, ctx, httputils.GET, urlstr, nil, nil, c.isDebug)
	if err != nil {
		return nil, errors.Wrap(err, "JSONRequest")
	}
	return resp, nil
}

func signedString(cont *jsonutils.JSONDict) string {
	keys := cont.SortedKeys()
	segs := make([]string, len(keys))
	for i, key := range keys {
		val, _ := cont.GetString(key)
		segs[i] = fmt.Sprintf("%s=%s", key, val)
	}
	return strings.Join(segs, "&")
}

func (c *SAlipayClient) sign(request *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	content := request.CopyExcludes("sign")
	s := sha256.New()
	_, err := s.Write([]byte(signedString(content)))
	if err != nil {
		return nil, errors.Wrap(err, "sha256.Write")
	}
	signByte, err := c.appPrivateKey.Sign(rand.Reader, s.Sum(nil), crypto.SHA256)
	if err != nil {
		return nil, errors.Wrap(err, "privateKey.Sign")
	}

	signStr := base64.StdEncoding.EncodeToString(signByte)
	content.Set("sign", jsonutils.NewString(signStr))
	return content, nil
}

type SAlipaySystemOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	AlipayUserId string `json:"alipay_user_id"`
	ExpiresIn    int    `json:"expires_in"`
	ReExpiresIn  int    `json:"re_expires_in"`
	RefreshToken string `json:"refresh_token"`
	UserId       string `json:"user_id"`
}

// {"alipay_system_oauth_token_response":{"access_token":"authusrB9ee9ecc8105e4fc4869b41e8a470dX90","alipay_user_id":"20881023391875409149385062614990","expires_in":1296000,
//
//	"re_expires_in":2592000,"refresh_token":"authusrBe1a9c0bdf8d344e786ee57f2df9d6E90","user_id":"2088002723447908"},
//	"sign":"rXGE/YX12UrmkEae9jw9WD7B2dS13Hs0r+EnqWwKdERGsUiFmP..."}
func (c *SAlipayClient) GetOAuthToken(ctx context.Context, code string) (*SAlipaySystemOAuthTokenResponse, error) {
	resp, err := c.execute(ctx, "alipay.system.oauth.token", map[string]string{
		"code":       code,
		"grant_type": "authorization_code",
	})
	if err != nil {
		return nil, errors.Wrap(err, "Execute")
	}
	tokenResp := SAlipaySystemOAuthTokenResponse{}
	err = resp.Unmarshal(&tokenResp, "alipay_system_oauth_token_response")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal fail")
	}
	return &tokenResp, nil
}

// {"alipay_user_info_share_response":{"code":"10000","msg":"Success","city":"北京市","gender":"m","nick_name":"剑","province":"北京","user_id":"2088002723447908"},
//
//	"sign":"WZ+uBloiHvQYOOxq02aS/Y4MEoZf5+ANBnt1OKQ9Z8hOPmQsw=="}
func (c *SAlipayClient) GetUserInfo(ctx context.Context, authToken string) (map[string]string, error) {
	resp, err := c.execute(ctx, "alipay.user.info.share", map[string]string{
		"auth_token": authToken,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Execute")
	}
	ret := make(map[string]string)
	err = resp.Unmarshal(&ret, "alipay_user_info_share_response")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal map")
	}
	return ret, nil
}
