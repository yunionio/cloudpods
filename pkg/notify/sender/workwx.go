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

func (workwxSender *SWorkwxSender) Send(args api.SendParams) error {
	body := map[string]interface{}{
		"agentid": models.ConfigMap[api.WORKWX].Content.AgentId,
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": fmt.Sprintf("# %s\n\n%s", args.Title, args.Message),
		},
		"touser": args.Receivers.Contact,
	}
	_, err := workwxSender.sendMessageWithToken(ApiWorkwxSendMessage, httputils.POST, nil, nil, jsonutils.Marshal(body))
	if err != nil {
		return errors.Wrap(err, "workwx send message")
	}

	return err
}

func (workwxSender *SWorkwxSender) ValidateConfig(config api.NotifyConfig) (string, error) {
	// 校验accesstoken
	_, err := workwxSender.getAccessToken(config.CorpId, config.Secret)
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

func (workwxSender *SWorkwxSender) ContactByMobile(mobile, domainId string) (string, error) {
	err := workwxSender.GetAccessToken()
	if err != nil {
		return "", err
	}
	body := jsonutils.Marshal(map[string]interface{}{
		"mobile": mobile,
	})
	res, err := workwxSender.sendMessageWithToken(ApiWorkwxGetUserByMobile, httputils.POST, nil, nil, jsonutils.Marshal(body))
	if err != nil {
		return "", errors.Wrap(err, "get user by mobile")
	}
	return res.GetString("userid")
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

func (workwxSender *SWorkwxSender) GetAccessToken() error {
	corpId, secret := models.ConfigMap[api.WORKWX].Content.CorpId, models.ConfigMap[api.WORKWX].Content.Secret
	token, err := workwxSender.getAccessToken(corpId, secret)
	if err != nil {
		return errors.Wrap(err, "workwx getAccessToken")
	}
	models.ConfigMap[api.WORKWX].Content.AccessToken = token
	return nil
}

func (workwxSender *SWorkwxSender) getAccessToken(corpId, secret string) (string, error) {
	// url := ApiWorkwxGetToken + fmt.Sprintf("?corpid=%s&corpsecret=%s", corpId, secret)
	params := url.Values{}
	params.Set("corpid", corpId)
	params.Set("corpsecret", secret)
	res, err := sendRequest(ApiWorkwxGetToken, httputils.GET, nil, params, nil)
	if err != nil {
		return "", errors.Wrap(err, "get workwx token")
	}
	return res.GetString("access_token")
}

func (workwxSender *SWorkwxSender) sendMessageWithToken(uri string, method httputils.THttpMethod, header http.Header, params url.Values, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", models.ConfigMap[api.WORKWX].Content.AccessToken)
	return sendRequest(uri, httputils.POST, nil, params, jsonutils.Marshal(body))
}

func init() {
	models.Register(&SWorkwxSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
