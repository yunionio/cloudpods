package godingtalk

import (
	"net/url"
	"strconv"
)

//SendAppMessage is 发送企业会话消息
func (c *DingTalkClient) SendAppMessage(agentID string, touser string, msg string) error {
	if agentID == "" {
		agentID = c.AgentID
	}
	var data OAPIResponse
	request := map[string]interface{}{
		"touser":  touser,
		"agentid": agentID,
		"msgtype": "text",
		"text": map[string]interface{}{
			"content": msg,
		},
	}
	err := c.httpRPC("message/send", nil, request, &data)
	return err
}

//SendAppOAMessage is 发送OA消息
func (c *DingTalkClient) SendAppOAMessage(agentID string, touser string, msg OAMessage) error {
	if agentID == "" {
		agentID = c.AgentID
	}
	var data OAPIResponse
	request := map[string]interface{}{
		"touser":  touser,
		"agentid": agentID,
		"msgtype": "oa",
		"oa":      msg,
	}
	err := c.httpRPC("message/send", nil, request, &data)
	return err
}

// ActionCardMessage
func (c *DingTalkClient) SendOverAllActionCardMessage(agentID string, touser string, msg OverAllActionCardMessage) error {
	if agentID == "" {
		agentID = c.AgentID
	}
	var data OAPIResponse
	request := map[string]interface{}{
		"touser":  touser,
		"agentid": agentID,
		"msgtype": "action_card",
		"action_card":      msg,
	}
	err := c.httpRPC("message/send", nil, request, &data)
	return err
}

func (c *DingTalkClient) SendIndependentActionCardMessage(agentID string, touser string, msg IndependentActionCardMessage) error {
	if agentID == "" {
		agentID = c.AgentID
	}
	var data OAPIResponse
	request := map[string]interface{}{
		"touser":  touser,
		"agentid": agentID,
		"msgtype": "action_card",
		"action_card":      msg,
	}
	err := c.httpRPC("message/send", nil, request, &data)
	return err
}

//SendAppLinkMessage is 发送企业会话链接消息
func (c *DingTalkClient) SendAppLinkMessage(agentID, touser string, title, text string, picUrl, url string) error {
	if agentID == "" {
		agentID = c.AgentID
	}
	var data OAPIResponse
	request := map[string]interface{}{
		"touser":  touser,
		"agentid": agentID,
		"msgtype": "link",
		"link": map[string]string{
			"messageUrl": url,
			"picUrl":     picUrl,
			"title":      title,
			"text":       text,
		},
	}
	err := c.httpRPC("message/send", nil, request, &data)
	return err
}

//SendTextMessage is 发送普通文本消息
func (c *DingTalkClient) SendTextMessage(sender string, cid string, msg string) (data MessageResponse, err error) {
	request := map[string]interface{}{
		"chatid":  cid,
		"sender":  sender,
		"msgtype": "text",
		"text": map[string]interface{}{
			"content": msg,
		},
	}
	err = c.httpRPC("chat/send", nil, request, &data)
	return data, err
}

//SendImageMessage is 发送图片消息
func (c *DingTalkClient) SendImageMessage(sender string, cid string, mediaID string) (data MessageResponse, err error) {
	request := map[string]interface{}{
		"chatid":  cid,
		"sender":  sender,
		"msgtype": "image",
		"image": map[string]string{
			"media_id": mediaID,
		},
	}
	err = c.httpRPC("chat/send", nil, request, &data)
	return data, err
}

//SendVoiceMessage is 发送语音消息
func (c *DingTalkClient) SendVoiceMessage(sender string, cid string, mediaID string, duration string) (data MessageResponse, err error) {
	request := map[string]interface{}{
		"chatid":  cid,
		"sender":  sender,
		"msgtype": "voice",
		"voice": map[string]string{
			"media_id": mediaID,
			"duration": duration,
		},
	}
	err = c.httpRPC("chat/send", nil, request, &data)
	return data, err
}

//SendFileMessage is 发送文件消息
func (c *DingTalkClient) SendFileMessage(sender string, cid string, mediaID string) (data MessageResponse, err error) {
	request := map[string]interface{}{
		"chatid":  cid,
		"sender":  sender,
		"msgtype": "file",
		"file": map[string]string{
			"media_id": mediaID,
		},
	}
	err = c.httpRPC("chat/send", nil, request, &data)
	return data, err
}

//SendLinkMessage is 发送链接消息
func (c *DingTalkClient) SendLinkMessage(sender string, cid string, mediaID string, url string, title string, text string) (data MessageResponse, err error) {
	request := map[string]interface{}{
		"chatid":  cid,
		"sender":  sender,
		"msgtype": "link",
		"link": map[string]string{
			"messageUrl": url,
			"picUrl":     mediaID,
			"title":      title,
			"text":       text,
		},
	}
	err = c.httpRPC("chat/send", nil, request, &data)
	return data, err
}


// OverAllActionCardMessage 整体跳转ActionCard
type OverAllActionCardMessage struct {
	Title 		string `json:"title"`
	MarkDown 	string `json:"markdown"`
	SingleTitle string `json:"single_title"`
	SingleUrl 	string `json:"single_url"`
}

// IndependentActionCardMessage 独立跳转ActionCard
type IndependentActionCardMessage struct {
	Title 			string `json:"title"`
	MarkDown 		string `json:"markdown"`
	BtnOrientation 	string `json:"btn_orientation"`
	BtnJsonList 	[]ActionCardMessageBtnList `json:"btn_json_list"`
}

type ActionCardMessageBtnList struct {
	Title   	string `json:"title,omitempty"`
	ActionUrl 	string `json:"action_url,omitempty"`
}

func (m *IndependentActionCardMessage) AppendBtnItem(title string, action_url string) {
	f := ActionCardMessageBtnList{Title: title, ActionUrl: action_url}

	if m.BtnJsonList == nil {
		m.BtnJsonList = []ActionCardMessageBtnList{}
	}

	m.BtnJsonList = append(m.BtnJsonList, f)
}

//OAMessage is the Message for OA
type OAMessage struct {
	URL   string `json:"message_url"`
	PcURL string `json:"pc_message_url"`
	Head  struct {
		BgColor string `json:"bgcolor,omitempty"`
		Text    string `json:"text,omitempty"`
	} `json:"head,omitempty"`
	Body struct {
		Title     string          `json:"title,omitempty"`
		Form      []OAMessageForm `json:"form,omitempty"`
		Rich      OAMessageRich   `json:"rich,omitempty"`
		Content   string          `json:"content,omitempty"`
		Image     string          `json:"image,omitempty"`
		FileCount int             `json:"file_count,omitempty"`
		Author    string          `json:"author,omitempty"`
	} `json:"body,omitempty"`
}

type OAMessageForm struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type OAMessageRich struct {
	Num  string `json:"num,omitempty"`
	Unit string `json:"body,omitempty"`
}

func (m *OAMessage) AppendFormItem(key string, value string) {
	f := OAMessageForm{Key: key, Value: value}

	if m.Body.Form == nil {
		m.Body.Form = []OAMessageForm{}
	}

	m.Body.Form = append(m.Body.Form, f)
}

//SendOAMessage is 发送OA消息
func (c *DingTalkClient) SendOAMessage(sender string, cid string, msg OAMessage) (data MessageResponse, err error) {
	request := map[string]interface{}{
		"chatid":  cid,
		"sender":  sender,
		"msgtype": "oa",
		"oa":      msg,
	}
	err = c.httpRPC("chat/send", nil, request, &data)
	return data, err
}

//GetMessageReadList is 获取已读列表
func (c *DingTalkClient) GetMessageReadList(messageID string, cursor int, size int) (data MessageReadListResponse, err error) {
	params := url.Values{}
	params.Add("messageId", messageID)
	params.Add("cursor", strconv.Itoa(cursor))
	params.Add("size", strconv.Itoa(size))
	err = c.httpRPC("chat/getReadList", params, nil, &data)
	return data, err
}
