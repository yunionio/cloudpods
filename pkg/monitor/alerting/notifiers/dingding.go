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

package notifiers

import (
	"encoding/json"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/alerting/notifiers/templates"
)

const (
	defaultDingdingMsgType    = DingdingMsgTypeMarkdown
	DingdingMsgTypeLink       = "link"
	DingdingMsgTypeMarkdown   = "markdown"
	DingdingMsgTypeActionCard = "actionCard"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:    monitor.AlertNotificationTypeDingding,
		Factory: newDingdingNotifier,
		ValidateCreateData: func(cred mcclient.IIdentityProvider, input monitor.NotificationCreateInput) (monitor.NotificationCreateInput, error) {
			settings := new(monitor.NotificationSettingDingding)
			if err := input.Settings.Unmarshal(settings); err != nil {
				return input, errors.Wrap(err, "unmarshal setting")
			}
			if settings.Url == "" {
				return input, httperrors.NewInputParameterError("url is empty")
			}
			if _, err := url.Parse(settings.Url); err != nil {
				return input, httperrors.NewInputParameterError("invalid url: %v", err)
			}
			if settings.MessageType == "" {
				settings.MessageType = defaultDingdingMsgType
			}
			if !utils.IsInStringArray(settings.MessageType, []string{
				DingdingMsgTypeMarkdown,
				DingdingMsgTypeLink,
				DingdingMsgTypeActionCard,
			}) {
				return input, httperrors.NewInputParameterError("unsupport type: %s", settings.MessageType)
			}
			input.Settings = jsonutils.Marshal(settings)
			return input, nil
		},
	})
}

type DingDingNotifier struct {
	NotifierBase
	MsgType string
	Url     string
}

func newDingdingNotifier(config alerting.NotificationConfig) (alerting.Notifier, error) {
	settings := new(monitor.NotificationSettingDingding)
	if err := config.Settings.Unmarshal(settings); err != nil {
		return nil, errors.Wrap(err, "unmarshal setting")
	}
	return &DingDingNotifier{
		NotifierBase: NewNotifierBase(config),
		Url:          settings.Url,
		MsgType:      settings.MessageType,
	}, nil
}

func (dd *DingDingNotifier) Notify(evalCtx *alerting.EvalContext, d jsonutils.JSONObject) error {
	log.Infof("Sending alert notification to dingding")
	errs := []error{}
	if len(evalCtx.EvalMatches) > 0 {
		if err := dd.notify(evalCtx, evalCtx.EvalMatches, false, d); err != nil {
			errs = append(errs, errors.Wrap(err, "notify alerting matches"))
		}
	}
	if evalCtx.HasRecoveredMatches() {
		if err := dd.notify(evalCtx, evalCtx.GetRecoveredMatches(), true, d); err != nil {
			errs = append(errs, errors.Wrap(err, "notify recovered matches"))
		}
	}
	return errors.NewAggregate(errs)
}

func (dd *DingDingNotifier) notify(ctx *alerting.EvalContext, matches []*monitor.EvalMatch, isRecoverd bool, _ jsonutils.JSONObject) error {
	// msgUrl, err := ctx.GetRuleURL()

	body, err := dd.genBody(ctx, matches, isRecoverd)
	if err != nil {
		return err
	}
	input := &monitor.SendWebhookSync{
		Url:  dd.Url,
		Body: string(body),
	}
	return SendWebRequestSync(ctx.Ctx, input)
}

func (dd *DingDingNotifier) genBody(ctx *alerting.EvalContext, matches []*monitor.EvalMatch, isRecoverd bool) ([]byte, error) {
	q := url.Values{
		"pc_slide": {"false"},
		// "url": {messageURL},
	}

	// Use special link to auto open the message url outside of Dingding
	// Refer: https://open-doc.dingtalk.com/docs/doc.htm?treeId=385&articleId=104972&docType=1#s9
	messageURL := "dingtalk://dingtalkclient/page/link?" + q.Encode()

	log.Infof("messageUrl: " + messageURL)

	config := GetNotifyTemplateConfig(ctx, isRecoverd, matches)
	contentConfig := templates.NewTemplateConfig(config)
	content, err := contentConfig.GenerateMarkdown()
	if err != nil {
		return nil, errors.Wrap(err, "build content")
	}

	var bodyMsg map[string]interface{}
	switch dd.MsgType {
	case DingdingMsgTypeMarkdown:
		bodyMsg = map[string]interface{}{
			"msgtype": DingdingMsgTypeMarkdown,
			DingdingMsgTypeMarkdown: map[string]string{
				"title": config.Title,
				"text":  content,
			},
		}
	case DingdingMsgTypeActionCard:
		bodyMsg = map[string]interface{}{
			"msgtype": DingdingMsgTypeActionCard,
			DingdingMsgTypeActionCard: map[string]string{
				"text":  content,
				"title": config.Title,
				// "singleTitle": "More",
				// "singleURL": messageURL,
			},
		}
	case DingdingMsgTypeLink:
		bodyMsg = map[string]interface{}{
			"msgtype": DingdingMsgTypeLink,
			"link": map[string]string{
				"text":  content,
				"title": config.Title,
				// "messageUrl": messageURL,
			},
		}
	}
	return json.Marshal(bodyMsg)
}
