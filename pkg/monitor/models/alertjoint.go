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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SAlertJointsManager struct {
	db.SJointResourceBaseManager
}

func NewAlertJointsManager(
	dt interface{}, tableName string,
	keyword string, keywordPlural string,
	slave db.IStandaloneModelManager) SAlertJointsManager {
	return SAlertJointsManager{
		db.NewJointResourceBaseManager(
			dt, tableName, keyword, keywordPlural, AlertManager, slave),
	}
}

// +onecloud:swagger-gen-ignore
type SAlertJointsBase struct {
	db.SVirtualJointResourceBase

	AlertId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (b *SAlertJointsBase) getAlert() *SAlert {
	alert, _ := AlertManager.GetAlert(b.AlertId)
	return alert
}

func (man *SAlertJointsManager) GetMasterFieldName() string {
	return "alert_id"
}

func (man *SAlertJointsManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.AlertJointListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.JointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SJointResourceBaseManager.ListItemFilter")
	}
	if len(query.AlertIds) != 0 {
		q = q.In("alert_id", query.AlertIds)
	}
	return q, nil
}

func (man *SAlertJointsManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.AlertJointResourceBaseDetails {
	rows := make([]monitor.AlertJointResourceBaseDetails, len(objs))
	jointRows := man.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	alertIds := make([]string, len(rows))
	for i := range jointRows {
		rows[i] = monitor.AlertJointResourceBaseDetails{
			JointResourceBaseDetails: jointRows[i],
		}
		var base *SAlertJointsBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil && len(base.AlertId) > 0 {
			alertIds[i] = base.AlertId
		}
	}

	alertIdMaps, err := db.FetchIdNameMap2(AlertManager, alertIds)
	if err != nil {
		log.Errorf("alert joints FetchIdNameMap2 fail: %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := alertIdMaps[alertIds[i]]; ok {
			rows[i].Alert = name
		}
	}

	return rows
}
