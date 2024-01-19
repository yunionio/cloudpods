// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sender

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/monitor/notifydrivers/feishu"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SFeishuSender struct {
	config map[string]api.SNotifyConfigContent
}

func (self *SFeishuSender) GetSenderType() string {
	return api.FEISHU
}

const ApiSendMessageForFeishuByOpenId = feishu.ApiRobotSendMessage + "?receive_id_type=open_id"

// 发送飞书消息
func (feishuSender *SFeishuSender) Send(ctx context.Context, args api.SendParams) error {
	// 发送通知消息体
	body := map[string]interface{}{
		"open_id":  args.Receivers.Contact,
		"msg_type": "text",
		"content": map[string]interface{}{
			"text": args.Message,
		},
	} // 添加bearer token请求头
	header := http.Header{}
	header.Add("Authorization", fmt.Sprintf("Bearer %s", models.ConfigMap[fmt.Sprintf("%s-%s", api.FEISHU, args.DomainId)].Content.AccessToken))
	rep, err := sendRequest(ctx, ApiSendMessageForFeishuByOpenId, httputils.POST, header, nil, jsonutils.Marshal(body))
	if err == nil {
		return nil
	}
	if rep == nil {
		return errors.Wrap(err, "feishu.sendRequest")
	}
	// 发送通知失败的情况
	code, err := rep.GetString("code")
	if err != nil {
		return err
	}
	switch code {
	case "99991663": //token过期
		err = feishuSender.GetAccessToken(ctx, fmt.Sprintf("%s-%s", api.FEISHU, args.DomainId))
		if err != nil {
			return errors.Wrap(err, "tenant token invalid && getToken err")
		}
		header.Set("Authorization", fmt.Sprintf("Bearer %s", models.ConfigMap[fmt.Sprintf("%s-%s", api.FEISHU, args.DomainId)].Content.AccessToken))
		_, err = sendRequest(ctx, ApiSendMessageForFeishuByOpenId, httputils.POST, header, nil, jsonutils.Marshal(body))
		if err == nil {
			return nil
		}
	case "99991672": // 未开放发送消息通知的权限
		err = errors.Wrap(ErrIncompleteConfig, err.Error())
	}

	return err
}

// 校验appId与appSecret
func (feishuSender *SFeishuSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	rep, err := feishuSender.getAccessToken(config.AppId, config.AppSecret)
	if err == nil {
		return "", nil
	}
	var msg string
	switch rep.Code {
	case 10003:
		msg = "invalid AppId"
	case 10014:
		msg = "invalid AppSecret"
	}
	return msg, err
}

// 根据用户手机号获取用户的open_id
func (feishuSender *SFeishuSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	body := jsonutils.NewDict()
	body.Set("mobiles", jsonutils.NewArray(jsonutils.NewString(mobile)))
	header := http.Header{}
	// 考虑到获取用户id需求较少，可通过直接更新token来避免token失效
	err := feishuSender.GetAccessToken(ctx, domainId)
	if err != nil {
		return "", errors.Wrap(err, "GetAccessToken")
	}

	params := url.Values{}
	params.Set("mobiles", mobile)

	header.Set("Authorization", fmt.Sprintf("Bearer %s", models.ConfigMap[fmt.Sprintf("%s-%s", api.FEISHU, domainId)].Content.AccessToken))
	resp, err := sendRequest(ctx, ApiFetchUserID, httputils.GET, header, params, body)
	if err != nil {
		return "", err
	}
	mobileNotExist, _ := resp.GetArray("data", "mobiles_not_exist")
	if len(mobileNotExist) != 0 {
		return "", errors.Wrapf(errors.ErrNotFound, "no such user whose mobile is %s", mobile)
	}
	list, err := resp.GetArray("data", "mobile_users", mobile)
	if err != nil {
		return "", errors.Wrap(err, "jsonutils.JSONObject.GetArray")
	}
	// len(list) must be positive
	userId, err := list[0].GetString("open_id")
	if err != nil {
		return "", errors.Wrapf(err, "user result:%v", resp)
	}
	return userId, nil
}

func (feishuSender *SFeishuSender) IsPersonal() bool {
	return true
}

func (feishuSender *SFeishuSender) IsRobot() bool {
	return false
}

func (feishuSender *SFeishuSender) IsValid() bool {
	return len(feishuSender.config) > 0
}

func (feishuSender *SFeishuSender) IsPullType() bool {
	return true
}

func (feishuSender *SFeishuSender) IsSystemConfigContactType() bool {
	return true
}

func (feishuSender *SFeishuSender) RegisterConfig(config models.SConfig) {
	models.ConfigMap[fmt.Sprintf("%s-%s", config.Type, config.DomainId)] = config
}

// 获取token
func (feishuSender *SFeishuSender) GetAccessToken(ctx context.Context, domainId string) error {
	key := fmt.Sprintf("%s-%s", api.FEISHU, domainId)
	if _, ok := models.ConfigMap[key]; !ok {
		return errors.Wrapf(errors.ErrNotSupported, "contact-type:%s,domain_id:%s is missing config", api.FEISHU, domainId)
	}
	appId, appSecret := models.ConfigMap[key].Content.AppId, models.ConfigMap[key].Content.AppSecret
	resp, err := feishuSender.getAccessToken(appId, appSecret)
	if err != nil {
		return errors.Wrap(err, "get accessToken")
	}
	models.ConfigMap[key].Content.AccessToken = resp.TenantAccessToken
	return nil
}

func (feishuSender *SFeishuSender) getAccessToken(appId, appSecret string) (*feishu.TenantAccesstokenResp, error) {
	resp, err := feishu.GetTenantAccessTokenInternal(appId, appSecret)
	if err != nil {
		return resp, errors.Wrap(err, "feishu.GetTenantAccessTokenInternal")
	}
	return resp, nil
}

func init() {
	models.Register(&SFeishuSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
