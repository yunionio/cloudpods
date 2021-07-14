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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	MonitorResourceAlertManager *SMonitorResourceAlertManager
)

func init() {
	MonitorResourceAlertManager = &SMonitorResourceAlertManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			SMonitorResourceAlert{},
			"monitor_resource_alert_tbl",
			"monitorresourcealert",
			"monitorresourcealerts",
			MonitorResourceManager, CommonAlertManager),
	}
	MonitorResourceAlertManager.SetVirtualObject(MonitorResourceAlertManager)
}

type SMonitorResourceAlertManager struct {
	db.SJointResourceBaseManager
}

type SMonitorResourceAlert struct {
	db.SJointResourceBase

	MonitorResourceId string               `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true" json:"monitor_resource_id"`
	AlertId           string               `width:"36" charset:"ascii" list:"user" create:"required" index:"true"`
	AlertRecordId     string               `width:"36" charset:"ascii" list:"user" update:"user"`
	AlertState        string               `width:"18" charset:"ascii" default:"init" list:"user" update:"user"`
	TriggerTime       time.Time            `list:"user"  update:"user" json:"trigger_time"`
	Data              jsonutils.JSONObject `list:"user"  update:"user"`
}

func (manager *SMonitorResourceAlertManager) GetMasterFieldName() string {
	return "monitor_resource_id"
}

func (manager *SMonitorResourceAlertManager) GetSlaveFieldName() string {
	return "alert_id"
}

func (manager *SMonitorResourceAlertManager) DetachJoint(ctx context.Context, userCred mcclient.TokenCredential,
	input monitor.MonitorResourceJointListInput) error {
	joints, err := manager.GetJoinsByListInput(input)
	if err != nil {
		return errors.Wrapf(err, "SMonitorResourceAlertManager DetachJoint when  GetJoinsByListInput err,input:%v", input)
	}
	errs := make([]error, 0)
	for _, joint := range joints {
		err := joint.Delete(ctx, nil)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "joint %s:%s ,%s:%s", manager.GetMasterFieldName(),
				joint.MonitorResourceId, manager.GetSlaveFieldName(), joint.AlertId))
			continue
		}
		resources, err := MonitorResourceManager.GetMonitorResources(monitor.MonitorResourceListInput{ResId: []string{joint.
			MonitorResourceId}})
		if err != nil {
			errs = append(errs, err)
		}
		for _, res := range resources {
			err := res.UpdateAlertState()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.NewAggregate(errs)
}

func (obj *SMonitorResourceAlert) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SMonitorResourceAlert) GetAlert() (*SCommonAlert, error) {
	sObj, err := CommonAlertManager.FetchById(obj.AlertId)
	if err != nil {
		return nil, err
	}
	return sObj.(*SCommonAlert), nil
}

func (manager *SMonitorResourceAlertManager) GetJoinsByListInput(input monitor.MonitorResourceJointListInput) ([]SMonitorResourceAlert, error) {
	joints := make([]SMonitorResourceAlert, 0)
	query := manager.Query()

	if len(input.MonitorResourceId) != 0 {
		query.Equals(manager.GetMasterFieldName(), input.MonitorResourceId)
	}
	if len(input.AlertId) != 0 {
		query.Equals(manager.GetSlaveFieldName(), input.AlertId)
	}
	if len(input.JointId) != 0 {
		query.In("row_id", input.JointId)
	}
	err := db.FetchModelObjects(manager, query, &joints)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchModelObjects by GetJoinsByMasterId:%s err", input.MonitorResourceId)
	}
	return joints, nil
}

func (obj *SMonitorResourceAlert) UpdateAlertRecordData(record *SAlertRecord, match *monitor.EvalMatch) error {
	if _, err := db.Update(obj, func() error {
		obj.AlertRecordId = record.GetId()
		obj.AlertState = record.State
		obj.TriggerTime = record.CreatedAt
		obj.Data = jsonutils.Marshal(match)
		return nil
	}); err != nil {
		return err
	}
	return nil
}
