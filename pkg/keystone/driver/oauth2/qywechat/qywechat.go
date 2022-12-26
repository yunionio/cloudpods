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

package qywechat

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
)

type SQywxOAuth2Driver struct {
	oauth2.SOAuth2BaseDriver
}

func NewQywxOAuth2Driver(appId string, secret string) oauth2.IOAuth2Driver {
	drv := &SQywxOAuth2Driver{
		SOAuth2BaseDriver: oauth2.SOAuth2BaseDriver{
			AppId:  appId,
			Secret: secret,
		},
	}
	return drv
}

const (
	AuthUrl = "https://open.work.weixin.qq.com/wwopen/sso/qrConnect"
)

func splitAppId(appId string) (corpId, agentId string, err error) {
	slash := strings.LastIndexByte(appId, '/')
	if slash < 0 {
		err = errors.Wrap(httperrors.ErrInputParameter, "invalid qywx appid, in the format of corp_id/agent_id")
		return
	} else {
		corpId = appId[:slash]
		agentId = appId[slash+1:]
		return
	}
}

func (drv *SQywxOAuth2Driver) GetSsoRedirectUri(ctx context.Context, callbackUrl, state string) (string, error) {
	corpId, agentId, err := splitAppId(drv.AppId)
	if err != nil {
		return "", err
	}
	req := map[string]string{
		"appid":        corpId,
		"agentid":      agentId,
		"redirect_uri": callbackUrl,
		"state":        state,
	}
	urlStr := fmt.Sprintf("%s?%s", AuthUrl, jsonutils.Marshal(req).QueryString())
	return urlStr, nil
}

const (
	AccessTokenUrl = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	UserIdUrl      = "https://qyapi.weixin.qq.com/cgi-bin/user/getuserinfo"
	UserInfoUrl    = "https://qyapi.weixin.qq.com/cgi-bin/user/get"
)

type sAccessTokenInput struct {
	Corpid     string `json:"corpid"`
	Corpsecret string `json:"corpsecret"`
}

type SBaseData struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

type sAccessTokenData struct {
	SBaseData
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (drv *SQywxOAuth2Driver) fetchAccessToken(ctx context.Context) (*sAccessTokenData, error) {
	corpId, _, err := splitAppId(drv.AppId)
	if err != nil {
		return nil, err
	}
	// ?corpid=ID&corpsecret=SECRET
	httpclient := httputils.GetDefaultClient()
	qs := sAccessTokenInput{
		Corpid:     corpId,
		Corpsecret: drv.Secret,
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

type sUserIdInput struct {
	AccessToken string `json:"access_token"`
	Code        string `json:"code"`
}

type sUserIdData struct {
	SBaseData
	UserId string `json:"UserId"`
}

func fetchUserId(ctx context.Context, accessToken, code string) (*sUserIdData, error) {
	httpclient := httputils.GetDefaultClient()
	qs := sUserIdInput{
		AccessToken: accessToken,
		Code:        code,
	}
	urlStr := fmt.Sprintf("%s?%s", UserIdUrl, jsonutils.Marshal(qs).QueryString())
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.GET, urlStr, nil, nil, true)
	if err != nil {
		return nil, errors.Wrap(err, "request access token")
	}
	data := sUserIdData{}
	err = resp.Unmarshal(&data)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &data, nil
}

type sUserInfoInput struct {
	AccessToken string `json:"access_token"`
	Userid      string `json:"userid"`
}

type sUserInfoData struct {
	SBaseData
	Userid           string          `json:"userid"`
	Name             string          `json:"name"`
	Department       []int64         `json:"department"`
	Order            []int64         `json:"order"`
	Position         string          `json:"position"`
	Mobile           string          `json:"mobile"`
	Gender           string          `json:"gender"`
	Email            string          `json:"email"`
	IsLeaderInDept   []int64         `json:"is_leader_in_dept"`
	Avatar           string          `json:"avatar"`
	ThumbAvatar      string          `json:"thumb_avatar"`
	Telephone        string          `json:"telephone"`
	Alias            string          `json:"alias"`
	Address          string          `json:"address"`
	OpenUserid       string          `json:"open_userid"`
	MainDepartment   int64           `json:"main_department"`
	Extattr          Extattr         `json:"extattr"`
	Status           int64           `json:"status"`
	QrCode           string          `json:"qr_code"`
	ExternalPosition string          `json:"external_position"`
	ExternalProfile  ExternalProfile `json:"external_profile"`
}

type Extattr struct {
	Attrs []Attr `json:"attrs"`
}

type Attr struct {
	Type        int64        `json:"type"`
	Name        string       `json:"name"`
	Text        *Text        `json:"text,omitempty"`
	Web         *Web         `json:"web,omitempty"`
	Miniprogram *Miniprogram `json:"miniprogram,omitempty"`
}

type Miniprogram struct {
	Appid    string `json:"appid"`
	Pagepath string `json:"pagepath"`
	Title    string `json:"title"`
}

type Text struct {
	Value string `json:"value"`
}

type Web struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type ExternalProfile struct {
	ExternalCorpName string `json:"external_corp_name"`
	ExternalAttr     []Attr `json:"external_attr"`
}

func fetchUserInfo(ctx context.Context, accessToken, userId string) (*sUserInfoData, error) {
	// https://qyapi.weixin.qq.com/cgi-bin/user/get?access_token=ACCESS_TOKEN&userid=USERID
	httpclient := httputils.GetDefaultClient()
	qs := sUserInfoInput{
		AccessToken: accessToken,
		Userid:      userId,
	}
	urlStr := fmt.Sprintf("%s?%s", UserInfoUrl, jsonutils.Marshal(qs).QueryString())
	_, resp, err := httputils.JSONRequest(httpclient, ctx, httputils.GET, urlStr, nil, nil, true)
	if err != nil {
		return nil, errors.Wrap(err, "request user info")
	}
	data := sUserInfoData{}
	err = resp.Unmarshal(&data)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &data, nil
}

func (drv *SQywxOAuth2Driver) Authenticate(ctx context.Context, code string) (map[string][]string, error) {
	accessData, err := drv.fetchAccessToken(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "fetchAccessToken")
	}
	userId, err := fetchUserId(ctx, accessData.AccessToken, code)
	if err != nil {
		return nil, errors.Wrap(err, "fetchUserId")
	}
	userInfo, err := fetchUserInfo(ctx, accessData.AccessToken, userId.UserId)
	if err != nil {
		return nil, errors.Wrap(err, "fetchUserInfo")
	}
	ret := make(map[string][]string)
	ret["name"] = []string{userId.UserId}
	ret["user_id"] = []string{userId.UserId}
	ret["displayname"] = []string{userInfo.Name}
	ret["email"] = []string{userInfo.Email}
	ret["mobile"] = []string{userInfo.Mobile}
	return ret, nil
}
