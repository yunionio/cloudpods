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
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

type InfluxdbQueryParser struct{}

func (qp *InfluxdbQueryParser) Parse(model *tsdb.Query, dsInfo *tsdb.DataSource) (*Query, error) {
	policy := "default"
	if model.Policy != "" {
		policy = model.Policy
	}
	alias := model.Alias
	tz := model.Tz
	measurement := model.Measurement
	resultFormat := model.ResultFormat

	tags := model.Tags
	groupBys, err := qp.parseGroupBy(model.GroupBy)
	if err != nil {
		return nil, err
	}

	selects, err := qp.parseSelects(model.Selects)
	if err != nil {
		return nil, err
	}

	parsedInterval, err := tsdb.GetIntervalFrom(dsInfo, model, time.Millisecond*1)
	if err != nil {
		return nil, err
	}
	return &Query{
		Measurement:  measurement,
		Policy:       policy,
		ResultFormat: resultFormat,
		GroupBy:      groupBys,
		Tags:         tags,
		Selects:      selects,
		Interval:     parsedInterval,
		Alias:        alias,
		Tz:           tz,
	}, nil
}

func (qp *InfluxdbQueryParser) parseSelects(selects []api.MetricQuerySelect) ([]*Select, error) {
	var result []*Select

	for _, selectObj := range selects {
		var parts Select
		for _, part := range selectObj {
			queryPart, err := qp.parseQueryPart(part)
			if err != nil {
				return nil, err
			}
			parts = append(parts, *queryPart)
		}
		result = append(result, &parts)
	}
	return result, nil
}

func (qp *InfluxdbQueryParser) parseGroupBy(groupBy []api.MetricQueryPart) ([]*QueryPart, error) {
	var result []*QueryPart

	for _, gb := range groupBy {
		queryPart, err := qp.parseQueryPart(gb)
		if err != nil {
			return nil, err
		}
		result = append(result, queryPart)
	}

	return result, nil
}

func (qp *InfluxdbQueryParser) parseQueryPart(part api.MetricQueryPart) (*QueryPart, error) {
	return NewQueryPart(part.Type, part.Params)
}
