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
	"yunion.io/x/pkg/errors"
)

var (
	ErrIPWhiteList         = errors.Error("Need to add ip to whtelist")
	ErrNoSupportSecSetting = errors.Error("Only the IP address option in the security Settings is supported")
	ErrNoSuchMobile        = errors.Error("No such mobile")
	ErrIncompleteConfig    = errors.Error("Incomplete config")
	ErrDuplicateConfig     = errors.Error("Duplicate config for a domain")
)

const (
	ApiWebhookRobotV2SendMessage = "https://open.feishu.cn/open-apis/bot/v2/hook/"
	// 钉钉发送消息
	ApiDingtalkSendMessage = "https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2?"
	// 钉钉获取token
	ApiDingtalkGetToken = "https://oapi.dingtalk.com/gettoken?"
	// 钉钉使用手机号获取用户ID
	ApiDingtalkGetUserByMobile = "https://oapi.dingtalk.com/topapi/v2/user/getbymobile?"
	// 钉钉获取消息发送结果
	ApiDingtalkGetSendResult = "https://oapi.dingtalk.com/topapi/message/corpconversation/getsendresult?"
	// 企业微信获取token
	ApiWorkwxGetToken = "https://qyapi.weixin.qq.com/cgi-bin/gettoken?"
	// 企业微信使用手机号获取用户ID
	ApiWorkwxGetUserByMobile = "https://qyapi.weixin.qq.com/cgi-bin/user/getuserid?"
	// 企业微信发送消息
	ApiWorkwxSendMessage = "https://qyapi.weixin.qq.com/cgi-bin/message/send?"
	// 飞书使用手机号或邮箱获取用户ID
	ApiFetchUserID = "https://open.feishu.cn/open-apis/user/v1/batch_get_id?"
)

var (
	FAIL_KEY = []string{"失败", "fail ", "failed"}
)

const EVENT_HEADER = "X-Yunion-Event"
