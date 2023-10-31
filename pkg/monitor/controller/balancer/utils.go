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

package balancer

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

type HostMetric struct {
	Id     string
	Values map[string]float64
}

type HostMetrics struct {
	metrics []*HostMetric
	indexes map[string]*HostMetric
}

func NewHostMetrics(ms []*HostMetric) *HostMetrics {
	h := &HostMetrics{
		metrics: ms,
		indexes: make(map[string]*HostMetric),
	}
	for _, m := range ms {
		h.indexes[m.Id] = m
	}
	return h
}

func (hs HostMetrics) JSONString() string {
	return jsonutils.Marshal(hs.metrics).String()
}

func (hs HostMetrics) Get(id string) *HostMetric {
	return hs.indexes[id]
}

type TsdbQuery struct {
	Database    string
	Measurement string
	Fields      []string
}

func InfluxdbQuery(
	ds *tsdb.DataSource,
	idKey string,
	hosts []IResource,
	query *TsdbQuery) (*HostMetrics, error) {
	q := monitor.NewAlertQuery(query.Database, query.Measurement).From("5m").To("now")
	sels := q.Selects()
	for _, field := range query.Fields {
		sels.Select(field).MEAN().AS(field)
	}
	ids := []string{}
	for _, h := range hosts {
		ids = append(ids, h.GetId())
	}
	q.Where().IN(idKey, ids)
	q.GroupBy().TAG(idKey).FILL_NULL()
	qCtx := q.ToTsdbQuery()

	resp, err := tsdb.HandleRequest(context.Background(), ds, qCtx)
	if err != nil {
		return nil, errors.Wrap(err, "TSDB endpoint Query")
	}

	ss := resp.Results[""].Series
	ms := make([]*HostMetric, len(ss))
	for i, s := range ss {
		m := &HostMetric{
			Id:     s.Tags[idKey],
			Values: make(map[string]float64),
		}
		for j, f := range query.Fields {
			m.Values[f] = *(s.Points[0][j].(*float64))
		}
		ms[i] = m
	}
	return NewHostMetrics(ms), nil
}

func GetAlertSettingThreshold(s *api.AlertSetting) (float64, error) {
	if len(s.Conditions) != 1 {
		return 0, errors.Errorf("AlertSetting conditions %d != 1", len(s.Conditions))
	}
	return s.Conditions[0].Evaluator.Params[0], nil
}
