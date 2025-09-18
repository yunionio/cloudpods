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
	"yunion.io/x/log"
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
	err := workwxSender.GetAccessToken(ctx, args.DomainId)
	if err != nil {
		return errors.Wrap(err, "GetAccessToken")
	}

	resp, err := workwxSender.sendMessageWithToken(ctx, ApiWorkwxSendMessage, fmt.Sprintf("%s-%s", api.WORKWX, args.DomainId), jsonutils.Marshal(body))
	if err != nil {
		return errors.Wrap(err, "sendMessageWithToken")
	}
	result := api.SWorkwxSendMessageResp{}
	resp.Unmarshal(&result)
	if result.ErrCode == 0 {
		return nil
	}
	return errors.Errorf("%s", resp.String())
}

func (workwxSender *SWorkwxSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	// 校验accesstoken
	_, _, err := workwxSender.getAccessToken(ctx, config.CorpId, config.Secret)
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
	res, err := workwxSender.sendMessageWithToken(ctx, ApiWorkwxGetUserByMobile, fmt.Sprintf("%s-%s", api.WORKWX, domainId), jsonutils.Marshal(body))
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
	if len(models.ConfigMap[key].Content.AccessToken) > 0 && models.ConfigMap[key].Content.AccessTokenExpireTime.After(time.Now()) {
		log.Debugf("workwx access token is valid %s expire time %s", key, models.ConfigMap[key].Content.AccessTokenExpireTime.Format(time.RFC3339))
		return nil
	}
	corpId, secret := models.ConfigMap[key].Content.CorpId, models.ConfigMap[key].Content.Secret
	token, expireTime, err := workwxSender.getAccessToken(ctx, corpId, secret)
	if err != nil {
		return errors.Wrap(err, "workwx getAccessToken")
	}
	models.ConfigMap[key].Content.AccessToken = token
	models.ConfigMap[key].Content.AccessTokenExpireTime = expireTime
	log.Debugf("workwx access token is valid %s expire time %s", key, models.ConfigMap[key].Content.AccessTokenExpireTime.Format(time.RFC3339))
	return nil
}

func (workwxSender *SWorkwxSender) getAccessToken(ctx context.Context, corpId, secret string) (string, time.Time, error) {
	params := url.Values{}
	params.Set("corpid", corpId)
	params.Set("corpsecret", secret)
	res, err := sendRequest(ctx, ApiWorkwxGetToken, httputils.GET, nil, params, nil)
	if err != nil {
		return "", time.Time{}, errors.Wrap(err, "get workwx token")
	}
	info := struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}{}
	err = res.Unmarshal(&info)
	if err != nil {
		return "", time.Time{}, errors.Wrap(err, "get workwx token")
	}
	if len(info.AccessToken) == 0 {
		return "", time.Time{}, errors.Wrapf(errors.ErrNotFound, "get workwx token %s", res.String())
	}
	expireTime := time.Now().Add(time.Duration(info.ExpiresIn) * time.Second)

	return info.AccessToken, expireTime, nil
}

func (workwxSender *SWorkwxSender) sendMessageWithToken(ctx context.Context, uri, key string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := url.Values{}
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
