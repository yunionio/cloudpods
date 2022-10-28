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

package feishu

import "yunion.io/x/jsonutils"

type CommonResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (r CommonResp) GetCode() int {
	return r.Code
}

func (r CommonResp) GetMsg() string {
	return r.Msg
}

type CommonResponser interface {
	GetCode() int
	GetMsg() string
}

type TenantAccesstokenResp struct {
	CommonResp
	TenantAccesstoken
}

type TenantAccesstoken struct {
	TenantAccessToken string `json:"tenant_access_token"`
	Expire            int64  `json:"expire"`
	Created           int64
}

func (t TenantAccesstoken) CreatedAt() int64 {
	return t.Created
}

func (t TenantAccesstoken) ExpiresIn() int64 {
	return t.Expire
}

type GroupListResp struct {
	CommonResp
	Data *UserGroupListData `json:"data"`
}

type UserGroupListData struct {
	HasMore   bool        `json:"has_more"`
	PageToken string      `json:"page_token"`
	Groups    []GroupData `json:"groups"`
}

type GroupData struct {
	Avatar      string `json:"avatar"`
	ChatId      string `json:"chat_id"`
	Description string `json:"description"`
	Name        string `json:"name"`
	OwnerOpenId string `json:"owner_open_id"`
	OwnerUserId string `json:"owner_user_id"`
}

type ChatMembersResp struct {
	CommonResp
	Data *ChatGroupData `json:"data"`
}

type ChatGroupData struct {
	ChatId  string       `json:"chat_id"`
	HasMore bool         `json:"has_more"`
	Members []MemberData `json:"members"`
}

type MemberData struct {
	OpenId string `json:"open_id"`
	UserId string `json:"user_id"`
	Name   string `json:"name"`
}

const (
	MsgTypePost        = "post"
	MsgTypeInteractive = "interactive"
)

// 定义参照: https://open.feishu.cn/open-apis/message/v4/send/
type MsgReq struct {
	OpenId      string `json:"open_id,omitempty"`
	UserId      string `json:"user_id,omitempty"`
	Email       string `json:"email,omitempty"`
	ChatId      string `json:"chat_id,omitempty"`
	MsgType     string `json:"msg_type"`
	RootId      string `json:"root_id,omitempty"`
	UpdateMulti bool   `json:"update_multi"`

	Card    *Card       `json:"card,omitempty"`
	Content *MsgContent `json:"content,omitempty"`
}

type BatchMsgReq struct {
	DepartMentIds []string    `json:"department_ids"`
	OpenIds       []string    `json:"open_ids"`
	UserIds       []string    `json:"user_ids"`
	MsgType       string      `json:"msg_type,omitempty"`
	Content       *MsgContent `json:"content,omitempty"`
}

type MsgContent struct {
	Text     string   `json:"text"`
	ImageKey string   `json:"image_key"`
	Post     *MsgPost `json:"post,omitempty"`
}

type MsgPost struct {
	ZhCn *MsgPostValue `json:"zh_cn,omitempty"`
	EnUs *MsgPostValue `json:"en_us,omitempty"`
	JaJp *MsgPostValue `json:"ja_jp,omitempty"`
}

type MsgPostValue struct {
	Title   string      `json:"title"`
	Content interface{} `json:"content"`
}

type MsgPostContentText struct {
	Tag      string `json:"tag"`
	UnEscape bool   `json:"un_escape"`
	Text     string `json:"text"`
}

type MsgPostContentA struct {
	Tag  string `json:"tag"`
	Text string `json:"text"`
	Href string `json:"href"`
}

type MsgPostContentAt struct {
	Tag    string `json:"tag"`
	UserId string `json:"user_id"`
}

type MsgPostContentImage struct {
	Tag      string  `json:"tag"`
	ImageKey string  `json:"image_key"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
}

// 机器人消息Card字段数据格式定义
type Card struct {
	Config       *CardConfig     `json:"config,omitempty"`
	CardLink     *CardElementUrl `json:"card_link,omitempty"`
	Header       *CardHeader     `json:"header,omitempty"`
	I18nElements *I18nElement    `json:"i18n_elements"`
	Elements     []interface{}   `json:"elements"`
}

type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type CardHeader struct {
	Title *CardHeaderTitle `json:"title,omitempty"`
}

type CardHeaderTitle struct {
	Tag     string    `json:"tag"`
	Content string    `json:"content"`
	Lines   int       `json:"lines,omitempty"`
	I18n    *CardI18n `json:"i18n,omitempty"`
}

type CardI18n struct {
	ZhCn string `json:"zh_cn"`
	EnUs string `json:"en_us"`
	JaJp string `json:"ja_jp"`
}

type CardElementUrl struct {
	Url        string `json:"url"`
	AndroidUrl string `json:"android_url"`
	IosUrl     string `json:"ios_url"`
	PcUrl      string `json:"pc_url"`
}

const (
	TagDiv       = "div"
	TagPlainText = "plain_text"
	TagImg       = "img"
	TagNote      = "note"
	TagLarkMd    = "lark_md"
	TagHR        = "hr"
)

type CardElement struct {
	Tag      string              `json:"tag"`
	Content  string              `json:"content"`
	Text     *CardElement        `json:"text"`
	Fields   []*CardElementField `json:"fields"`
	Elements []*CardElement      `json:"elements"`
}

type CardElementField struct {
	IsShort bool         `json:"is_short"`
	Text    *CardElement `json:"text"`
}

func NewCardElementTextField(isShort bool, content string) *CardElementField {
	return &CardElementField{
		IsShort: isShort,
		Text:    &CardElement{Tag: TagLarkMd, Content: content},
	}
}

func NewCardElementHR() *CardElement {
	return &CardElement{Tag: TagHR}
}

func NewCardElementText(content string) *CardElement {
	return &CardElement{Tag: TagPlainText, Content: content}
}

type I18nElement struct {
	ZhCn []interface{} `json:"zh_cn"`
	EnUs []interface{} `json:"en_us"`
	JaJp []interface{} `json:"ja_jp"`
}

type MsgResp struct {
	CommonResp

	Data MsgRespData `json:"data"`
}

type MsgRespData struct {
	MessageId string `json:"message_id"`
}

type BatchMsgResp struct {
	CommonResp

	Data BatchMsgRespData
}

type BatchMsgRespData struct {
	MessageId            string   `json:"message_id"`
	InvalidDepartmentIds string   `json:"invalid_department_ids"`
	InvalidOpenIds       []string `json:"invalid_open_ids"`
	InvalidUserIds       []string `json:"invalid_user_ids"`
}

type UserIDResp struct {
	CommonResp

	Data UserIDRespData `json:"data"`
}

type UserIDRespData struct {
	EmailUsers      jsonutils.JSONObject `json:"email_users"`
	EmailsNotExist  []string             `json:"emails_not_exist"`
	MobileUsers     jsonutils.JSONObject `json:"mobile_users"`
	MobilesNotExist []string             `json:"mobiles_not_exist"`
}

type WebhookRobotMsgReq struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type WebhookRobotMsgResp struct {
	Error string `json:"error"`
	Ok    bool   `json:"ok"`
}
