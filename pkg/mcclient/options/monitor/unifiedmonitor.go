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
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
)

type MeasurementsQueryOptions struct {
	Scope           string `json:"scope"`
	ProjectDomainId string `json:"project_domin_id"`
	ProjectId       string `json:"project_id"`
}

func (o MeasurementsQueryOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func (o MeasurementsQueryOptions) Property() string {
	return "measurements"
}

type DatabasesQueryOptions struct{}

func (o DatabasesQueryOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

func (o DatabasesQueryOptions) Property() string {
	return "databases"
}

type MetricQueryOptions struct {
	MeasurementsQueryOptions

	MEASUREMENT string `help:"metric measurement. e.g.: cpu, vm_cpu, vm_mem, disk..."`
	FIELD       string `help:"metric field. e.g.: usage_active, free..."`

	Interval        string   `help:"metric interval. e.g.: 5m, 1h"`
	From            string   `help:"start time(RFC3339 format). e.g.: 2023-12-06T21:54:42.123Z"`
	To              string   `help:"end time(RFC3339 format). e.g.: 2023-12-18T21:54:42.123Z"`
	Tags            []string `help:"filter tags. e.g.: vm_name=vm1"`
	GroupBy         []string `help:"group by tag"`
	UseMean         bool     `help:"calcuate mean result for field"`
	SkipCheckSeries bool     `help:"skip checking series: not fetch extra tags from region service"`
	Reducer         string   `help:"series result reducer. e.g.: sum, percentile(95)"`
}

func (o MetricQueryOptions) GetQueryInput() (*api.MetricQueryInput, error) {
	input := monitor.NewMetricQueryInput(o.MEASUREMENT)
	input.Interval(o.Interval)
	if o.SkipCheckSeries {
		input.SkipCheckSeries(true)
	}
	input.Scope(o.Scope)

	// parse time
	if o.From != "" {
		fromTime, err := time.Parse(time.RFC3339, o.From)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid from time: %q", o.From)
		}
		input.From(fromTime)
	}
	if o.To != "" {
		toTime, err := time.Parse(time.RFC3339, o.To)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid to time: %q", o.To)
		}
		input.To(toTime)
	}

	sel := input.Selects().Select(o.FIELD)
	if o.UseMean {
		sel.MEAN()
	}

	where := input.Where()
	for _, tag := range o.Tags {
		if strings.Contains(tag, "=") {
			info := strings.Split(tag, "=")
			if len(info) == 2 {
				where.Equal(info[0], info[1])
			} else {
				return nil, errors.Errorf("invalid tag: %q, len: %d", tag, len(info))
			}
		} else {
			return nil, errors.Errorf("invalid tag: %q", tag)
		}
	}

	groupBy := input.GroupBy()
	for _, tag := range o.GroupBy {
		groupBy.TAG(tag)
	}

	if o.Reducer != "" {
		r, err := o.parseReducer(o.Reducer)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid reducer: %q", o.Reducer)
		}
		input.Reducer(r.Type, r.Params)
	}

	return input.ToQueryData(), nil
}

func (o MetricQueryOptions) parseReducer(reducer string) (*api.Condition, error) {
	if reducer == "" {
		return nil, errors.Errorf("invalid reducer %q", reducer)
	}
	parts := strings.Split(reducer, "(")
	if len(parts) < 1 {
		return nil, errors.Errorf("invalid reducer %q", reducer)
	}
	rType := parts[0]
	cond := &api.Condition{
		Type: rType,
	}
	if len(parts) > 1 {
		params := []float64{}
		paramStr := parts[1]
		paramsStr := strings.Split(strings.TrimSuffix(paramStr, ")"), ",")
		for _, param := range paramsStr {
			f, err := strconv.ParseFloat(strings.ReplaceAll(param, " ", ""), 64)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid reducer param %q", param)
			}
			params = append(params, f)
		}
		cond.Params = params
	}
	return cond, nil
}
