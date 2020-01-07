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

	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	// 获取 tenant_access_token（企业自建应用）
	ApiTenantAccessTokenInternal = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal/"
	// 获取群列表
	ApiChatList = "https://open.feishu.cn/open-apis/chat/v4/list"
	// 机器人发送消息
	ApiRobotSendMessage = "https://open.feishu.cn/open-apis/message/v4/send/"
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
	AccessToken string
}

func BuildTokenHeader(token string) http.Header {
	h := http.Header{}
	h.Add("Authorization", fmt.Sprintf("Bearer "+token))
	return h
}

func NewTenant(appId, appSecret string) (*Tenant, error) {
	resp, err := GetTenantAccessTokenInternal(appId, appSecret)
	if err != nil {
		return nil, err
	}
	return &Tenant{
		AccessToken: resp.TenantAccessToken,
	}, nil
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
	return t.request(httputils.GET, url, query, out)
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
