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

package feishu

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
)

type SFeishuOAuth2Driver struct {
	oauth2.SOAuth2BaseDriver
}

func NewFeishuOAuth2Driver(appId string, secret string) oauth2.IOAuth2Driver {
	drv := &SFeishuOAuth2Driver{
		SOAuth2BaseDriver: oauth2.SOAuth2BaseDriver{
			AppId:  appId,
			Secret: secret,
		},
	}
	return drv
}

const (
	AuthUrl = "https://open.feishu.cn/open-apis/authen/v1/index"
)

func (drv *SFeishuOAuth2Driver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	req := map[string]string{
		"app_id":       drv.AppId,
		"state":        state,
		"redirect_uri": callbackUrl,
	}
	urlStr := fmt.Sprintf("%s?%s", AuthUrl, jsonutils.Marshal(req).QueryString())
	return urlStr, nil
}

const (
	AppAccessTokenUrl = "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal/"
	AccessTokenUrl    = "https://open.feishu.cn/open-apis/authen/v1/access_token"
	UserInfoUrl       = "https://open.feishu.cn/open-apis/authen/v1/user_info"
)

type sAccessTokenInput struct {
	AppAccessToken string `json:"app_access_token"`
	GrantType      string `json:"grant_type"`
	Code           string `json:"code"`
}

type sAccessTokenData struct {
	AccessToken      string `json:"access_token"`
	AvatarURL        string `json:"avatar_url"`
	AvatarThumb      string `json:"avatar_thumb"`
	AvatarMiddle     string `json:"avatar_middle"`
	AvatarBig        string `json:"avatar_big"`
	ExpiresIn        int64  `json:"expires_in"`
	Name             string `json:"name"`
	EnName           string `json:"en_name"`
	OpenID           string `json:"open_id"`
	TenantKey        string `json:"tenant_key"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
}

func fetchAccessToken(ctx context.Context, appAccessToken string, code string) (*sAccessTokenData, error) {
	httpclient := httputils.GetDefaultClient()
	body := sAccessTokenInput{
		AppAccessToken: appAccessToken,
		GrantType:      "authorization_code",
		Code:           code,
	}
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.POST, AccessTokenUrl, nil, jsonutils.Marshal(body), true)
	if err != nil {
		return nil, errors.Wrap(err, "request access token")
	}
	data := sAccessTokenData{}
	err = resp.Unmarshal(&data, "data")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return &data, nil
}

type sUserInfoData struct {
	Name         string `json:"name"`
	AvatarURL    string `json:"avatar_url"`
	AvatarThumb  string `json:"avatar_thumb"`
	AvatarMiddle string `json:"avatar_middle"`
	AvatarBig    string `json:"avatar_big"`
	Email        string `json:"email"`
	UserID       string `json:"user_id"`
	Mobile       string `json:"mobile"`
	Status       int64  `json:"status"`
}

func fetchUserInfo(ctx context.Context, accessToken string) (*sUserInfoData, error) {
	httpclient := httputils.GetDefaultClient()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+accessToken)
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.GET, UserInfoUrl, header, nil, true)
	if err != nil {
		return nil, errors.Wrap(err, "request access token")
	}
	data := sUserInfoData{}
	err = resp.Unmarshal(&data, "data")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &data, nil
}

type sAppAccessTokenInput struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type sAppAccessTokenData struct {
	Code              int64  `json:"code"`
	Msg               string `json:"msg"`
	AppAccessToken    string `json:"app_access_token"`
	Expire            int64  `json:"expire"`
	TenantAccessToken string `json:"tenant_access_token"`
}

// https://open.feishu.cn/document/ukTMukTMukTM/uADN14CM0UjLwQTN
func fetchAppAccessToken(ctx context.Context, appId, appSecret string) (*sAppAccessTokenData, error) {
	httpclient := httputils.GetDefaultClient()
	body := sAppAccessTokenInput{
		AppID:     appId,
		AppSecret: appSecret,
	}
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.POST, AppAccessTokenUrl, nil, jsonutils.Marshal(body), true)
	if err != nil {
		return nil, errors.Wrap(err, "request access token")
	}
	data := sAppAccessTokenData{}
	err = resp.Unmarshal(&data)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return &data, nil
}

func (drv *SFeishuOAuth2Driver) Authenticate(ctx context.Context, code string) (map[string][]string, error) {
	appData, err := fetchAppAccessToken(ctx, drv.AppId, drv.Secret)
	if err != nil {
		return nil, errors.Wrap(err, "fetchAppAccessToken")
	}
	accessData, err := fetchAccessToken(ctx, appData.AppAccessToken, code)
	if err != nil {
		return nil, errors.Wrap(err, "fetchAccessToken")
	}
	userInfo, err := fetchUserInfo(ctx, accessData.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, "fetchUserInfo")
	}
	ret := make(map[string][]string)
	ret["name"] = []string{userInfo.Name}
	ret["user_id"] = []string{userInfo.UserID}
	ret["name_en"] = []string{accessData.EnName}
	ret["email"] = []string{userInfo.Email}
	ret["mobile"] = []string{userInfo.Mobile}
	return ret, nil
}
