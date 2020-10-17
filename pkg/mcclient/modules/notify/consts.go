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

package notify

type TNotifyPriority string

type TNotifyChannel string

const (
	NotifyPriorityImportant = TNotifyPriority("important")
	NotifyPriorityCritical  = TNotifyPriority("fatal")
	NotifyPriorityNormal    = TNotifyPriority("normal")

	NotifyByEmail      = TNotifyChannel("email")
	NotifyByMobile     = TNotifyChannel("mobile")
	NotifyByDingTalk   = TNotifyChannel("dingtalk")
	NotifyByWebConsole = TNotifyChannel("webconsole")
	NotifyByFeishu     = TNotifyChannel("feishu")
	NotifyByWorkwx     = TNotifyChannel("workwx")

	NotifyFeishuRobot     = TNotifyChannel("feishu-robot")
	NotifyByDingTalkRobot = TNotifyChannel("dingtalk-robot")
	NotifyByWorkwxRobot   = TNotifyChannel("workwx-robot")
	NotifyByWebhook       = TNotifyChannel("webhook")
)
