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
	"strings"

	"github.com/hugozhu/godingtalk"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/notify/models"
)

const (
	WEBHOOK_PREFIX = "https://oapi.dingtalk.com/robot/send?access_token="
)

type SDingTalkRobotSender struct {
	config map[string]api.SNotifyConfigContent
}

func (dingRobotSender *SDingTalkRobotSender) GetSenderType() string {
	return api.DINGTALK_ROBOT
}

func (dingRobotSender *SDingTalkRobotSender) Send(args api.SendParams) error {
	var token string
	var atStr strings.Builder
	title, msg := args.Title, args.Message

	webhook := args.Receivers.Contact
	if strings.HasPrefix(webhook, WEBHOOK_PREFIX) {
		token = webhook[len(WEBHOOK_PREFIX):]
	} else {
		return errors.Wrap(InvalidWebhook, webhook)
	}
	processText := fmt.Sprintf("### %s\n%s%s", title, msg, atStr.String())
	// atList := &godingtalk.RobotAtList{}
	client := godingtalk.NewDingTalkClient("", "")
	rep, err := client.SendRobotMarkdownMessage(token, title, processText)
	if err == nil {
		return nil
	}
	if rep.ErrCode == 310000 {
		if strings.Contains(rep.ErrMsg, "whitelist") {
			return errors.Wrap(ErrIPWhiteList, rep.ErrMsg)
		} else {
			return errors.Wrap(err, jsonutils.Marshal(rep).PrettyString())
		}
	}
	if rep.ErrCode == 300001 && strings.Contains(rep.ErrMsg, "token") {
		return ErrNoSuchWebhook
	}
	return errors.Wrap(err, "this is res err")
}

func (dingRobotSender *SDingTalkRobotSender) ValidateConfig(config api.NotifyConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (dingRobotSender *SDingTalkRobotSender) ContactByMobile(mobile, domainId string) (string, error) {
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

func (dingRobotSender *SDingTalkRobotSender) GetAccessToken(key string) error {
	return nil
}

func init() {
	models.Register(&SDingTalkRobotSender{
		config: map[string]api.SNotifyConfigContent{},
	})
}
