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
	"time"
)

const (
	topAPICreateProcInstMethod = "dingtalk.smartwork.bpms.processinstance.create"
	topAPIGetProcInstMethod    = "dingtalk.smartwork.bpms.processinstance.get"
	topAPIListProcInstMethod   = "dingtalk.smartwork.bpms.processinstance.list"
)

type TopAPICreateProcInst struct {
	// 审批模板code
	ProcessCode string `json:"process_code"`
	// 发起人UID
	OriginatorUID string `json:"originator_user_id"`
	// 发起人所在部门
	DeptID int `json:"dept_id"`
	// 审批人列表
	Approvers []string `json:"approvers"`
	// 抄送人列表
	CCList []string `json:"cc_list"`
	//抄送时间,分为（START,FINISH,START_FINISH
	CCPosition string `json:"cc_position"`
	// 审批单内容， Name为审批模板中的列名， value 为该列的值
	FormCompntValues []ProcInstCompntValues `json:"form_component_values"`
}

type ProcInst struct {
	ProcInstID         string                      `json:"process_instance_id"`
	Title              string                      `json:"title"`
	CreateTime         string                      `json:"create_time"`
	FinishTime         string                      `json:"finish_time"`
	OriginatorUID      string                      `json:"originator_userid"`
	Status             string                      `json:"status"`
	ApproverUIDS       []string                    `json:"approver_userids"`
	CCUIDS             []string                    `json:"cc_userids"`
	Result             string                      `json:"result"`
	BusinessID         string                      `json:"business_id"`
	FormCompntValues   []ProcInstCompntValues      `json:"form_component_values"` // 表单详情列表
	Tasks              []_ProcInstTasks            `json:"tasks"`                 // 任务列表
	OperationRecords   []_ProcInstOperationRecords `json:"operation_records"`     // 操作记录列表
	OriginatorDeptID   string                      `json:"originator_dept_id"`
	OriginatorDeptName string                      `json:"originator_dept_name"`
}

type ProcInstCompntValues struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	ExtValue string `json:"ext_value"`
}

type _ProcInstOperationRecords struct {
	UID    string `json:"userid"`
	Date   string `json:"date"`
	Type   string `json:"operation_type"`
	Result string `json:"operation_result"`
	Remark string `json:"remark"`
}

type _ProcInstTasks struct {
	UID        string `json:"userid"`
	Status     string `json:"task_status"`
	Result     string `json:"task_result"`
	CreateTime string `json:"create_time"`
	FinishTime string `json:"finish_time"`
}

type topAPICreateProcInstResp struct {
	topAPIErrResponse
	OK struct {
		Errcode    int    `json:"ding_open_errcode"`
		ErrMsg     string `json:"error_msg"`
		IsSuccess  bool   `jons:"is_success"`
		ProcInstID string `json:"process_instance_id"`
	} `json:"result"`
	RequestID string `json:"request_id"`
}

// 发起审批
func (c *DingTalkClient) TopAPICreateProcInst(data TopAPICreateProcInst) (string, error) {
	var resp topAPICreateProcInstResp
	values, err := json.Marshal(data.FormCompntValues)
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Add("method", topAPICreateProcInstMethod)
	form.Add("cc_list", strings.Join(data.CCList, ","))
	form.Add("dept_id", strconv.Itoa(data.DeptID))
	form.Add("approvers", strings.Join(data.Approvers, ","))
	form.Add("cc_position", data.CCPosition)
	form.Add("process_code", data.ProcessCode)
	form.Add("originator_user_id", data.OriginatorUID)
	form.Add("form_component_values", string(values))
	if c.AgentID != "" {
		form.Add("agent_id", c.AgentID)
	}

	return resp.OK.ProcInstID, c.topAPIRequest(form, &resp)
}

type topAPIGetProcInstResp struct {
	Ok struct {
		ErrCode  int      `json:"ding_open_errcode"`
		ErrMsg   string   `json:"error_msg"`
		Success  bool     `json:"success"`
		ProcInst ProcInst `json:"process_instance"`
	} `json:"result"`
	RequestID string `json:"request_id"`
	topAPIErrResponse
}

// 根据审批实例id获取单条审批实例详情
func (c *DingTalkClient) TopAPIGetProcInst(pid string) (ProcInst, error) {
	var resp topAPIGetProcInstResp
	reqForm := url.Values{}
	reqForm.Add("process_instance_id", pid)
	reqForm.Add("method", topAPIGetProcInstMethod)
	err := c.topAPIRequest(reqForm, &resp)
	if err != nil {
		return resp.Ok.ProcInst, err
	}
	resp.Ok.ProcInst.ProcInstID = pid
	return resp.Ok.ProcInst, err
}

type ListProcInst struct {
	ApproverUIDS     []string               `json:"approver_userid_list"`
	CCUIDS           []string               `json:"cc_userid_list"`
	FormCompntValues []ProcInstCompntValues `json:"form_component_values"`
	ProcInstID       string                 `json:"process_instance_id"`
	Title            string                 `json:"title"`
	CreateTime       string                 `json:"create_time"`
	FinishTime       string                 `json:"finish_time"`
	OriginatorUID    string                 `json:"originator_userid"`
	Status           string                 `json:"status"`
	BusinessID       string                 `json:"business_id"`
	OriginatorDeptID string                 `json:"originator_dept_id"`
	ProcInstResult   string                 `json:"process_instance_result"` // "agree",
}

type TopAPIListProcInstResp struct {
	topAPIErrResponse
	OK struct {
		ErrCode int    `json:"ding_open_errcode"`
		ErrMsg  string `json:"error_msg"`
		Success bool   `json:"success"`
		Result  struct {
			List       []ListProcInst `json:"list"`
			NextCursor int            `json:"next_cursor"`
		} `json:"result"`
	} `json:"result"`
	RequestID string `json:"request_id"`
}

// 获取审批实例列表
// Note: processCode 官方不会检查错误，请保证processCode正确
func (c *DingTalkClient) TopAPIListProcInst(processCode string, startTime, endTime time.Time, size, cursor int, useridList []string) (TopAPIListProcInstResp, error) {
	var resp TopAPIListProcInstResp
	if size > 10 {
		return resp, errors.New("Max size is 10")
	}

	reqForm := url.Values{}
	reqForm.Add("process_code", processCode)
	reqForm.Add("start_time", strconv.FormatInt(startTime.UnixNano()/int64(time.Millisecond), 10))
	reqForm.Add("end_time", strconv.FormatInt(endTime.UnixNano()/int64(time.Millisecond), 10))
	reqForm.Add("size", strconv.Itoa(size))
	reqForm.Add("cursor", strconv.Itoa(cursor))
	reqForm.Add("userid_list", strings.Join(useridList, ","))
	reqForm.Add("method", topAPIListProcInstMethod)
	return resp, c.topAPIRequest(reqForm, &resp)
}
