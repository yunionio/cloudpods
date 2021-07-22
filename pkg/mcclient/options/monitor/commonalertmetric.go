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
	"yunion.io/x/jsonutils"

	monitorapi "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type MonitorMetricListOptions struct {
	options.BaseListOptions
	MeasurementName        []string `help:"name of Measurement"`
	ResType                string   `help:"Resource properties of measurement e.g. guest/host/redis/oss/rds"`
	MeasurementDisplayName string   `help:"The name of the measurement customization"`
	FieldName              []string `help:"Name of Field"`
	Unit                   string   `help:"Unit of Field " choices:"%|bps|Mbps|Bps|cps|count|ms|byte"`
	FieldDisplayName       string   `help:"The name of the field customization"`
}

func (o *MonitorMetricListOptions) Params() (jsonutils.JSONObject, error) {
	param, err := options.ListStructToParams(&(o.BaseListOptions))
	if err != nil {
		return nil, err
	}
	metricInput := new(monitorapi.MetricListInput)
	metricInput.Measurement.Names = o.MeasurementName
	metricInput.Measurement.ResType = o.ResType
	metricInput.Measurement.DisplayName = o.MeasurementDisplayName
	metricInput.MetricFields.Names = o.FieldName
	metricInput.MetricFields.Unit = o.Unit
	metricInput.MetricFields.DisplayName = o.FieldDisplayName
	listParam := metricInput.JSON(metricInput)
	param.Update(listParam)
	return param, nil
}

type MetricUpdateOptions struct {
	ID                     string `help:"ID of Metric " required:"true" positional:"true"`
	ResType                string `help:"Resource properties of measurement e.g. guest/host/redis/oss/rds" required:"true"`
	MeasurementDisplayName string `help:"The name of the measurement customization" required:"true"`
	FieldName              string `help:"Name of Field" required:"true"`
	FieldDisplayName       string `help:"The name of the field customization" required:"true"`
	Unit                   string `help:"Unit of Field" choices:"%|bps|Mbps|Bps|cps|count|ms|byte" required:"true"`
}

func (o *MetricUpdateOptions) GetId() string {
	return o.ID
}

func (o *MetricUpdateOptions) Params() (jsonutils.JSONObject, error) {
	updateInput := new(monitorapi.MetricUpdateInput)
	updateInput.Measurement.DisplayName = o.MeasurementDisplayName
	updateInput.Measurement.ResType = o.ResType
	updateField := new(monitorapi.MetricFieldUpdateInput)
	updateField.Name = o.FieldName
	updateField.DisplayName = o.FieldDisplayName
	updateField.Unit = o.Unit
	updateInput.MetricFields = []monitorapi.MetricFieldUpdateInput{*updateField}
	updateInput.Scope = "system"
	return updateInput.JSON(updateInput), nil
}

type MetricShowOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *MetricShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *MetricShowOptions) GetId() string {
	return o.ID
}

type MetricDeleteOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *MetricDeleteOptions) GetId() string {
	return o.ID
}

func (o *MetricDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
