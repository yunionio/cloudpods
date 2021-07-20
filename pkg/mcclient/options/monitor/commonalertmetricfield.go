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

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type MonitorMetricFieldListOptions struct {
	options.BaseListOptions
	Names       []string `help:"name of field"`
	Unit        string   `help:"Unit of Field " choices:"%|bps|Mbps|Bps|cps|count|ms|byte"`
	DisplayName string   `help:"The name of the field customization"`
}

func (o *MonitorMetricFieldListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type MetricFieldUpdateOptions struct {
	ID          string `help:"ID of Metric " required:"true" positional:"true"`
	DisplayName string `help:"The name of the field customization" required:"true"`
	Name        string `help:"Name of Field" required:"true"`
	Unit        string `help:"Unit of Field" choices:"%|bps|Mbps|Bps|cps|count|ms|byte" required:"true"`
}

func (o *MetricFieldUpdateOptions) GetId() string {
	return o.ID
}

func (o *MetricFieldUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(o)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type MetricFieldShowOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *MetricFieldShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *MetricFieldShowOptions) GetId() string {
	return o.ID
}

type MetricFieldDeleteOptions struct {
	ID string `help:"ID of Metric " json:"-"`
}

func (o *MetricFieldDeleteOptions) GetId() string {
	return o.ID
}

func (o *MetricFieldDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
