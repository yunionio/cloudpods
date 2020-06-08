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

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

type InfluxdbShemaListOptions struct {
}

type InfluxdbShemaShowOptions struct {
	ID          string `help:"attribute of the inluxdb" choices:"databases|measurements|metric-measurement"`
	Database    string `help:"influxdb database"`
	Measurement string `help:"influxdb table"`
}

func (opt InfluxdbShemaShowOptions) Params() (jsonutils.JSONObject, error) {
	input := new(monitor.InfluxMeasurement)
	input.Measurement = opt.Measurement
	input.Database = opt.Database
	return input.JSON(input), nil
}
