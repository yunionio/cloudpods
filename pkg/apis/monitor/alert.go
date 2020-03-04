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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
)

type AlertStateType string
type AlertSeverityType string
type NoDataOption string
type ExecutionErrorOption string

const (
	AlertStateNoData   AlertStateType = "no_data"
	AlertStatePaused   AlertStateType = "paused"
	AlertStateAlerting AlertStateType = "alerting"
	AlertStateOK       AlertStateType = "ok"
	AlertStatePending  AlertStateType = "pending"
	AlertStateUnknown  AlertStateType = "unknown"
)

const (
	NoDataSetOK       NoDataOption = "ok"
	NoDataSetNoData   NoDataOption = "no_data"
	NoDataKeepState   NoDataOption = "keep_state"
	NoDataSetAlerting NoDataOption = "alerting"
)

const (
	ExecutionErrorSetAlerting ExecutionErrorOption = "alerting"
	ExecutionErrorKeepState   ExecutionErrorOption = "keep_state"
)

var (
	ErrCannotChangeStateOnPausedAlert = errors.Error("Cannot change state on pause alert")
	ErrRequiresNewState               = errors.Error("update alert state requires a new state")
)

func (s AlertStateType) IsValid() bool {
	return s == AlertStateOK ||
		s == AlertStateNoData ||
		s == AlertStatePaused ||
		s == AlertStatePending ||
		s == AlertStateAlerting ||
		s == AlertStateUnknown
}

func (s NoDataOption) IsValid() bool {
	return s == NoDataSetNoData || s == NoDataSetAlerting || s == NoDataKeepState || s == NoDataSetOK
}

func (s NoDataOption) ToAlertState() AlertStateType {
	return AlertStateType(s)
}

func (s ExecutionErrorOption) IsValid() bool {
	return s == ExecutionErrorSetAlerting || s == ExecutionErrorKeepState
}

func (s ExecutionErrorOption) ToAlertState() AlertStateType {
	return AlertStateType(s)
}

// AlertSettings contains alert conditions
type AlertSetting struct {
	Conditions []AlertCondition `json:"conditions"`
}

type AlertCondition struct {
	Type      string     `json:"type"`
	Query     AlertQuery `json:"query"`
	Reducer   Condition  `json:"reducer"`
	Evaluator Condition  `json:"evaluator"`
	Operator  string     `json:"operator"`
}

type AlertQuery struct {
	Model        MetricQuery `json:"model"`
	DataSourceId string      `json:"data_source_id"`
	From         string      `json:"from"`
	To           string      `json:"to"`
}

type AlertCreateInput struct {
	apis.Meta

	// 报警名称
	Name string `json:"name"`
	// 报警执行频率
	Frequency int64 `json:"frequency"`
	// 报警设置
	Settings AlertSetting `json:"settings"`
	// 启用报警
	Enabled *bool `json:"enabled"`
	// 报警级别
	Level string `json:"level"`
}

type AlertUpdateInput struct {
	apis.Meta

	// 报警名称
	Name *string `json:"name"`
	// 报警执行频率
	Frequency *int64 `json:"frequency"`
	// 报警设置
	Settings *AlertSetting `json:"settings"`
	// 启用报警
	Enabled *bool `json:"enabled"`
	// 报警级别
	Level *string `json:"level"`
}

type AlertListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	// 以报警是否启用/禁用过滤列表
	// Enabled *bool `json:"enabled"`
}
