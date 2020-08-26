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

package google

import (
	"fmt"
	"strconv"
	"time"

	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"yunion.io/x/jsonutils"
)

func (region *SRegion) GetMonitorData(id, serverName, metricName string, since time.Time,
	until time.Time) (*jsonutils.JSONArray, error) {
	params := map[string]string{
		"filter": `metric.type="` + metricName + `" AND metric.labels.instance_name="` + serverName + `"`,
		//"filter":             "metric.type=" + metricName + " AND metric.labels.instance_name=" + serverName,
		"interval.startTime": since.Format(time.RFC3339),
		"interval.endTime":   until.Format(time.RFC3339),
		"view":               strconv.FormatInt(int64(monitoringpb.ListTimeSeriesRequest_FULL), 10),
	}
	resource := fmt.Sprintf("%s/%s/%s", "projects", id, "timeSeries")

	timeSeries, err := region.client.monitorListAll(resource, params)
	if err != nil {
		return nil, err
	}
	return timeSeries, nil
}
