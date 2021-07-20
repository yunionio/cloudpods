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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type SMetricManager struct {
	db.SJointResourceBaseManager
}

type SMetric struct {
	db.SVirtualJointResourceBase
	MeasurementId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	FieldId       string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

var MetricManager *SMetricManager

func init() {
	MetricManager = &SMetricManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			SMetric{},
			"metric_tbl",
			"metric",
			"metrics",
			MetricMeasurementManager,
			MetricFieldManager,
		),
	}
	MetricManager.SetVirtualObject(MetricManager)
	MetricManager.TableSpec().AddIndex(true, MetricManager.GetMasterFieldName(), MetricManager.GetSlaveFieldName())
}

func (man *SMetricManager) GetMasterFieldName() string {
	return "measurement_id"
}

func (man *SMetricManager) GetSlaveFieldName() string {
	return "field_id"
}

func (metric *SMetric) DoSave(ctx context.Context) error {
	if err := MetricManager.TableSpec().Insert(ctx, metric); err != nil {
		return err
	}
	metric.SetModelManager(MetricManager, metric)
	return nil
}

func (self *SMetric) GetMetricField() (*SMetricField, error) {
	return MetricFieldManager.GetFieldByIdOrName(self.FieldId, auth.AdminCredential())
}

func (joint *SMetric) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, joint)
}
