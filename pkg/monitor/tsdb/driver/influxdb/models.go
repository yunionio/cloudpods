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

package influxdb

import (
	"time"

	api "yunion.io/x/onecloud/pkg/apis/monitor"
)

type Query struct {
	Measurement  string
	Policy       string
	ResultFormat string
	Tags         []api.MetricQueryTag
	GroupBy      []*QueryPart
	Selects      []*Select
	Alias        string
	Tz           string
	Interval     time.Duration
}

type Select []QueryPart

type Response struct {
	Results []Result `json:"results"`
	Err     error    `json:"err"`
}

type Result struct {
	Series  []Row      `json:"series"`
	Message []*Message `json:"message"`
	Err     error      `json:"err"`
}

type Message struct {
	Level string `json:"level,omitempty"`
	Text  string `json:"text,omitempty"`
}

type Row struct {
	Name    string            `json:"name,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`
	Columns []string          `json:"columns,omitempty"`
	Values  [][]interface{}   `json:"values,omitempty"`
}
