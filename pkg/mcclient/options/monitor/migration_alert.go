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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type MigrationAlertListOptions struct {
	options.BaseListOptions
	MetricType string `help:"Migration alert metric type" choices:"cpu.usage_active|mem.available"`
}

func (o *MigrationAlertListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type MigrationAlertShowOptions struct {
	ID string `help:"ID of alart " json:"-"`
}

func (o *MigrationAlertShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

func (o *MigrationAlertShowOptions) GetId() string {
	return o.ID
}

type MigrationAlertCreateOptions struct {
	NAME        string   `help:"Name of the migration alert"`
	METRIC      string   `help:"Metric type" choices:"cpu.usage_active.gt|mem.available.lt"`
	THRESHOLD   float64  `help:"Metric threshold"`
	Period      string   `help:"Period of execution, e.g. '5m', '1h'" default:"5m"`
	SourceHost  []string `help:"Source hosts' id or name"`
	SourceGuest []string `help:"Source guests's id or name"`
	TargetHost  []string `help:"Target hosts' id or name"`
}

func (o *MigrationAlertCreateOptions) parseMetric(m string) (monitor.MigrationAlertMetricType, error) {
	parts := strings.Split(m, ".")
	if len(parts) != 3 {
		return "", errors.Errorf("Invalid metric %q", m)
	}
	return monitor.MigrationAlertMetricType(strings.Join([]string{parts[0], parts[1]}, ".")), nil
}

func (o *MigrationAlertCreateOptions) Params() (jsonutils.JSONObject, error) {
	input := new(monitor.MigrationAlertCreateInput)
	input.Name = o.NAME
	input.Threshold = o.THRESHOLD
	input.Period = o.Period
	mt, err := o.parseMetric(o.METRIC)
	if err != nil {
		return nil, errors.Wrap(err, "metric_type")
	}
	input.MetricType = mt

	input.MigrationSettings = &monitor.MigrationAlertSettings{
		Source: &monitor.MigrationAlertSettingsSource{},
		Target: &monitor.MigrationAlertSettingsTarget{},
	}
	if len(o.SourceHost) != 0 {
		input.MigrationSettings.Source.HostIds = o.SourceHost
	}
	if len(o.SourceGuest) != 0 {
		input.MigrationSettings.Source.GuestIds = o.SourceGuest
	}
	if len(o.TargetHost) != 0 {
		input.MigrationSettings.Target.HostIds = o.TargetHost
	}

	return input.JSON(input), nil
}
