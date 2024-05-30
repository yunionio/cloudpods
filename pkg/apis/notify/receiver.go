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
	"strings"

	"yunion.io/x/onecloud/pkg/apis"
)

type ReceiverCreateInput struct {
	apis.VirtualResourceCreateInput
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

	// force verified if admin create the records
	ForceVerified bool `json:"force_verified"`
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

// 支持接受分机号（仅保留数字）
func (im *SInternationalMobile) AcceptExtMobile() {
	im.Mobile = moveAreaCode(im.Mobile)
	im.Mobile = moveExtStr(im.Mobile)
}

// 对传入的手机号去除地区编号
func moveAreaCode(mobile string) string {
	// 所有地区编号
	allArea := `283|282|281|280|269|268|267|266|265|264|263|262|261|260|259|258|257|256|255|254|253|252|251|250|249|248|247|246|245|244|243|242|241|240|239|238|237|236|235|234|233|232|231|230|229|228|227|226|225|224|223|222|221|220|219|218|217|216|215|214|213|212|211|210|98|95|94|93|92|91|90|86|84|82|81|66|65|64|63|62|61|60|58|57|56|55|54|53|52|51|49|48|47|46|45|44|43|41|40|39|36|34|33|32|31|30|27|20|7|1`
	temp := strings.Split(allArea, "|")
	for _, area := range temp {
		if strings.HasPrefix(mobile, "+"+area) {
			mobile = mobile[1+len(area):]
			break
		}
	}
	return mobile
}

// 对传入的手机号去除分机号后缀
func moveExtStr(mobile string) string {
	// 定义正则表达式，只保留数字字符串，到不为数字时结束
	re := regexp.MustCompile(`^\d+`)
	// 匹配正则表达式并获取原号码
	match := re.FindStringSubmatch(mobile)
	if match != nil {
		mobile = match[0]
	} else {
		mobile = ""
	}
	return mobile
}

func (im SInternationalMobile) String() string {
	if im.AreaCode == "" {
		return im.Mobile
	}
	return fmt.Sprintf("+%s %s", im.AreaCode, im.Mobile)
}

type ReceiverDetails struct {
	apis.VirtualResourceDetails

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
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	UID string `json:"uid"`

	UName string `json:"uname"`

	EnabledContactType string `json:"enabled_contact_type"`

	VerifiedContactType string `json:"verified_contact_type"`

	ProjectDomainFilter bool `json:"project_domain_filter"`
}

type ReceiverUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	// description: user email
	// example: example@gmail.com
	Email string `json:"email"`

	InternationalMobile SInternationalMobile `json:"international_mobile"`

	// description: enabled contacts for user
	// example: {"email", "mobile", "feishu", "dingtalk", "workwx"}
	EnabledContactTypes []string `json:"enabled_contact_types"`

	ForceVerified bool `json:"force_verified"`
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

type ReceiverEnableContactTypeInput struct {
	EnabledContactTypes []string `json:"enabled_contact_types"`
}

type ReceiverGetSubscriptionOptions struct {
	Id           string
	ShowDisabled bool
}

type SRoleContactInput struct {
	RoleIds         []string
	Scope           string
	ProjectDomainId string
	ProjectId       string `json:"project_id"`
}

type SRoleContactOutput struct {
	ContactType []string
}
