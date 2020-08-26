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

package zstack

import (
	"time"

	"yunion.io/x/jsonutils"
)

type SDataPoint struct {
	DataPoints []DataPoint `json:"data"`
}

type DataPoint struct {
	Value     float64 `json:"value"`
	TimeStemp int64   `json:"time"`
	Labels    *Label  `json:"labels"`
}

type Label struct {
	VMUuid   string `json:"VMUuid"`
	HostUuid string `json:"HostUuid"`
}

func (region *SRegion) GetMonitorData(name string, namespace string, since time.Time,
	until time.Time) (*SDataPoint, error) {
	datas := SDataPoint{}
	param := jsonutils.NewDict()
	param.Add(jsonutils.NewString(namespace), "namespace")
	param.Add(jsonutils.NewString(name), "metricName")
	param.Add(jsonutils.NewString("60"), "period")
	param.Add(jsonutils.NewInt(since.Unix()), "startTime")
	param.Add(jsonutils.NewInt(until.Unix()), "endTime")
	rep, err := region.client.getMonitor("zwatch/metrics", param)
	if err != nil {
		return nil, err
	}
	rep.Unmarshal(&datas)
	return &datas, nil
}
