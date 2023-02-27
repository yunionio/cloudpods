package godingtalk

import (
	"net/url"
)

type RobotAtList struct {
	AtMobiles []string `json:"atMobiles"`
	IsAtAll   bool     `json:"isAtAll"`
}

type RobotOutgoingMessage struct {
	MessageType string `json:"msgtype"`
	Text        struct {
		Content string `json:"content,omitempty"`
	} `json:"text,omitempty"`
	MessageID         string `json:"msgId"`
	CreatedAt         int64  `json:"createAt"`
	ConversationID    string `json:"conversationId"`
	ConversationType  string `json:"conversationType"`
	ConversationTitle string `json:"conversationTitle"`
	SenderID          string `json:"senderId"`
	SenderNick        string `json:"senderNick"`
	SenderCorpID      string `json:"senderCorpId"`
	SenderStaffID     string `json:"senderStaffId"`
	ChatbotUserID     string `json:"chatbotUserId"`
	AtUsers           []struct {
		DingTalkID string `json:"dingtalkId,omitempty"`
		StaffID    string `json:"staffId,omitempty"`
	} `json:"atUsers,omitempty"`
}

//SendRobotTextMessage can send a text message to a group chat
func (c *DingTalkClient) SendRobotTextMessage(accessToken string, msg string) (data MessageResponse, err error) {
	params := url.Values{}
	params.Add("access_token", accessToken)
	request := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content": msg,
		},
	}
	err = c.httpRPC("robot/send", params, request, &data)
	return data, err
}

//SendRobotMarkdownMessage can send a text message to a group chat
func (c *DingTalkClient) SendRobotMarkdownMessage(accessToken string, title string, msg string) (data MessageResponse, err error) {
	params := url.Values{}
	params.Add("access_token", accessToken)
	request := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": title,
			"text":  msg,
		},
	}
	err = c.httpRPC("robot/send", params, request, &data)
	return data, err
}

// SendRobotTextAtMessage can send a text message and at user to a group chat
func (c *DingTalkClient) SendRobotTextAtMessage(accessToken string, msg string, at *RobotAtList) (data OAPIResponse, err error) {
	params := url.Values{}
	params.Add("access_token", accessToken)
	request := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content": msg,
		},
		"at": at,
	}
	err = c.httpRPC("robot/send", params, request, &data)
	return data, err
}
