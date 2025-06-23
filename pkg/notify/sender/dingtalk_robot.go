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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type SDingTalkRobotSender struct {
	config map[string]api.SNotifyConfigContent
}

func (dingRobotSender *SDingTalkRobotSender) GetSenderType() string {
	return api.DINGTALK_ROBOT
}

func (dingRobotSender *SDingTalkRobotSender) Send(ctx context.Context, args api.SendParams) error {
	var atStr strings.Builder
	title, msg := args.Title, args.Message

	urlAddr, err := url.Parse(args.Receivers.Contact)
	if err != nil {
		return errors.Wrapf(err, "invalid url %s", args.Receivers.Contact)
	}
	query := urlAddr.Query()
	token := query.Get("access_token")
	if len(token) == 0 {
		return httperrors.NewMissingParameterError("access_token")
	}

	sign := query.Get("sign")
	if len(sign) > 0 {
		timestamp := time.Now().UnixNano() / 1e6
		stringToSign := fmt.Sprintf("%d\n%s", timestamp, sign)

		h := hmac.New(sha256.New, []byte(sign))
		h.Write([]byte(stringToSign))
		signature := h.Sum(nil)

		encodedSignature := base64.StdEncoding.EncodeToString(signature)
		sign = url.QueryEscape(encodedSignature)
		query.Set("timestamp", fmt.Sprintf("%d", timestamp))
		query.Set("sign", sign)
	}

	urlAddr.RawQuery = query.Encode()
	processText := fmt.Sprintf("### %s\n%s%s", title, msg, atStr.String())
	request := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": title,
			"text":  processText,
		},
	}
	_, resp, err := httputils.JSONRequest(nil, ctx, httputils.POST, urlAddr.String(), nil, jsonutils.Marshal(request), true)
	if err != nil {
		return err
	}

	ret := struct {
		Errcode int
		Errmsg  string
	}{}

	err = resp.Unmarshal(&ret)
	if err != nil {
		return errors.Wrapf(err, "Unmarshal")
	}
	if ret.Errcode == 310000 {
		if strings.Contains(ret.Errmsg, "whitelist") {
			return errors.Wrap(ErrIPWhiteList, ret.Errmsg)
		} else {
			return errors.Errorf(resp.String())
		}
	}
	if ret.Errcode == 300001 && strings.Contains(ret.Errmsg, "token") {
		return ErrNoSuchWebhook
	}
	return nil
}

func (dingRobotSender *SDingTalkRobotSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (dingRobotSender *SDingTalkRobotSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (dingRobotSender *SDingTalkRobotSender) IsPersonal() bool {
	return true
}

func (dingRobotSender *SDingTalkRobotSender) IsRobot() bool {
	return true
}

func (dingRobotSender *SDingTalkRobotSender) IsValid() bool {
	return len(dingRobotSender.config) > 0
}

func (dingRobotSender *SDingTalkRobotSender) IsPullType() bool {
	return true
}

func (dingRobotSender *SDingTalkRobotSender) IsSystemConfigContactType() bool {
	return true
}

func (dingRobotSender *SDingTalkRobotSender) GetAccessToken(ctx context.Context, key string) error {
	return nil
}

func (dingRobotSender *SDingTalkRobotSender) RegisterConfig(config models.SConfig) {
}

func init() {
	models.Register(&SDingTalkRobotSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
