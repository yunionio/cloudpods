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

package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/logger"
	noapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/logger/extern"
	"yunion.io/x/onecloud/pkg/logger/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

var WhiteListMap = make(map[string]bool)

func InitActionWhiteList() {
	for _, value := range logclient.WhiteList {
		WhiteListMap[value] = true
	}
}

func IsInActionWhiteList(key string) bool {
	return WhiteListMap[key]
}

type SActionlogManager struct {
	db.SOpsLogManager
	db.SRecordChecksumResourceBaseManager

	lastExceedCountNotifyTime time.Time
}

type SActionlog struct {
	db.SOpsLog
	db.SRecordChecksumResourceBase

	// 开始时间
	StartTime time.Time `nullable:"true" list:"user" create:"optional"`
	// 结果
	Success bool `list:"user" create:"required"`
	// 服务类别
	Service string `width:"32" charset:"utf8" nullable:"true" list:"user" create:"optional"`
	// 系统账号
	IsSystemAccount tristate.TriState `default:"false" list:"user" create:"optional"`

	// 用户IP
	Ip string `width:"17" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	// 风险级别 0 紧急(Emergency) 1 警报(Alert) 2 关键(Critical) 3 错误(Error) 4 警告(Warning) 5 通知(Notice) 6 信息(informational) 7 调试(debug)
	Severity api.TEventSeverity `width:"32" charset:"ascii" nullable:"false" default:"INFO" list:"user" create:"optional"`
	// 行为类别，0 一般行为(normal) 1 异常行为(abnormal) 2 违规行为(illegal)
	Kind api.TEventKind `width:"16" charset:"ascii" nullable:"false" default:"NORMAL" list:"user" create:"optional"`
}

var ActionLog *SActionlogManager
var AdminActionLog *SActionlogManager

var logQueue = make(chan *SActionlog, 50)

func InitActionLog() {
	InitActionWhiteList()
	initTable := func(tbname string) *SActionlogManager {
		tbl := &SActionlogManager{
			SOpsLogManager: db.NewOpsLogManager(SActionlog{}, tbname, "action", "actions", "ops_time", consts.OpsLogWithClickhouse),
		}
		if consts.OpsLogWithClickhouse {
			tbl.SRecordChecksumResourceBaseManager = *db.NewRecordChecksumResourceBaseManager()
		}
		return tbl
	}
	ActionLog = initTable("action_tbl")
	ActionLog.SetVirtualObject(ActionLog)
	if options.Options.EnableSeparateAdminLog {
		AdminActionLog = initTable("admin_action_tbl")
		AdminActionLog.SetVirtualObject(AdminActionLog)
	}
}

func isRoleInNames(userCred mcclient.TokenCredential, roles []string) bool {
	for _, r := range userCred.GetRoles() {
		if utils.IsInStringArray(r, roles) {
			return true
		}
	}
	return false
}

func (manager *SActionlogManager) GetImmutableInstance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) db.IModelManager {
	match := isRoleInNames(userCred, options.Options.AuditorRoleNames)
	log.Debugf("roles: %s auditor: %s match auditor: %v", userCred.GetRoles(), options.Options.AuditorRoleNames, match)
	if options.Options.EnableSeparateAdminLog && match {
		return AdminActionLog
	}
	return ActionLog
}

func (manager *SActionlogManager) GetMutableInstance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) db.IModelManager {
	matchSec := isRoleInNames(userCred, options.Options.SecadminRoleNames)
	matchOps := isRoleInNames(userCred, options.Options.OpsadminRoleNames)
	log.Debugf("roles: %s sec: %s match: %v ops: %s match: %v", userCred.GetRoles(), options.Options.SecadminRoleNames, matchSec, options.Options.OpsadminRoleNames, matchOps)
	if options.Options.EnableSeparateAdminLog && (matchSec || matchOps) {
		return AdminActionLog
	}
	return ActionLog
}

func (action *SActionlog) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	log.Debugf("action: %s", data)
	now := time.Now().UTC()
	action.OpsTime = now
	if action.StartTime.IsZero() {
		action.StartTime = now
	}
	if len(action.Severity) == 0 {
		if action.Success {
			action.Severity = api.SeverityInfo
		} else {
			action.Severity = api.SeverityError
		}
	}
	return action.SOpsLog.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SActionlog) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SOpsLog.PostCreate(ctx, userCred, ownerId, query, data)
	parts := []string{}
	// 厂商代码
	parts = append(parts, options.Options.SyslogVendorCode)
	// 事件ID
	parts = append(parts, fmt.Sprintf("%d", self.Id))
	// 用户名
	parts = append(parts, self.User)
	// 模块类型
	parts = append(parts, self.Service)
	// 事件时间
	parts = append(parts, timeutils.MysqlTime(self.OpsTime))
	// 事件类型
	parts = append(parts, self.Action)
	succ := "success"
	if !self.Success {
		succ = "fail"
	}
	// 事件结果
	parts = append(parts, succ)
	// 事件描述
	notes := fmt.Sprintf("%s %s %s(%s)", self.Action, self.ObjType, self.ObjName, self.ObjId)
	if len(self.Notes) > 0 {
		notes = fmt.Sprintf("%s: %s", notes, self.Notes)
	}
	notes = extern.FormatMsg(notes, options.Options.SyslogSeparator, options.Options.SyslogSepEscape)
	parts = append(parts, notes)
	kind := ""
	switch self.Kind {
	case api.KindNormal:
		kind = "0"
	case api.KindAbnormal:
		kind = "1"
	case api.KindIllegal:
		kind = "2"
	}
	// 行为类别
	parts = append(parts, kind)
	parts = append(parts, "")

	msg := strings.Join(parts, options.Options.SyslogSeparator)
	if len(self.Severity) == 0 {
		if self.Success {
			extern.Info(msg)
		} else {
			extern.Error(msg)
		}
	} else {
		switch self.Severity {
		case api.SeverityEmergency:
			extern.Emergency(msg)
		case api.SeverityAlert:
			extern.Alert(msg)
		case api.SeverityCritical:
			extern.Critical(msg)
		case api.SeverityError:
			extern.Error(msg)
		case api.SeverityWarning:
			extern.Warning(msg)
		case api.SeverityNotice:
			extern.Notice(msg)
		case api.SeverityInfo:
			extern.Info(msg)
		case api.SeverityDebug:
			extern.Debug(msg)
		default:
			extern.Info(msg)
		}
	}
	for k, v := range map[string]string{
		"service":  self.Service,
		"action":   self.Action,
		"obj_type": self.ObjType,
		"severity": string(self.Severity),
		"kind":     string(self.Kind),
	} {
		db.DistinctFieldManager.InsertOrUpdate(ctx, ActionLog, k, v)
	}

	ActionLog.notifyIfExceedCount(ctx, self)
}

func (manager *SActionlogManager) notifyIfExceedCount(ctx context.Context, l *SActionlog) {
	exceedCnt := options.Options.ActionLogExceedCount
	if exceedCnt <= 0 {
		return
	}

	interval := utils.ToDuration(options.Options.ActionLogExceedCountNotifyInterval)
	if time.Since(manager.lastExceedCountNotifyTime) < interval {
		return
	}
	totalCnt := ActionLog.Query().Count()
	if totalCnt >= exceedCnt {
		result := map[string]interface{}{
			"action":        jsonutils.Marshal(l),
			"current_count": totalCnt,
			"exceed_count":  exceedCnt,
		}
		resultObj := jsonutils.Marshal(result).(*jsonutils.JSONDict)
		notifyclient.SystemExceptionNotifyWithResult(ctx, noapi.ActionExceedCount, noapi.TOPIC_RESOURCE_ACTION_LOG, "", resultObj)
		manager.lastExceedCountNotifyTime = time.Now()
	}
}

// 操作日志列表
func (manager *SActionlogManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.ActionLogListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SOpsLogManager.ListItemFilter(ctx, q, userCred, input.OpsLogListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "ListItemFilter")
	}

	if len(input.Service) > 0 {
		if len(input.Service) == 1 {
			q = q.Equals("service", input.Service[0])
		} else {
			q = q.In("service", input.Service)
		}
	}

	if input.Success != nil {
		q = q.Equals("success", *input.Success)
	}

	if input.IsSystemAccount != nil {
		q = q.Equals("is_system_account", *input.IsSystemAccount)
	}

	if len(input.Ip) > 0 {
		if len(input.Ip) == 1 {
			q = q.Equals("ip", input.Ip[0])
		} else {
			q = q.In("ip", input.Ip)
		}
	}

	if len(input.Severity) > 0 {
		if len(input.Severity) == 1 {
			q = q.Equals("severity", input.Severity[0])
		} else {
			q = q.In("severity", input.Severity)
		}
	}

	if len(input.Kind) > 0 {
		if len(input.Kind) == 1 {
			q = q.Equals("kind", input.Kind[0])
		} else {
			q = q.In("kind", input.Kind)
		}
	}

	return q, nil
}

func (manager *SActionlogManager) GetPropertyDistinctField(ctx context.Context, userCred mcclient.TokenCredential, input apis.DistinctFieldInput) (jsonutils.JSONObject, error) {
	fields, err := db.DistinctFieldManager.GetObjectDistinctFields(manager.Keyword())
	if err != nil {
		return nil, errors.Wrapf(err, "DistinctFieldManager.GetObjectDistinctFields")
	}
	fieldMaps := map[string][]string{}
	for _, field := range fields {
		_, ok := fieldMaps[field.Key]
		if !ok {
			fieldMaps[field.Key] = []string{}
		}
		fieldMaps[field.Key] = append(fieldMaps[field.Key], field.Value)
	}
	ret := map[string][]string{}
	for _, key := range input.Field {
		ret[key], _ = fieldMaps[key]
	}
	return jsonutils.Marshal(ret), nil
}

func (action *SActionlog) GetI18N(ctx context.Context) *jsonutils.JSONDict {
	r := jsonutils.NewDict()
	act18 := logclient.OpsActionI18nTable.Lookup(ctx, action.Action)
	ser18 := logclient.OpsServiceI18nTable.Lookup(ctx, action.Service)
	obj18 := logclient.OpsObjTypeI18nTable.Lookup(ctx, action.ObjType)
	r.Set("action", jsonutils.NewString(act18))
	r.Set("service", jsonutils.NewString(ser18))
	r.Set("obj_type", jsonutils.NewString(obj18))
	return r
}

func (action *SActionlog) GetModelManager() db.IModelManager {
	return action.SModelBase.GetModelManager()
}

func (man *SActionlogManager) GetI18N(ctx context.Context, idstr string, resObj jsonutils.JSONObject) *jsonutils.JSONDict {
	if idstr != "distinct-field" {
		return nil
	}
	res := &struct {
		Action  []string `json:"action"`
		Service []string `json:"service"`
		ObjType []string `json:"obj_type"`
	}{}
	if err := resObj.Unmarshal(res); err != nil {
		return nil
	}
	for i, act := range res.Action {
		act18 := logclient.OpsActionI18nTable.Lookup(ctx, act)
		res.Action[i] = act18
	}
	for i, ser := range res.Service {
		ser18 := logclient.OpsServiceI18nTable.Lookup(ctx, ser)
		res.Service[i] = ser18
	}
	for i, obj := range res.ObjType {
		obj18 := logclient.OpsObjTypeI18nTable.Lookup(ctx, obj)
		res.ObjType[i] = obj18
	}
	robj := jsonutils.Marshal(res)
	rdict := robj.(*jsonutils.JSONDict)
	return rdict
}

// Websockets 不再拉取 ActionLog 的消息，因此注释掉如下代码
// 可以保留，以便有需求时，再次打开
// func (manager *SActionlogManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
//	actionLog := items[0].(*SActionlog)
//	if IsInActionWhiteList(actionLog.Action) {
//		select {
//		case logQueue <- actionLog:
//			return
//		default:
//			log.Warningf("Log queue full, insert failed, log ignored: %s", actionLog.Action)
//		}
//	}
// }
//
// func StartNotifyToWebsocketWorker() {
// 	go func() {
// 		for {
// 			actionLog := <-logQueue
// 			params := jsonutils.Marshal(actionLog)
// 			s := auth.GetAdminSession(context.Background(), "", "")
// 			_, err := websocket.Websockets.PerformClassAction(s, "action-log", params)
// 			if err != nil {
// 				log.Errorf("Send action log error %s", err)
// 			}
// 		}
// 	}()
// }

func (manager *SActionlogManager) InitializeData() error {
	fileds, err := db.DistinctFieldManager.GetObjectDistinctFields(manager.Keyword())
	if err != nil {
		return errors.Wrapf(err, "GetObjectDistinctFields")
	}
	if len(fileds) > 0 {
		return nil
	}
	for _, key := range []string{"service", "obj_type", "action", "severity", "kind"} {
		values, err := db.FetchDistinctField(manager, key)
		if err != nil {
			return errors.Wrapf(err, "db.FetchDistinctField")
		}
		for _, value := range values {
			if len(value) > 0 {
				err = db.DistinctFieldManager.InsertOrUpdate(context.TODO(), manager, key, value)
				if err != nil {
					return errors.Wrapf(err, "DistinctFieldManager.InsertOrUpdate(%s, %s)", key, value)
				}
			}
		}
	}
	return nil
}
