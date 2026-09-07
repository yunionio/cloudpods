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

type ResourceMetricsOptions struct {
	RES_TYPE  string   `help:"resource type: host or guest" choices:"host|guest"`
	ResIds    []string `help:"resource IDs" json:"res_ids" nargs:"+"`
	StartTime string   `help:"start time (RFC3339). e.g.: 2023-12-06T21:54:42Z" json:"-"`
	EndTime   string   `help:"end time (RFC3339). e.g.: 2023-12-18T21:54:42Z" json:"-"`
	Interval  string   `help:"query interval. e.g.: 5m, 1h" json:"interval"`
}

func (o *ResourceMetricsOptions) GetInput() (*api.ResourceMetricsQueryInput, error) {
	input := &api.ResourceMetricsQueryInput{
		ResType:  o.RES_TYPE,
		ResIds:   o.ResIds,
		Interval: o.Interval,
	}
	if o.StartTime != "" {
		t, err := time.Parse(time.RFC3339, o.StartTime)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid start_time %q", o.StartTime)
		}
		input.StartTime = t
	}
	if o.EndTime != "" {
		t, err := time.Parse(time.RFC3339, o.EndTime)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid end_time %q", o.EndTime)
		}
		input.EndTime = t
	}
	return input, nil
}

type MeasurementsQueryOptions struct {
	Scope           string `json:"scope" mcp:"true"`
	ProjectDomainId string `json:"project_domin_id" mcp:"true"`
	ProjectId       string `json:"project_id" mcp:"true"`
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
	_ struct{} `mcp-desc:"查询监控时序。必填 MEASUREMENT+FIELD。资源监控：vm_cpu/usage_active。账户余额：database=meter_db measurement=cloudaccount_balance field=balance from=720h to=now interval=24h。告警趋势：measurement=alert_record_history field=res_num func=sum group-by=res_type from=720h。from 可为 RFC3339 或 720h。不要用 climc_server_monitor"`

	MeasurementsQueryOptions

	MEASUREMENT string `help:"metric measurement. e.g.: cpu, vm_cpu, vm_mem, disk, cloudaccount_balance, alert_record_history"`
	FIELD       string `help:"metric field. e.g.: usage_active, balance, res_num"`

	Database        string   `help:"influx database. 默认 monitor；账户余额用 meter_db" mcp:"true"`
	Interval        string   `help:"metric interval. e.g.: 5m, 1h, 24h" mcp:"true"`
	From            string   `help:"start: RFC3339 或相对时长 720h" mcp:"true"`
	To              string   `help:"end: RFC3339 或 now" mcp:"true"`
	Tags            []string `help:"filter tags. e.g.: vm_name=vm1" mcp:"true"`
	GroupBy         []string `help:"group by tag，如 res_type" mcp:"true"`
	Func            string   `help:"select aggregation: mean|sum|max|min" mcp:"true"`
	UseMean         bool     `help:"calcuate mean result for field（同 func=mean）" mcp:"true"`
	SkipCheckSeries bool     `help:"skip checking series: not fetch extra tags from region service" mcp:"true"`
	Reducer         string   `help:"series result reducer. e.g.: sum, percentile(95)" mcp:"true"`
}

func (o MetricQueryOptions) GetQueryInput() (*api.MetricQueryInput, error) {
	input := monitor.NewMetricQueryInputWithDB(o.Database, o.MEASUREMENT)
	input.Interval(o.Interval)
	if o.SkipCheckSeries {
		input.SkipCheckSeries(true)
	}
	input.Scope(o.Scope)

	if o.From != "" {
		if fromTime, err := time.Parse(time.RFC3339, o.From); err == nil {
			input.From(fromTime)
		} else {
			input.FromRaw(o.From)
		}
	}
	if o.To != "" {
		if strings.EqualFold(o.To, "now") {
			input.ToRaw("now")
		} else if toTime, err := time.Parse(time.RFC3339, o.To); err == nil {
			input.To(toTime)
		} else {
			input.ToRaw(o.To)
		}
	}

	sel := input.Selects().Select(o.FIELD)
	switch strings.ToLower(strings.TrimSpace(o.Func)) {
	case "sum":
		sel.SUM()
	case "max":
		sel.MAX()
	case "min":
		sel.MIN()
	case "mean", "avg":
		sel.MEAN()
	default:
		if o.UseMean {
			sel.MEAN()
		}
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
