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
	"fmt"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	"yunion.io/x/onecloud/pkg/monitor/notifydrivers/feishu"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:    monitor.AlertNotificationTypeFeishu,
		Factory: newFeishuNotifier,
		ValidateCreateData: func(cred mcclient.IIdentityProvider, input monitor.NotificationCreateInput) (monitor.NotificationCreateInput, error) {
			settings := new(monitor.NotificationSettingFeishu)
			if err := input.Settings.Unmarshal(settings); err != nil {
				return input, errors.Wrap(err, "unmarshal setting")
			}
			if settings.AppId == "" {
				return input, httperrors.NewInputParameterError("app_id is empty")
			}
			if settings.AppSecret == "" {
				return input, httperrors.NewInputParameterError("app_secret is empty")
			}
			_, err := feishu.NewTenant(settings.AppId, settings.AppSecret)
			if err != nil {
				return input, httperrors.NewGeneralError(errors.Wrap(err, "test connection"))
			}
			input.Settings = jsonutils.Marshal(settings)
			return input, nil
		},
	})
}

type FeishuNotifier struct {
	NotifierBase
	// Settings *monitor.NotificationSettingFeishu
	Client  *feishu.Tenant
	ChatIds []string
}

func newFeishuNotifier(config alerting.NotificationConfig) (alerting.Notifier, error) {
	settings := new(monitor.NotificationSettingFeishu)
	if err := config.Settings.Unmarshal(settings); err != nil {
		return nil, errors.Wrap(err, "unmarshal setting")
	}
	cli, err := feishu.NewTenant(settings.AppId, settings.AppSecret)
	if err != nil {
		return nil, err
	}
	ret, err := cli.ChatList(0, "")
	if err != nil {
		return nil, err
	}
	chatIds := make([]string, 0)
	for _, obj := range ret.Data.Groups {
		chatIds = append(chatIds, obj.ChatId)
	}
	return &FeishuNotifier{
		NotifierBase: NewNotifierBase(config),
		Client:       cli,
		ChatIds:      chatIds,
	}, nil
}

func (fs *FeishuNotifier) Notify(ctx *alerting.EvalContext, _ jsonutils.JSONObject) error {
	log.Infof("Sending alert notification to feishu")
	errGrp := errgroup.Group{}
	for _, cId := range fs.ChatIds {
		errGrp.Go(func() error {
			msg, err := fs.genCard(ctx, cId)
			if err != nil {
				return err
			}
			if _, err := fs.Client.SendMessage(*msg); err != nil {
				log.Errorf("--feishu send msg error: %s, error: %v", jsonutils.Marshal(msg), err)
				return err
			}
			log.Errorf("--feishu send msg: %s", jsonutils.Marshal(msg))
			return nil
		})
	}
	return errGrp.Wait()
}

func (fs *FeishuNotifier) getCommonInfoMod(config monitor.NotificationTemplateConfig) feishu.CardElement {
	elem := feishu.CardElement{
		Tag: feishu.TagDiv,
		// Text: feishu.NewCardElementText(config.Title),
		Fields: []*feishu.CardElementField{
			feishu.NewCardElementTextField(false, fmt.Sprintf("**时间:** %s", config.StartTime)),
			feishu.NewCardElementTextField(false, fmt.Sprintf("**级别:** %s", config.Level)),
		},
	}
	return elem
}

func (fs *FeishuNotifier) getMetricElem(idx int, m monitor.EvalMatch) *feishu.CardElement {
	var val string
	if m.Value == nil {
		val = "NaN"
	} else {
		val = fmt.Sprintf("%.2f", *m.Value)
	}

	elem := feishu.CardElement{
		Tag: feishu.TagDiv,
		Fields: []*feishu.CardElementField{
			feishu.NewCardElementTextField(false,
				fmt.Sprintf("**指标 %d:** %s", idx, m.Metric)),
			feishu.NewCardElementTextField(false,
				fmt.Sprintf("**当前值:** %s", val)),
			feishu.NewCardElementTextField(true,
				fmt.Sprintf("**触发条件:**\n%s", m.Condition)),
		},
	}
	return &elem
}

func (fs *FeishuNotifier) getMetricTagElem(m monitor.EvalMatch) *feishu.CardElement {
	inElems := make([]*feishu.CardElement, 0)
	for val, key := range m.Tags {
		inElems = append(inElems, feishu.NewCardElementText(fmt.Sprintf("%s: %s", val, key)))
	}
	elem := feishu.CardElement{
		Tag:      feishu.TagNote,
		Elements: inElems,
	}
	return &elem
}

func (fs *FeishuNotifier) getMetricsMod(config monitor.NotificationTemplateConfig) []*feishu.CardElement {
	inElems := make([]*feishu.CardElement, 0)
	for idx, m := range config.Matches {
		hrE := feishu.NewCardElementHR()
		mE := fs.getMetricElem(idx+1, *m)
		mTE := fs.getMetricTagElem(*m)
		inElems = append(inElems, hrE, mE, mTE)
	}
	return inElems
}

func (fs *FeishuNotifier) genCard(ctx *alerting.EvalContext, chatId string) (*feishu.MsgReq, error) {
	config := GetNotifyTemplateConfig(ctx, false, ctx.EvalMatches)
	commonElem := fs.getCommonInfoMod(config)

	msElems := fs.getMetricsMod(config)
	// 消息卡片: https://open.feishu.cn/document/ukTMukTMukTM/uYTNwUjL2UDM14iN1ATN
	msg := &feishu.MsgReq{
		ChatId:  chatId,
		MsgType: feishu.MsgTypeInteractive,
		Card: &feishu.Card{
			Config:   &feishu.CardConfig{WideScreenMode: false},
			CardLink: nil,
			Header: &feishu.CardHeader{
				Title: &feishu.CardHeaderTitle{
					Tag:     feishu.TagPlainText,
					Content: config.Title,
				},
			},
			Elements: []interface{}{
				commonElem,
			},
		},
	}
	for _, elem := range msElems {
		msg.Card.Elements = append(msg.Card.Elements, elem)
	}
	return msg, nil
}

func (fs *FeishuNotifier) genMsg(ctx *alerting.EvalContext, chatId string) (*feishu.MsgReq, error) {
	config := GetNotifyTemplateConfig(ctx, false, ctx.EvalMatches)
	// 富文本: https://open.feishu.cn/document/ukTMukTMukTM/uMDMxEjLzATMx4yMwETM
	return &feishu.MsgReq{
		ChatId:  chatId,
		MsgType: feishu.MsgTypePost,
		Content: &feishu.MsgContent{
			Post: &feishu.MsgPost{
				ZhCn: &feishu.MsgPostValue{
					Title: config.Title,
					Content: []interface{}{
						[]interface{}{
							feishu.MsgPostContentText{
								Tag:      "text",
								UnEscape: true,
								Text:     "first line",
							},
						},
					},
				},
			},
		},
	}, nil
}
