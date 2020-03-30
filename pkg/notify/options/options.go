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

package options

import (
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type NotifyOption struct {
	common_options.CommonOptions
	common_options.DBOptions

	DingtalkEnabled    bool   `help:"Enable dingtalk"`
	SocketFileDir      string `help:"Socket file directory" default:"/etc/yunion/notify"`
	UpdateInterval     int    `help:"Update send services interval(unit:min)" default:"30"`
	VerifyEmailUrlPath string `help:"url of verify email" json:"verify_email_url_path"`

	// Deprecated
	VerifyEmailUrl string `help:"url of verify email" json:"verify_email_url"`

	ReSendScope int `help:"Resend all messages that have not been sent successfully within ReSendScope seconds" default:"30"`

	InitNotificationScope int `help:"initialize data of notification with in InitNotificationScope hours" default:"100"`
}

var Options NotifyOption

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*NotifyOption)
	newOpts := newO.(*NotifyOption)

	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		return true
	}

	if oldOpts.SocketFileDir != newOpts.SocketFileDir {
		return true
	}

	return false
}
