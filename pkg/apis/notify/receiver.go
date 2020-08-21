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

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type ReceiverCreateInput struct {
	apis.StatusStandaloneResourceCreateInput
	apis.DomainizedResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	// description: user id in keystone
	// example: adfb720ccdd34c638346ea4fa7a713a8
	UID string `json:"uid"`

	// description: user name in keystone
	// example: hello
	UName string `json:"uname"`

	// description: user email
	// example: example@gmail.com
	Email string `json:"email"`

	// description: user mobile
	// example: 17812345678
	Mobile string `json:"mobile"`

	// description: enabled contact types for user
	// example: {"email", "mobile", "feishu", "dingtalk", "workwx"}
	EnabledContactTypes []string `json:"enabled_contact_types"`
}

type ReceiverDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.DomainizedResourceInfo

	SReceiver

	// description: enabled contact types for user
	// example: eamil, mobile, feishu, dingtalk, workwx
	EnabledContactTypes []string `json:"enabled_contact_types"`

	// description: verified contact types for user
	// example: email, mobile, feishu, dingtalk, workwx
	VerifiedContactTypes []string `json:"verified_contact_types"`
}

type ReceiverListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.DomainizedResourceListInput
	apis.EnabledResourceBaseListInput

	UID string `json:"uid"`

	UName string `json:"uname"`

	EnabledContactType string `json:"enabled_contact_type"`

	VerifiedContactType string `json:"verified_contact_type"`
}

type ReceiverUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput

	// description: user email
	// example: example@gmail.com
	Email string `json:"email"`

	// description: user mobile
	// example: 17812345678
	Mobile string `json:"mobile"`

	// description: enabled contacts for user
	// example: {"email", "mobile", "feishu", "dingtalk", "workwx"}
	EnabledContactTypes []string `json:"enabled_contact_types"`
}

type ReceiverTriggerVerifyInput struct {
	// description: contact type
	// required: true
	// example: email
	// enum: email,mobile
	ContactType string `json:"contact_type"`
}

type ReceiverVerifyInput struct {
	// description: Contact type
	// required: true
	// example: email
	// enum: email,mobile
	ContactType string `json:"contact_type"`
	// description: token user input
	// required: true
	// example: 123456
	Token string `json:"token"`
}
