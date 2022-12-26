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

package dingtalk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
)

type SDingtalkOAuth2Driver struct {
	oauth2.SOAuth2BaseDriver
}

func NewDingtalkOAuth2Driver(appId string, secret string) oauth2.IOAuth2Driver {
	drv := &SDingtalkOAuth2Driver{
		SOAuth2BaseDriver: oauth2.SOAuth2BaseDriver{
			AppId:  appId,
			Secret: secret,
		},
	}
	return drv
}

const (
	AuthUrl = "https://oapi.dingtalk.com/connect/qrconnect"
)

func (drv *SDingtalkOAuth2Driver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	req := map[string]string{
		"appid":         drv.AppId,
		"response_type": "code",
		"scope":         "snsapi_login",
		"state":         state,
		"redirect_uri":  callbackUrl,
	}
	urlStr := fmt.Sprintf("%s?%s", AuthUrl, jsonutils.Marshal(req).QueryString())
	return urlStr, nil
}

const (
	UserInfoUrl    = "https://oapi.dingtalk.com/sns/getuserinfo_bycode"
	AccessTokenUrl = "https://oapi.dingtalk.com/gettoken"
)

func (drv *SDingtalkOAuth2Driver) signUrl(timestamp string) string {
	mac := hmac.New(sha256.New, []byte(drv.Secret))
	mac.Write([]byte(timestamp))
	sigBytes := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(sigBytes)
}

type sFetchUserInfoQuery struct {
	AccessKey string `json:"accessKey"`
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
}

type sFetchUserInfoBody struct {
	TmpAuthCode string `json:"tmp_auth_code"`
}

type sFetchUserInfoData struct {
	Nick    string `json:"nick"`
	Openid  string `json:"openid"`
	Unionid string `json:"unionid"`
}

func (drv *SDingtalkOAuth2Driver) fetchUserInfo(ctx context.Context, code string) (*sFetchUserInfoData, error) {
	httpclient := httputils.GetDefaultClient()
	timestamp := strconv.FormatInt(time.Now().UnixNano()/1000000, 10) // microseconds
	qs := sFetchUserInfoQuery{
		AccessKey: drv.AppId,
		Timestamp: timestamp,
		Signature: drv.signUrl(timestamp),
	}
	body := sFetchUserInfoBody{
		TmpAuthCode: code,
	}
	urlstr := fmt.Sprintf("%s?%s", UserInfoUrl, jsonutils.Marshal(qs).QueryString())
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.POST, urlstr, nil, jsonutils.Marshal(body), true)
	if err != nil {
		return nil, errors.Wrap(err, "request access token")
	}
	data := sFetchUserInfoData{}
	err = resp.Unmarshal(&data, "user_info")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &data, nil
}

func (drv *SDingtalkOAuth2Driver) Authenticate(ctx context.Context, code string) (map[string][]string, error) {
	data, err := drv.fetchUserInfo(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "fetchUserInfo")
	}
	ret := make(map[string][]string)
	ret["name"] = []string{data.Nick}
	ret["user_id"] = []string{data.Unionid}
	return ret, nil
}
