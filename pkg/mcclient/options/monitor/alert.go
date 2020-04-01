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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	monitor2 "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertListOptions struct {
	options.BaseListOptions
}

type AlertShowOptions struct {
	ID string `help:"ID or name of the alert" json:"-"`
}

type AlertDeleteOptions struct {
	ID []string `help:"ID of alert to delete"`
}

type AlertTestRunOptions struct {
	ID    string `help:"ID of alert to delete"`
	Debug bool   `help:"Show more debug info"`
}

type AlertPauseOptions struct {
	ID      string `help:"ID of alert to delete"`
	UnPause bool   `help:"Unpause alert"`
}

type AlertConditionOptions struct {
	REDUCER    string   `help:"Metric query reducer, e.g. 'avg'" choices:"avg|sum|min|max|count|last|median"`
	DATABASE   string   `help:"Metric database, e.g. 'telegraf'"`
	METRIC     string   `help:"Query metric format <measurement>.<field>, e.g. 'cpu.cpu_usage'"`
	COMPARATOR string   `help:"Evaluator compare" choices:"gt|lt"`
	THRESHOLD  float64  `help:"Alert threshold"`
	Period     string   `help:"Query metric period e.g. '5m', '1h'" default:"5m"`
	Tag        []string `help:"Query tag, e.g. 'zone=zon0,name=vmname'"`
	For        string   `help:"For time duration"`
}

func (opt AlertConditionOptions) Params(conf *monitor2.AlertConfig) (*monitor2.AlertCondition, error) {
	parts := strings.Split(opt.METRIC, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("metric %s is invalid format", opt.METRIC)
	}
	cond := conf.Condition(opt.DATABASE, parts[0])
	if opt.COMPARATOR == "gt" {
		cond.GT(opt.THRESHOLD)
	}
	if opt.COMPARATOR == "lt" {
		cond.LT(opt.THRESHOLD)
	}
	switch opt.REDUCER {
	case "avg":
		cond.Avg()
	case "sum":
		cond.Sum()
	case "min":
		cond.Min()
	case "max":
		cond.Max()
	case "count":
		cond.Count()
	case "last":
		cond.Last()
	case "median":
		cond.Median()
	}

	q := cond.Query().From(opt.Period)
	q.Selects().Select(parts[1])

	for _, tag := range opt.Tag {
		parts := strings.Split(tag, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag format: %s", tag)
		}
		q.Where().Equal(parts[0], parts[1])
	}

	return cond, nil
}

type AlertStatesOptions struct {
	NoDataState         string `help:"Set state when no data"`
	ExecutionErrorState string `help:"Set state when execution error"`
}

type AlertCreateOptions struct {
	AlertConditionOptions
	AlertStatesOptions
	NAME      string `help:"Name of the alert"`
	Frequency string `help:"Alert execute frequency, e.g. '5m', '1h'"`
	Enabled   bool   `help:"Enable alert"`
	Level     string `help:"Alert level"`
}

func (opt AlertCreateOptions) Params() (*monitor2.AlertConfig, error) {
	input, err := monitor2.NewAlertConfig(opt.NAME, opt.Frequency, opt.Enabled)
	if err != nil {
		return nil, err
	}
	_, err = opt.AlertConditionOptions.Params(input)
	if err != nil {
		return nil, err
	}
	input.NoDataState(opt.NoDataState)
	input.ExecutionErrorState(opt.ExecutionErrorState)
	return input, nil
}

type AlertUpdateOptions struct {
	ID        string `help:"ID or name of the alert"`
	Name      string `help:"Update alert name"`
	Frequency string `help:"Alert execute frequency, e.g. '5m', '1h'"`
	AlertStatesOptions
}

func (opt AlertUpdateOptions) Params() (*monitor.AlertUpdateInput, error) {
	input := new(monitor.AlertUpdateInput)
	if opt.Name != "" {
		input.Name = &opt.Name
	}
	if opt.Frequency != "" {
		freq, err := time.ParseDuration(opt.Frequency)
		if err != nil {
			return nil, fmt.Errorf("Invalid frequency time format %s: %v", opt.Frequency, err)
		}
		f := int64(freq / time.Second)
		input.Frequency = &f
	}
	input.NoDataState = opt.NoDataState
	input.ExecutionErrorState = opt.ExecutionErrorState
	return input, nil
}

type AlertNotificationAttachOptions struct {
	ALERT        string `help:"ID or name of alert"`
	NOTIFICATION string `help:"ID or name of alert notification"`
	UsedBy       string `help:"UsedBy annotation"`
}

type AlertNotificationListOptions struct {
	options.BaseListOptions
	Alert        string `help:"ID or name of alert" short-token:"a"`
	Notification string `help:"ID or name of notification" short-token:"n"`
}

func (o AlertNotificationListOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	return params, nil
}
