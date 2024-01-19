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

package sender

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SDingTalkSender struct {
	config map[string]api.SNotifyConfigContent
}

func (dingSender *SDingTalkSender) GetSenderType() string {
	return api.DINGTALK
}

func (dingSender *SDingTalkSender) Send(ctx context.Context, args api.SendParams) error {
	body := map[string]interface{}{
		"agent_id": models.ConfigMap[fmt.Sprintf("%s-%s", api.DINGTALK, args.DomainId)].Content.AgentId,
		"msg": map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]interface{}{
				"title": args.Title,
				"text":  fmt.Sprintf("%s (%s)", args.Message, time.Now().Format("2006-01-02 15:04")),
			},
		},
		"userid_list": args.Receivers.Contact,
	}
	params := url.Values{}
	params.Set("access_token", models.ConfigMap[fmt.Sprintf("%s-%s", api.DINGTALK, args.DomainId)].Content.AccessToken)
	req, err := sendRequest(ctx, ApiDingtalkSendMessage, httputils.POST, nil, params, jsonutils.Marshal(body))
	if err != nil {
		subCode, _ := req.GetString("sub_code")
		switch subCode {
		// token失效或不合法
		case "40014":
			// 尝试重新获取token
			err = dingSender.GetAccessToken(ctx, fmt.Sprintf("%s-%s", api.DINGTALK, args.DomainId))
			if err != nil {
				return errors.Wrap(err, "reset token")
			}
			// 重新发送通知
			params = url.Values{}
			params.Set("access_token", models.ConfigMap[fmt.Sprintf("%s-%s", api.DINGTALK, args.DomainId)].Content.AccessToken)
			req, err = sendRequest(ctx, ApiDingtalkSendMessage, httputils.POST, nil, params, jsonutils.Marshal(body))
			if err != nil {
				return errors.Wrap(err, "dingtalk resend message")
			}
		}
		if err != nil {
			return errors.Wrap(err, "dingtalk send message")
		}
	}

	// 获取消息通知发送结果
	task_id, _ := req.GetString("task_id")
	body = map[string]interface{}{
		"agent_id": models.ConfigMap[fmt.Sprintf("%s-%s", api.DINGTALK, args.DomainId)].Content.AgentId,
		"task_id":  task_id,
	}
	_, err = sendRequest(ctx, ApiDingtalkSendMessage, httputils.POST, nil, params, jsonutils.Marshal(body))
	return err
}

func (dingSender *SDingTalkSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	// 校验accesstoken
	_, err := dingSender.getAccessToken(ctx, config.AppKey, config.AppSecret)
	if err != nil {
		if strings.Contains(err.Error(), "40089") {
			return "invalid AppKey or AppSecret", err
		}
		return "", err
	}
	return "", nil
}

func (dingSender *SDingTalkSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	err := dingSender.GetAccessToken(ctx, domainId)
	if err != nil {
		return "", err
	}
	body := jsonutils.Marshal(map[string]interface{}{
		"mobile": mobile,
	})
	params := url.Values{}
	params.Set("access_token", models.ConfigMap[fmt.Sprintf("%s-%s", api.DINGTALK, domainId)].Content.AccessToken)
	res, err := sendRequest(ctx, ApiDingtalkGetUserByMobile, httputils.POST, nil, params, body)
	if err != nil {
		return "", errors.Wrap(err, "get user by mobile")
	}
	userId, err := res.GetString("result", "userid")
	if err != nil {
		return "", errors.Wrapf(err, "user result:%v", res)
	}
	return userId, err
}

func (dingSender *SDingTalkSender) IsPersonal() bool {
	return true
}

func (dingSender *SDingTalkSender) IsRobot() bool {
	return false
}

func (dingSender *SDingTalkSender) IsValid() bool {
	return len(dingSender.config) > 0
}

func (dingSender *SDingTalkSender) IsPullType() bool {
	return true
}

func (dingSender *SDingTalkSender) IsSystemConfigContactType() bool {
	return true
}

func (dingSender *SDingTalkSender) RegisterConfig(config models.SConfig) {
	models.ConfigMap[fmt.Sprintf("%s-%s", config.Type, config.DomainId)] = config
}

func (dingSender *SDingTalkSender) GetAccessToken(ctx context.Context, domainId string) error {
	key := fmt.Sprintf("%s-%s", api.DINGTALK, domainId)
	if _, ok := models.ConfigMap[key]; !ok {
		return errors.Wrapf(errors.ErrNotSupported, "contact-type:%s,domain_id:%s is missing config", api.DINGTALK, domainId)
	}
	appKey, appSecret := models.ConfigMap[key].Content.AppKey, models.ConfigMap[key].Content.AppSecret
	token, err := dingSender.getAccessToken(ctx, appKey, appSecret)
	if err != nil {
		return errors.Wrap(err, "dingtalk getAccessToken")
	}
	models.ConfigMap[key].Content.AccessToken = token
	return nil
}

func (dingSender *SDingTalkSender) getAccessToken(ctx context.Context, appKey, appSecret string) (string, error) {
	params := url.Values{}
	params.Set("appkey", appKey)
	params.Set("appsecret", appSecret)
	res, err := sendRequest(ctx, ApiDingtalkGetToken, httputils.GET, nil, params, nil)
	if err != nil {
		return "", errors.Wrap(err, "get dingtalk token")
	}
	return res.GetString("access_token")
}

func init() {
	models.Register(&SDingTalkSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
