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
)

type SimpleQueryOptions struct {
	Id         string   `json:"id"`
	Database   string   `json:"database"`
	MetricName string   `json:"metric_name"`
	StartTime  string   `json:"start_time" help:"e.g.: 2023-12-06T21:54:42.123Z"`
	EndTime    string   `json:"end_time" help:"e.g.: 2023-12-18T21:54:42.123Z"`
	Tags       []string `json:"tags"`
}

func (o *SimpleQueryOptions) GetId() string {
	return "simple-query"
}

func (o *SimpleQueryOptions) Params() (jsonutils.JSONObject, error) {
	ret := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	ret.Remove("tags")
	tags := map[string]string{}
	for _, tag := range o.Tags {
		if strings.Contains(tag, "=") {
			info := strings.Split(tag, "=")
			if len(info) == 2 {
				tags[info[0]] = info[1]
			}
		}
	}
	if len(tags) > 0 {
		ret.Set("tag_pairs", jsonutils.Marshal(tags))
	}
	return ret, nil
}
