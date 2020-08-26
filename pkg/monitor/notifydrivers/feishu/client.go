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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	// 获取 tenant_access_token（企业自建应用）
	ApiTenantAccessTokenInternal = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal/"
	// 获取群列表
	ApiChatList = "https://open.feishu.cn/open-apis/chat/v4/list"
	// 机器人发送消息
	ApiRobotSendMessage = "https://open.feishu.cn/open-apis/message/v4/send/"
	// 批量发送消息
	ApiRobotBatchSendMessage = "https://open.feishu.cn/open-apis/message/v4/batch_send/"
	// 使用手机号或邮箱获取用户ID
	ApiFetchUserID = "https://open.feishu.cn/open-apis/user/v1/batch_get_id"
	// 使用 webhook 机器人发送消息
	ApiWebhookRobotSendMessage = "https://open.feishu.cn/open-apis/bot/hook/"
)

var (
	cli = &http.Client{
		Transport: httputils.GetTransport(true),
	}
	ctx = context.Background()
)

func Request(method httputils.THttpMethod, url string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	_, resp, err := httputils.JSONRequest(cli, ctx, method, url, header, body, false)
	return resp, err
}

func checkErr(resp CommonResponser) error {
	if resp.GetCode() != 0 {
		return errors.Error(fmt.Sprintf("response error, code: %d, msg: %s", resp.GetCode(), resp.GetMsg()))
	}
	return nil
}

func unmarshal(resp jsonutils.JSONObject, obj CommonResponser) error {
	if err := resp.Unmarshal(obj); err != nil {
		return errors.Wrap(err, "unmarshal json")
	}
	return checkErr(obj)
}

// 获取 tenant_access_token（企业自建应用）https://open.feishu.cn/document/ukTMukTMukTM/uIjNz4iM2MjLyYzM
func GetTenantAccessTokenInternal(appId string, appSecret string) (*TenantAccesstokenResp, error) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(appId), "app_id")
	body.Add(jsonutils.NewString(appSecret), "app_secret")
	ret, err := Request(httputils.POST, ApiTenantAccessTokenInternal, http.Header{}, body)
	if err != nil {
		return nil, err
	}
	obj := new(TenantAccesstokenResp)
	err = unmarshal(ret, obj)
	return obj, err
}

type Tenant struct {
	AppId       string
	AppSecret   string
	AccessToken string
	Cache       ICache
}

func BuildTokenHeader(token string) http.Header {
	h := http.Header{}
	h.Add("Authorization", fmt.Sprintf("Bearer "+token))
	return h
}

func NewTenant(appId, appSecret string) (*Tenant, error) {
	t := &Tenant{
		AppId:       appId,
		AppSecret:   appSecret,
		AccessToken: "",
		Cache:       nil,
	}
	t.Cache = NewFileCache(fmt.Sprintf(".%s_auth_file", appId))
	err := t.RefreshAccessToken()
	return t, err
}

// RefreshAccessToken is to get a valid access token
func (t *Tenant) RefreshAccessToken() error {
	var data TenantAccesstoken
	err := t.Cache.Get(&data)
	if err == nil {
		t.AccessToken = data.TenantAccessToken
		return nil
	}

	tokenResp, err := GetTenantAccessTokenInternal(t.AppId, t.AppSecret)
	if err == nil {
		t.AccessToken = tokenResp.TenantAccessToken
		data = tokenResp.TenantAccesstoken
		data.Created = time.Now().Unix()
		err = t.Cache.Set(&data)
	}
	return err
}

func (t *Tenant) request(method httputils.THttpMethod, url string, data jsonutils.JSONObject, out CommonResponser) error {
	obj, err := Request(method, url, BuildTokenHeader(t.AccessToken), data)
	if err != nil {
		return err
	}
	err = unmarshal(obj, out)
	return err
}

func (t *Tenant) get(url string, query jsonutils.JSONObject, out CommonResponser) error {
	if query != nil {
		qs := query.QueryString()
		if len(qs) > 0 {
			url = fmt.Sprintf("%s?%s", url, qs)
		}
	}
	return t.request(httputils.GET, url, nil, out)
}

func (t *Tenant) post(url string, body jsonutils.JSONObject, out CommonResponser) error {
	return t.request(httputils.POST, url, body, out)
}

func (t *Tenant) ChatList(pageSize int, pageToken string) (*GroupListResp, error) {
	query := jsonutils.NewDict()
	if pageSize > 0 {
		query.Add(jsonutils.NewInt(int64(pageSize)), "page_size")
	}
	if pageToken != "" {
		query.Add(jsonutils.NewString(pageToken), "page_token")
	}
	resp := new(GroupListResp)
	err := t.get(ApiChatList, query, resp)
	return resp, err
}

func (t *Tenant) SendMessage(msg MsgReq) (*MsgResp, error) {
	body := jsonutils.Marshal(msg)
	resp := new(MsgResp)
	err := t.post(ApiRobotSendMessage, body, resp)
	return resp, err
}

// UserIdByMobile query the open id of user by mobile number. https://open.feishu.cn/document/ukTMukTMukTM/uUzMyUjL1MjM14SNzITN
func (t *Tenant) UserIdByMobile(mobile string) (string, error) {
	query := jsonutils.NewDict()
	query.Set("mobiles", jsonutils.NewString(mobile))
	resp := new(UserIDResp)
	err := t.get(ApiFetchUserID, query, resp)
	if err != nil {
		return "", err
	}
	if len(resp.Data.MobilesNotExist) != 0 {
		return "", errors.Wrapf(errors.ErrNotFound, "no such user whose mobile is %s", mobile)
	}
	list, err := resp.Data.MobileUsers.GetArray(mobile)
	if err != nil {
		return "", errors.Wrap(err, "jsonutils.JSONObject.GetArray")
	}
	// len(list) must be positive
	return list[0].GetString("open_id")
}

// SendWebhookRobotMessage will send to message to Webhook address.
// Webhook's format: https://open.feishu.cn/open-apis/bot/hook/xxxxxxxxxxxxxxxxxxxxxxxxxxx.
// The hook represents the last part of webhook: 'xxxxxxxxxxxxxxxxxxxxxxxxxxx'.
func SendWebhookRobotMessage(hook string, msg WebhookRobotMsgReq) (*WebhookRobotMsgResp, error) {
	url := ApiWebhookRobotSendMessage + hook
	obj, err := Request(httputils.POST, url, http.Header{}, jsonutils.Marshal(msg))
	if err != nil {
		return nil, err
	}
	resp := new(WebhookRobotMsgResp)
	err = obj.Unmarshal(resp)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal json")
	}
	if !resp.Ok {
		return resp, fmt.Errorf("response error, msg: %s", resp.Error)
	}
	return resp, err
}

// BatchSendMessage batch send messages. Doc: https://open.feishu.cn/document/ukTMukTMukTM/ucDO1EjL3gTNx4yN4UTM
func (t *Tenant) BatchSendMessage(msg BatchMsgReq) (*BatchMsgResp, error) {
	body := jsonutils.Marshal(msg)
	resp := new(BatchMsgResp)
	err := t.post(ApiRobotBatchSendMessage, body, resp)
	return resp, err
}
