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
	"fmt"
	"regexp"

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

	InternationalMobile SInternationalMobile `json:"international_mobile"`

	// description: enabled contact types for user
	// example: {"email", "mobile", "feishu", "dingtalk", "workwx"}
	EnabledContactTypes []string `json:"enabled_contact_types"`
}

type SInternationalMobile struct {
	// description: user mobile
	// example: 17812345678
	Mobile string `json:"mobile"`
	// description: area code of mobile
	// example: 86
	AreaCode string `json:"area_code"`
}

var (
	defaultAreaCode = "86"
	pareser         = regexp.MustCompile(`\+(\d*) (.*)`)
)

func ParseInternationalMobile(mobile string) SInternationalMobile {
	matchs := pareser.FindStringSubmatch(mobile)
	if len(matchs) == 0 {
		return SInternationalMobile{
			AreaCode: defaultAreaCode,
			Mobile:   mobile,
		}
	}
	return SInternationalMobile{
		Mobile:   matchs[2],
		AreaCode: matchs[1],
	}
}

func (im SInternationalMobile) String() string {
	if im.AreaCode == "" {
		return im.Mobile
	}
	return fmt.Sprintf("+%s %s", im.AreaCode, im.Mobile)
}

type ReceiverDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.DomainizedResourceInfo

	SReceiver
	InternationalMobile SInternationalMobile `json:"international_mobile"`

	// description: enabled contact types for user
	// example: eamil, mobile, feishu, dingtalk, workwx
	EnabledContactTypes []string `json:"enabled_contact_types"`

	// description: verified info
	VerifiedInfos []VerifiedInfo `json:"verified_infos"`
}

type VerifiedInfo struct {
	ContactType string
	Verified    bool
	Note        string
}

type ReceiverListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.DomainizedResourceListInput
	apis.EnabledResourceBaseListInput

	UID string `json:"uid"`

	UName string `json:"uname"`

	EnabledContactType string `json:"enabled_contact_type"`

	VerifiedContactType string `json:"verified_contact_type"`

	ProjectDomainFilter bool `json:"project_domain_filter"`
}

type ReceiverUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput

	// description: user email
	// example: example@gmail.com
	Email string `json:"email"`

	InternationalMobile SInternationalMobile `json:"international_mobile"`

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

type ReceiverIntellijGetInput struct {
	// description: user id in keystone
	// required: true
	UserId string `json:"user_id"`
	// description: create if receiver with UserId does not exist
	// required: false
	CreateIfNo *bool `json:"create_if_no"`
	// description: scope
	// required: true
	Scope string `json:"scope"`
}
