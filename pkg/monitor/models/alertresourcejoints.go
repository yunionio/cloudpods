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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type IAlertResourceJointModel interface {
	db.IJointModel

	GetAlertResource() (*SAlertResource, error)
	GetDetails(base monitor.AlertResourceJointBaseDetails, isList bool) interface{}
}

// +onecloud:swagger-gen-ignore
type SAlertResourceJointsManager struct {
	db.SJointResourceBaseManager
}

// +onecloud:swagger-gen-ignore
type SAlertResourceJointsBase struct {
	db.SJointResourceBase

	AlertResourceId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func NewAlertResourceJointManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IStandaloneModelManager) *SAlertResourceJointsManager {
	return &SAlertResourceJointsManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			dt, tableName, keyword, keywordPlural, GetAlertResourceManager(), slave,
		),
	}
}

func (m SAlertResourceJointsManager) GetMasterFieldName() string {
	return "alert_resource_id"
}

func (obj *SAlertResourceJointsBase) GetAlertResource() (*SAlertResource, error) {
	mMan := obj.GetJointModelManager().GetMasterManager()
	mObj, err := mMan.FetchById(obj.AlertResourceId)
	if err != nil {
		return nil, err
	}
	return mObj.(*SAlertResource), nil
}

func (m *SAlertResourceJointsManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []interface{} {
	baseGet := func(obj interface{}) interface{} {
		jRows := m.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, []interface{}{obj}, fields, isList)
		return jRows[0]
	}
	ret := make([]interface{}, len(objs))
	for idx := range objs {
		obj := objs[idx].(IAlertResourceJointModel)
		baseDetail := baseGet(obj).(apis.JointResourceBaseDetails)
		outBase := monitor.AlertResourceJointBaseDetails{
			JointResourceBaseDetails: baseDetail,
		}
		resObj, err := obj.GetAlertResource()
		if err == nil {
			outBase.AlertResource = resObj.GetName()
			outBase.Type = resObj.GetType()
		} else {
			log.Errorf("Get alert resource error: %v", err)
		}
		out := obj.GetDetails(outBase, isList)
		ret[idx] = out
	}
	return ret
}
