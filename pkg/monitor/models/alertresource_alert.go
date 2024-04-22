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
	"database/sql"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	alertResourceAlertManager *SAlertResourceAlertManager
)

func init() {
	GetAlertResourceAlertManager()
}

func GetAlertResourceAlertManager() *SAlertResourceAlertManager {
	if alertResourceAlertManager == nil {
		alertResourceAlertManager = &SAlertResourceAlertManager{
			SAlertResourceJointsManager: *NewAlertResourceJointManager(
				SAlertResourceAlert{},
				"alertresourcealerts_tbl",
				"alertresourcealert",
				"alertresourcealerts",
				AlertManager,
			),
		}
		alertResourceAlertManager.SetVirtualObject(alertResourceAlertManager)
	}
	return alertResourceAlertManager
}

// +onecloud:swagger-gen-ignore
type SAlertResourceAlertManager struct {
	SAlertResourceJointsManager
}

// +onecloud:swagger-gen-ignore
type SAlertResourceAlert struct {
	SAlertResourceJointsBase

	AlertId       string               `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	AlertRecordId string               `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	TriggerTime   time.Time            `nullable:"false" index:"true" get:"user" list:"user" json:"trigger_time"`
	Data          jsonutils.JSONObject `create:"required" list:"user"`
}

func (m *SAlertResourceAlertManager) GetSlaveFieldName() string {
	return "alert_id"
}

func (m *SAlertResourceAlertManager) GetJointAlert(res *SAlertResource, alertId string) (*SAlertResourceAlert, error) {
	q := m.Query().Equals(m.GetMasterFieldName(), res.GetId()).Equals(m.GetSlaveFieldName(), alertId)
	obj, err := db.NewModelObject(m)
	if err != nil {
		return nil, errors.Wrapf(err, "NewModelObject %s", m.Keyword())
	}
	if err := q.First(obj); err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return nil, err
		} else {
			return nil, nil
		}
	}
	return obj.(*SAlertResourceAlert), nil
}

func (obj *SAlertResourceAlert) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SAlertResourceAlert) GetData() (*monitor.EvalMatch, error) {
	out := new(monitor.EvalMatch)
	if err := obj.Data.Unmarshal(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (obj *SAlertResourceAlert) UpdateData(record *SAlertRecord, match *monitor.EvalMatch) error {
	if _, err := db.Update(obj, func() error {
		obj.AlertRecordId = record.GetId()
		obj.TriggerTime = record.CreatedAt
		obj.Data = jsonutils.Marshal(match)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (m *SAlertResourceAlertManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *monitor.AlertResourceAlertListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SAlertResourceJointsManager.ListItemFilter(ctx, q, userCred, input.JointResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if len(input.AlertResourceId) > 0 {
		masterM := m.GetMasterManager()
		obj, err := masterM.FetchByIdOrName(ctx, userCred, input.AlertResourceId)
		if err != nil {
			return nil, errors.Wrapf(err, "Get %s object", masterM.Keyword())
		}
		q = q.Equals(m.GetMasterFieldName(), obj.GetId())
	}
	if len(input.AlertId) > 0 {
		slaveM := m.GetSlaveManager()
		obj, err := slaveM.FetchByIdOrName(ctx, userCred, input.AlertId)
		if err != nil {
			return nil, errors.Wrapf(err, "Get %s object", slaveM.Keyword())
		}
		q = q.Equals(m.GetSlaveFieldName(), obj.GetId())
	}
	return q, nil
}

func (obj *SAlertResourceAlert) GetAlert() (*SCommonAlert, error) {
	// sMan := obj.GetJointModelManager().GetSlaveManager()
	sMan := CommonAlertManager
	sObj, err := sMan.FetchById(obj.AlertId)
	if err != nil {
		return nil, err
	}
	return sObj.(*SCommonAlert), nil
}

func (obj *SAlertResourceAlert) GetDetails(base monitor.AlertResourceJointBaseDetails, isList bool) interface{} {
	out := monitor.AlertResourceAlertDetails{
		AlertResourceJointBaseDetails: base,
	}
	alert, err := obj.GetAlert()
	if err == nil {
		out.Alert = alert.GetName()
		out.AlertType = alert.getAlertType()
		out.Level = alert.Level
		metricDetails, err := alert.GetCommonAlertMetricDetails()
		if err != nil {
			log.Errorf("GetCommonAlertMetricDetails error: %v", err)
		}
		out.CommonAlertMetricDetails = metricDetails
	} else {
		log.Errorf("Get alert error: %v", err)
	}
	return out
}
