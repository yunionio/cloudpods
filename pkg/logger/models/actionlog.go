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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
	"yunion.io/x/sqlchemy/backends/clickhouse"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/logger"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/logger/extern"
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
}

type SActionlog struct {
	db.SOpsLog
	db.SRecordChecksumResourceBase

	StartTime time.Time `nullable:"true" list:"user" create:"optional"`
	Success   bool      `list:"user" create:"required"`
	Service   string    `width:"32" charset:"utf8" nullable:"true" list:"user" create:"optional"`
}

var ActionLog *SActionlogManager
var logQueue = make(chan *SActionlog, 50)

func InitActionLog() {
	InitActionWhiteList()
	if consts.OpsLogWithClickhouse {
		ActionLog = &SActionlogManager{
			SOpsLogManager: db.SOpsLogManager{
				SModelBaseManager: db.NewModelBaseManagerWithDBName(
					SActionlog{},
					"action_tbl",
					"action",
					"actions",
					db.ClickhouseDB,
				),
			},
		}
		col := ActionLog.TableSpec().ColumnSpec("ops_time")
		if clickCol, ok := col.(clickhouse.IClickhouseColumnSpec); ok {
			clickCol.SetTTL(consts.SplitableMaxKeepMonths(), "MONTH")
		}
	} else {
		ActionLog = &SActionlogManager{
			SOpsLogManager: db.SOpsLogManager{
				SModelBaseManager: db.NewModelBaseManagerWithSplitable(
					SActionlog{},
					"action_tbl",
					"action",
					"actions",
					"id",
					"start_time",
					consts.SplitableMaxDuration(),
					consts.SplitableMaxKeepMonths(),
				),
			},
			SRecordChecksumResourceBaseManager: *db.NewRecordChecksumResourceBaseManager(),
		}
	}
	ActionLog.SetVirtualObject(ActionLog)
}

func (action *SActionlog) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	now := time.Now().UTC()
	action.OpsTime = now
	if action.StartTime.IsZero() {
		action.StartTime = now
	}
	return action.SOpsLog.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (self *SActionlog) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SOpsLog.PostCreate(ctx, userCred, ownerId, query, data)
	msg := fmt.Sprintf("#%d ", self.Id)
	msg += fmt.Sprintf("%s %s %s %s %s ", self.Service, self.ObjType, self.ObjName, self.ObjId, self.Action)
	msg += fmt.Sprintf("%s %s %s %s %s %s", self.Domain, self.Project, self.User, self.OwnerDomainId, self.OwnerProjectId, self.Notes)
	if self.Success {
		extern.Info(msg)
	} else {
		extern.Error(msg)
	}
	for k, v := range map[string]string{
		"service":  self.Service,
		"action":   self.Action,
		"obj_type": self.ObjType,
	} {
		db.DistinctFieldManager.InsertOrUpdate(ctx, ActionLog, k, v)
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
	for _, key := range []string{"service", "obj_type", "action"} {
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
