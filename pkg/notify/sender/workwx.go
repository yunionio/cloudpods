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
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SWorkwxSender struct {
	config map[string]api.SNotifyConfigContent
}

func (workwxSender *SWorkwxSender) GetSenderType() string {
	return api.WORKWX
}

func (workwxSender *SWorkwxSender) Send(ctx context.Context, args api.SendParams) error {
	body := map[string]interface{}{
		"agentid": models.ConfigMap[fmt.Sprintf("%s-%s", api.WORKWX, args.DomainId)].Content.AgentId,
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": fmt.Sprintf("# %s\n\n%s", args.Title, args.Message),
		},
		"touser": args.Receivers.Contact,
	}
	respObj, err := workwxSender.sendMessageWithToken(ctx, ApiWorkwxSendMessage, fmt.Sprintf("%s-%s", api.WORKWX, args.DomainId), httputils.POST, nil, nil, jsonutils.Marshal(body))
	if err != nil {
		return errors.Wrap(err, "workwx send message")
	}
	resp := api.SWorkwxSendMessageResp{}
	respObj.Unmarshal(&resp)
	// errcode大于0时返回错误
	if resp.ErrCode > 0 {
		// 对于token过期情况，进行重新获取token，并重新发消息
		if len(resp.UnlicensedUser) > 0 || resp.ErrCode == 42001 {
			err = workwxSender.GetAccessToken(ctx, args.DomainId)
			if err != nil {
				return errors.Wrap(err, "retenant token invalid && getToken err")
			}
			secRespObj, err := workwxSender.sendMessageWithToken(ctx, ApiWorkwxSendMessage, fmt.Sprintf("%s-%s", api.WORKWX, args.DomainId), httputils.POST, nil, nil, jsonutils.Marshal(body))
			secRespObj.Unmarshal(&resp)
			if err == nil && resp.ErrCode == 0 {
				return nil
			} else {
				return errors.Errorf(resp.ErrMsg)
			}
		}
		return errors.Errorf(resp.ErrMsg)
	}
	return nil
}

func (workwxSender *SWorkwxSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	// 校验accesstoken
	_, err := workwxSender.getAccessToken(ctx, config.CorpId, config.Secret)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "40013"):
			return "invalid corpid", nil
		case strings.Contains(err.Error(), "40001"):
			return "invalid corpsecret", nil
		}
		return "", err
	}
	return "", nil
}

func (workwxSender *SWorkwxSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	err := workwxSender.GetAccessToken(ctx, domainId)
	if err != nil {
		return "", err
	}
	body := jsonutils.Marshal(map[string]interface{}{
		"mobile": mobile,
	})
	res, err := workwxSender.sendMessageWithToken(ctx, ApiWorkwxGetUserByMobile, fmt.Sprintf("%s-%s", api.WORKWX, domainId), httputils.POST, nil, nil, jsonutils.Marshal(body))
	if err != nil {
		return "", errors.Wrap(err, "get user by mobile")
	}
	userId, err := res.GetString("userid")
	if err != nil {
		return "", errors.Wrapf(err, "user result:%v", res)
	}
	return userId, nil
}

func (workwxSender *SWorkwxSender) IsPersonal() bool {
	return true
}

func (workwxSender *SWorkwxSender) IsRobot() bool {
	return false
}

func (workwxSender *SWorkwxSender) IsValid() bool {
	return len(workwxSender.config) > 0
}

func (workwxSender *SWorkwxSender) IsPullType() bool {
	return true
}

func (workwxSender *SWorkwxSender) IsSystemConfigContactType() bool {
	return true
}

func (workwxSender *SWorkwxSender) RegisterConfig(config models.SConfig) {
	models.ConfigMap[fmt.Sprintf("%s-%s", config.Type, config.DomainId)] = config
}

func (workwxSender *SWorkwxSender) GetAccessToken(ctx context.Context, domainId string) error {
	key := fmt.Sprintf("%s-%s", api.WORKWX, domainId)
	if _, ok := models.ConfigMap[key]; !ok {
		return errors.Wrapf(errors.ErrNotSupported, "contact-type:%s,domain_id:%s is missing config", api.WORKWX, domainId)
	}
	corpId, secret := models.ConfigMap[key].Content.CorpId, models.ConfigMap[key].Content.Secret
	token, err := workwxSender.getAccessToken(ctx, corpId, secret)
	if err != nil {
		return errors.Wrap(err, "workwx getAccessToken")
	}
	models.ConfigMap[key].Content.AccessToken = token
	return nil
}

func (workwxSender *SWorkwxSender) getAccessToken(ctx context.Context, corpId, secret string) (string, error) {
	params := url.Values{}
	params.Set("corpid", corpId)
	params.Set("corpsecret", secret)
	res, err := sendRequest(ctx, ApiWorkwxGetToken, httputils.GET, nil, params, nil)
	if err != nil {
		return "", errors.Wrap(err, "get workwx token")
	}
	return res.GetString("access_token")
}

func (workwxSender *SWorkwxSender) sendMessageWithToken(ctx context.Context, uri, key string, method httputils.THttpMethod, header http.Header, params url.Values, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if params == nil {
		params = url.Values{}
	}
	if _, ok := models.ConfigMap[key]; !ok {
		return nil, errors.Wrapf(errors.ErrNotSupported, "contact-type:%s,domain_id:%s is missing config", strings.Split(key, "-")[0], strings.Split(key, "-")[1])
	}
	params.Set("access_token", models.ConfigMap[key].Content.AccessToken)
	return sendRequest(ctx, uri, httputils.POST, nil, params, jsonutils.Marshal(body))
}

func init() {
	models.Register(&SWorkwxSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
