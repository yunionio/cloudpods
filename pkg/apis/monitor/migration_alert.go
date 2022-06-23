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

package monitor

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
)

func init() {
	for _, drv := range []IMigrationAlertMetric{
		newMMemAailable(),
		newMCPUUsageActive(),
	} {
		GetMigrationAlertMetricDrivers().register(drv)
	}
}

var (
	migMetricDrivers *MigrationMetricDrivers
)

func GetMigrationAlertMetricDrivers() *MigrationMetricDrivers {
	if migMetricDrivers == nil {
		migMetricDrivers = newMigrationMetricDrivers()
	}
	return migMetricDrivers
}

type MigrationMetricDrivers struct {
	*sync.Map
}

func newMigrationMetricDrivers() *MigrationMetricDrivers {
	return &MigrationMetricDrivers{
		Map: new(sync.Map),
	}
}

func (d *MigrationMetricDrivers) register(di IMigrationAlertMetric) *MigrationMetricDrivers {
	d.Store(di.GetType(), di)
	return d
}

func (d *MigrationMetricDrivers) Get(t MigrationAlertMetricType) (IMigrationAlertMetric, error) {
	obj, ok := d.Load(t)
	if !ok {
		return nil, fmt.Errorf("driver type %q not found", t)
	}
	return obj.(IMigrationAlertMetric), nil
}

type MigrationAlertMetricType string

const (
	MigrationAlertMetricTypeCPUUsageActive = "cpu.usage_active"
	MigrationAlertMetricTypeMemAvailable   = "mem.available"
)

type MetricQueryFields struct {
	ResourceType MigrationAlertResourceType
	Database     string
	Measurement  string
	Field        string
	Comparator   string
}

type IMigrationAlertMetric interface {
	GetType() MigrationAlertMetricType
	GetQueryFields() *MetricQueryFields
}

type mMemAvailable struct{}

func newMMemAailable() IMigrationAlertMetric {
	return new(mMemAvailable)
}

func (_ mMemAvailable) GetType() MigrationAlertMetricType {
	return MigrationAlertMetricTypeMemAvailable
}

func (_ mMemAvailable) GetQueryFields() *MetricQueryFields {
	return &MetricQueryFields{
		ResourceType: MigrationAlertResourceTypeHost,
		Database:     METRIC_DATABASE_TELE,
		Measurement:  "mem",
		Field:        "available",
		Comparator:   ConditionLessThan, // <
	}
}

type mCPUUsageActive struct{}

func newMCPUUsageActive() IMigrationAlertMetric {
	return new(mCPUUsageActive)
}

func (_ mCPUUsageActive) GetType() MigrationAlertMetricType {
	return MigrationAlertMetricTypeCPUUsageActive
}

func (_ mCPUUsageActive) GetQueryFields() *MetricQueryFields {
	return &MetricQueryFields{
		ResourceType: MigrationAlertResourceTypeHost,
		Database:     METRIC_DATABASE_TELE,
		Measurement:  "cpu",
		Field:        "usage_active",
		Comparator:   ConditionGreaterThan, // >
	}
}

func IsValidMigrationAlertMetricType(t MigrationAlertMetricType) error {
	_, err := GetMigrationAlertMetricDrivers().Get(t)
	return err
}

type MigrationAlertResourceType string

const (
	MigrationAlertResourceTypeHost = "host"
)

type MigrationAlertCreateInput struct {
	AlertCreateInput
	// Threshold is the value to trigger migration
	Threshold float64 `json:"threshold"`
	// Period of querying metrics
	Period string `json:"period"`
	// MetricType is supported metric type by auto migration
	MetricType MigrationAlertMetricType `json:"metric_type"`
	// MigrationAlertSettings contain migration configuration
	MigrationSettings *MigrationAlertSettings `json:"migration_settings"`
}

func (m MigrationAlertCreateInput) GetMetricDriver() IMigrationAlertMetric {
	d, err := GetMigrationAlertMetricDrivers().Get(m.MetricType)
	if err != nil {
		panic(err)
	}
	return d
}

func (m MigrationAlertCreateInput) ToAlertCreateInput() *AlertCreateInput {
	freq, _ := time.ParseDuration(m.Period)
	ret := new(AlertCreateInput)
	ret.Name = m.Name
	ret.Frequency = int64(freq / time.Second)
	ret.Level = m.Level
	ret.CustomizeConfig = jsonutils.Marshal(m.MigrationSettings)

	drv := m.GetMetricDriver()
	fs := drv.GetQueryFields()

	ret.Settings = AlertSetting{
		Conditions: []AlertCondition{
			{
				Type:     "query",
				Operator: "and",
				Query: AlertQuery{
					Model: m.getQuery(fs),
					From:  m.Period,
					To:    "now",
				},
				Evaluator: m.GetEvaluator(fs),
				Reducer:   Condition{Type: "avg"},
			},
		},
	}

	return ret
}

func (m MigrationAlertCreateInput) GetEvaluator(fs *MetricQueryFields) Condition {
	return Condition{
		Type:      fs.Comparator,
		Operators: nil,
		Params:    []float64{m.Threshold},
	}
}

func (m MigrationAlertCreateInput) getQuery(fs *MetricQueryFields) MetricQuery {
	sels := make([]MetricQuerySelect, 0)
	sels = append(sels, NewMetricQuerySelect(
		MetricQueryPart{
			Type:   "field",
			Params: []string{fs.Field},
		},
		MetricQueryPart{
			Type:   "mean",
			Params: nil,
		},
	))
	q := MetricQuery{
		Selects: sels,
		GroupBy: []MetricQueryPart{
			{
				Type:   "field",
				Params: []string{"*"},
			},
			{
				Type:   "fill",
				Params: []string{"null"},
			},
		},
		Measurement: fs.Measurement,
		Database:    fs.Database,
	}
	q.Tags = []MetricQueryTag{
		{
			Condition: "and",
			Key:       "res_type",
			Operator:  "=",
			Value:     "host",
		},
	}
	if m.MigrationSettings != nil {
		if m.MigrationSettings.Source != nil && len(m.MigrationSettings.Source.HostIds) > 0 {
			ids := strings.Join(m.MigrationSettings.Source.HostIds, "|")
			q.Tags = append(q.Tags, MetricQueryTag{
				Key:      "host_id",
				Operator: "=~",
				Value:    fmt.Sprintf("/%s/", ids),
			})
		}
	}
	return q
}

type MigrationAlertSettings struct {
	Source *MigrationAlertSettingsSource `json:"source"`
	Target *MigrationAlertSettingsTarget `json:"target"`
}

type MigrationAlertSettingsSource struct {
	GuestIds []string `json:"guest_ids"`
	HostIds  []string `json:"host_ids"`
}

type MigrationAlertSettingsTarget struct {
	HostIds []string `json:"host_ids"`
}

type MigrationAlertListInput struct {
	AlertListInput

	MetricType string `json:"metric_type"`
}
