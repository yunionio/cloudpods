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

package wechat

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
)

type SWechatOAuth2Driver struct {
	oauth2.SOAuth2BaseDriver
}

func NewWechatOAuth2Driver(appId string, secret string) oauth2.IOAuth2Driver {
	drv := &SWechatOAuth2Driver{
		SOAuth2BaseDriver: oauth2.SOAuth2BaseDriver{
			AppId:  appId,
			Secret: secret,
		},
	}
	return drv
}

const (
	AuthUrl = "https://open.weixin.qq.com/connect/qrconnect"
)

func (drv *SWechatOAuth2Driver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	req := map[string]string{
		"appid":         drv.AppId,
		"redirect_uri":  callbackUrl,
		"response_type": "code",
		"scope":         "snsapi_login",
		"state":         state,
	}
	urlStr := fmt.Sprintf("%s?%s#wechat_redirect", AuthUrl, jsonutils.Marshal(req).QueryString())
	return urlStr, nil
}

const (
	AccessTokenUrl = "https://api.weixin.qq.com/sns/oauth2/access_token"
	UserInfoUrl    = "https://api.weixin.qq.com/sns/userinfo"
)

type sAccessTokenInput struct {
	Appid     string `json:"appid"`
	Secret    string `json:"secret"`
	Code      string `json:"code"`
	GrantType string `json:"grant_type"`
}

type sAccessTokenData struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Openid       string `json:"openid"`
	Scope        string `json:"scope"`
	Unionid      string `json:"unionid"`
}

func (drv *SWechatOAuth2Driver) fetchAccessToken(ctx context.Context, code string) (*sAccessTokenData, error) {
	// ?appid=APPID&secret=SECRET&code=CODE&grant_type=authorization_code
	httpclient := httputils.GetDefaultClient()
	qs := sAccessTokenInput{
		Appid:     drv.AppId,
		Secret:    drv.Secret,
		Code:      code,
		GrantType: "authorization_code",
	}
	urlstr := fmt.Sprintf("%s?%s", AccessTokenUrl, jsonutils.Marshal(qs).QueryString())
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.GET, urlstr, nil, nil, true)
	if err != nil {
		return nil, errors.Wrap(err, "request access token")
	}
	data := sAccessTokenData{}
	err = resp.Unmarshal(&data)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return &data, nil
}

type sUserInfoInput struct {
	AccessToken string `json:"access_token"`
	Openid      string `json:"openid"`
	Lang        string `json:"lang"`
}

type sUserInfoData struct {
	Openid     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Language   string   `json:"language"`
	City       string   `json:"city"`
	Province   string   `json:"province"`
	Country    string   `json:"country"`
	Headimgurl string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	Unionid    string   `json:"unionid"`
}

func fetchUserInfo(ctx context.Context, accessToken, openid string) (*sUserInfoData, error) {
	// https://api.weixin.qq.com/sns/userinfo?access_token=ACCESS_TOKEN&openid=OPENID&lang=zh_CN
	httpclient := httputils.GetDefaultClient()
	qs := sUserInfoInput{
		AccessToken: accessToken,
		Openid:      openid,
		Lang:        "zh_CN",
	}
	urlStr := fmt.Sprintf("%s?%s", UserInfoUrl, jsonutils.Marshal(qs).QueryString())
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.GET, urlStr, nil, nil, true)
	if err != nil {
		return nil, errors.Wrap(err, "request access token")
	}
	data := sUserInfoData{}
	err = resp.Unmarshal(&data)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &data, nil
}

func (drv *SWechatOAuth2Driver) Authenticate(ctx context.Context, code string) (map[string][]string, error) {
	accessData, err := drv.fetchAccessToken(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "fetchAccessToken")
	}
	userInfo, err := fetchUserInfo(ctx, accessData.AccessToken, accessData.Openid)
	if err != nil {
		return nil, errors.Wrap(err, "fetchUserInfo")
	}
	ret := make(map[string][]string)
	ret["name"] = []string{userInfo.Nickname}
	ret["user_id"] = []string{userInfo.Openid}
	return ret, nil
}
