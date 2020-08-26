package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
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
