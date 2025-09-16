/*
 * Author Kevin Zhu
 *
 * Direct questions, comments to <ipandtcp@gmail.com>
 */

package godingtalk

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

const (
	topAPIMsgAsyncSendMethod   = "dingtalk.corp.message.corpconversation.asyncsend"
	topAPIMsgGetResultMethod   = "dingtalk.corp.message.corpconversation.getsendresult"
	topAPIMsgGetprogressMethod = "dingtalk.corp.message.corpconversation.getsendprogress"
)

type topAPIMsgSendResponse struct {
	topAPIErrResponse
	OK struct {
		ErrCode int    `json:"ding_open_errcode"`
		ErrMsg  string `json:"error_msg"`
		Success bool   `json:"success"`
		TaskID  int    `json:"task_id"`
	} `json:"result"`
}

// mgType     消息类型：text;iamge;voice;file;link;oa;markdown;action_card
// userList   接收推送的UID 列表
// deptList   接收推送的部门ID列表
// toAll      是否发送给所有用户
// msgContent 消息内容
// If success return task_id, or is error is not nil when errored
func (c *DingTalkClient) TopAPIMsgSend(msgType string, userList []string, deptList []int, toAll bool, msgContent interface{}) (int, error) {
	var resp topAPIMsgSendResponse
	if len(userList) > 20 || len(deptList) > 20 {
		return 0, errors.New("Can't more than 20 users or departments at once")
	}

	mcontent, err := json.Marshal(msgContent)
	if err != nil {
		return 0, err
	}

	toAllStr := "false"
	if toAll {
		toAllStr = "true"
	}

	form := url.Values{
		"method":      {topAPIMsgAsyncSendMethod},
		"agent_id":    {c.AgentID},
		"userid_list": {strings.Join(userList, ",")},
		"to_all_user": {toAllStr},
		"msgtype":     {msgType},
		"msgcontent":  {string(mcontent)},
	}

	if len(deptList) > 0 {
		var deptListStr string
		for _, dept := range deptList {
			deptListStr = strconv.Itoa(dept) + ","
		}
		deptListStr = string([]uint8(deptListStr)[0 : len(deptListStr)-1])
		form.Set("dept_id_list", deptListStr)
	}

	return resp.OK.TaskID, c.topAPIRequest(form, &resp)
}

type TopAPIMsgGetSendResult struct {
	topAPIErrResponse
	OK struct {
		ErrCode    int    `json:"ding_open_errcode"`
		ErrMsg     string `json:"error_msg"`
		Success    bool   `json:"success"`
		SendResult struct {
			InvalidUserIDList   []string `json:"invalid_user_id_list"`
			ForbiddenUserIDList []string `json:"forbidden_user_id_list"`
			FaildedUserIDList   []string `json:"failed_user_id_list"`
			ReadUserIDLIst      []string `json:"read_user_id_list"`
			UnreadUserIDList    []string `json:"unread_user_id_list"`
			InvalidDeptIDList   []int    `json:"invalid_dept_id_list"`
		} `json:"send_result"`
	} `json:"result"`
}

func (c *DingTalkClient) TopAPIMsgGetSendResult(taskID int) (TopAPIMsgGetSendResult, error) {
	var resp TopAPIMsgGetSendResult
	form := url.Values{
		"method":   {topAPIMsgGetResultMethod},
		"agent_id": {c.AgentID},
		"task_id":  {strconv.Itoa(taskID)},
	}
	return resp, c.topAPIRequest(form, &resp)
}

type TopAPIMsgGetSendProgress struct {
	topAPIErrResponse
	OK struct {
		ErrCode  int    `json:"ding_open_errcode"`
		ErrMsg   string `json:"error_msg"`
		Success  bool   `json:"success"`
		Progress struct {
			Percent int `json:"progress_in_percent"`
			Status  int `json:"status"`
		} `json:"progress"`
	} `json:"result"`
}

func (c *DingTalkClient) TopAPIMsgGetSendProgress(taskID int) (TopAPIMsgGetSendProgress, error) {
	var resp TopAPIMsgGetSendProgress
	form := url.Values{
		"method":   {topAPIMsgGetprogressMethod},
		"agent_id": {c.AgentID},
		"task_id":  {strconv.Itoa(taskID)},
	}
	return resp, c.topAPIRequest(form, &resp)
}
