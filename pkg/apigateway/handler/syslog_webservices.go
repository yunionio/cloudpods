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

package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/logger"
)

type SActionLog struct {
	User     string
	Severity string
	Service  string
	Ip       string
	Notes    string
	Kind     string
	OpsTime  time.Time
	ObjType  string
	Id       uint64
	Success  *bool
	Action   string
}

func (a SActionLog) toMsg() Message {
	result := "fail"
	if a.Success != nil && *a.Success {
		result = "success"
	}
	level := 0
	switch a.Severity {
	case "EMERGENCY":
		level = 0
	case "ALERT":
		level = 1
	case "CRITICAL":
		level = 2
	case "ERROR":
		level = 3
	case "WARNING":
		level = 4
	case "NOTICE":
		level = 5
	case "INFO":
		level = 6
	case "DEBUG":
		level = 7
	}
	kind := 0
	switch a.Kind {
	case "NORMAL":
		kind = 0
	case "ABNORMAL":
		kind = 1
	case "ILLEGAL":
		kind = 2
	}
	return Message{
		RiskLevel:        level,
		SendIP:           a.Ip,
		ManufacturerCode: "0003",
		EventId:          fmt.Sprintf("%d", a.Id),
		Username:         a.User,
		ModuleType:       a.Service,
		EventDate:        timeutils.MysqlTime(a.OpsTime.Add(8 * time.Hour)),
		EventType:        a.Action,
		EventResult:      result,
		EventDesc:        a.Notes,
		BehaviorType:     kind,
	}
}

type Message struct {
	RiskLevel int `json:"riskLevel"`

	SendIP string `json:"sendIP"`

	ManufacturerCode string `json:"manufacturerCode"`

	EventId string `json:"eventId"`

	Username string `json:"username"`

	ModuleType string `json:"moduleType"`

	EventDate string `json:"eventDate"`

	EventType string `json:"eventType"`

	EventResult string `json:"eventResult"`

	EventDesc string `json:"eventDesc"`

	BehaviorType int `json:"behaviorType"`
}

type msgResponse struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    []Message `json:"data"`
	Date    string    `json:"date"`
	Count   int       `json:"count"`
}

func handleSyslogWebServiceMessage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if !options.Options.EnableSyslogWebservice {
		httperrors.ForbiddenError(ctx, w, "syslog webservice not enabled")
		return
	}
	resp := fetchSyslogMessage(r)
	appsrv.SendJSON(w, resp)
}

func fetchSyslogMessage(r *http.Request) jsonutils.JSONObject {
	token := r.FormValue("token")
	date := r.FormValue("date")
	eventId := r.FormValue("eventId")
	recordSize := r.FormValue("recordSize")
	// recordStart := r.FormValue("recordStart")
	moduleType := r.FormValue("moduleType")

	ret := msgResponse{}

	ntoken := genToken(options.Options.SyslogWebserviceUsername, options.Options.SyslogWebservicePassword)
	if ntoken != token {
		ret.Code = 2
		ret.Message = "token无效"
		return jsonutils.Marshal(ret)
	}

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("desc"), "order")
	params.Add(jsonutils.NewString("DESC"), "paging_order")
	if len(date) > 0 {
		params.Add(jsonutils.NewString(date), "since")
	}
	if len(eventId) > 0 {
		params.Add(jsonutils.NewString(eventId), "paging_marker")
	}
	limit, _ := strconv.ParseInt(recordSize, 10, 64)
	if limit > 0 {
		params.Add(jsonutils.NewInt(limit), "limit")
	}
	if len(moduleType) > 0 {
		params.Add(jsonutils.NewString(moduleType), "service")
	}

	sess := auth.GetAdminSession(nil, "")
	logs, err := modules.Actions.List(sess, params)
	if err != nil {
		ret.Code = 2
		ret.Message = fmt.Sprintf("list fail %s", err)
		return jsonutils.Marshal(ret)
	}

	for i := range logs.Data {
		action := SActionLog{}
		err := logs.Data[i].Unmarshal(&action)
		if err != nil {
			continue
		}
		msg := action.toMsg()
		ret.Data = append(ret.Data, msg)
		ret.Date = msg.EventDate
	}

	ret.Code = 1
	ret.Count = len(ret.Data)
	ret.Message = "成功"
	return jsonutils.Marshal(ret)
}

type authResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Token   string `json:"token"`
}

func genToken(uname string, passwd string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(uname+":"+passwd)).String()
}

func handleSyslogWebServiceToken(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if !options.Options.EnableSyslogWebservice {
		httperrors.ForbiddenError(ctx, w, "syslog webservice not enabled")
		return
	}
	resp := fetchSyslogToken(r)
	appsrv.SendJSON(w, resp)
}

func fetchSyslogToken(r *http.Request) jsonutils.JSONObject {
	uname := r.FormValue("username")
	passwd := r.FormValue("password")
	ret := authResponse{}

	if uname == options.Options.SyslogWebserviceUsername && passwd == options.Options.SyslogWebservicePassword {
		// succ
		token := genToken(uname, passwd)
		ret.Code = "1"
		ret.Message = "成功"
		ret.Token = token
	} else {
		// fail
		ret.Code = "2"
		ret.Message = "不匹配的username/password"
	}
	return jsonutils.Marshal(ret)
}
