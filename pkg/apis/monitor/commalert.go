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

const (
	ALERT_STATUS_READY       = "ready"
	ALERT_STATUS_DELETE      = "start_delete"
	ALERT_STATUS_DELETE_FAIL = "delete_fail"
	ALERT_STATUS_DELETING    = "deleting"
	ALERT_STATUS_DELETED     = "deleted"

	CommonAlertSystemAlertType = "system"
	CommonAlertNomalAlertType  = "normal"

	MonitorComponentType = "default-monitor"
	MonitorComponentPort = 30093
	SubscribAPI          = "subscriptions"
	MonitorDefaultRC     = "30day_only"
	MonitorSubName       = "commonalert"
	MonitorSubDataBase   = "telegraf"

	CommonAlertDefaultRecipient = "commonalert-default"

	//metirc fields 之间的运算
	CommonAlertFieldOpt_Division = "/"

	DEFAULT_SEND_NOTIFY_CHANNEL = "users"

	METRIC_QUERY_TYPE_NO_DATA     = "nodata_query"
	METRIC_QUERY_NO_DATA_THESHOLD = "nodata"
)

var CommonAlertLevels = []string{"normal", "important", "fatal"}

type CommonAlertCreateInput struct {
	CommonMetricInputQuery
	AlertCreateInput

	// 查询指标周期
	Period string `json:"period"`
	// 报警连续持续周期数
	AlertDuration int64 `json:"alert_duration"`
	// 通知方式, 比如: email, mobile
	Channel []string `json:"channel"`
	// 通知接受者
	Recipients []string `json:"recipients"`
	// 静默期
	SilentPeriod string `json:"silent_period"`
	// 报警类型
	AlertType string `json:"alert_type"`

	//scope Resource
	Scope       string `json:"scope"`
	DomainId    string `json:"domain_id"`
	ProjectId   string `json:"project_id"`
	GetPointStr bool   `json:"get_point_str"`
	MetaName    string `json:"meta_name"`
	Description string `json:"description"`
}

type CommonMetricInputQuery struct {
	From        string              `json:"from"`
	To          string              `json:"to"`
	Interval    string              `json:"interval"`
	MetricQuery []*CommonAlertQuery `json:"metric_query"`
}

type CommonAlertQuery struct {
	*AlertQuery
	// metric points'value的运算方式
	Reduce string `json:"reduce"`
	// 比较运算符, 比如: >, <, >=, <=
	Comparator string `json:"comparator"`
	// 报警阀值
	Threshold float64 `json:"threshold"`
	//field yunsuan
	FieldOpt      string `json:"field_opt"`
	ConditionType string `json:"condition_type"`
}

type CommonAlertListInput struct {
	AlertListInput
	//V1AlertListInput
	// 报警类型
	AlertType string `json:"alert_type"`
	// 监控指标名称
	Metric string `json:"metric"`

	Level   string   `json:"level"`
	ResType []string `json:"res_type"`
}

type CommonAlertUpdateInput struct {
	CommonMetricInputQuery
	V1AlertUpdateInput

	// 查询指标周期
	Period string `json:"period"`
	// 报警连续持续周期数
	AlertDuration int64 `json:"alert_duration"`
	// 通知方式, 比如: email, mobile
	Channel []string `json:"channel"`
	// 通知接受者
	Recipients []string `json:"recipients"`
	// 静默期
	SilentPeriod string `json:"silent_period"`
	// systemalert policy may need update through operator
	ForceUpdate bool   `json:"force_update"`
	GetPointStr bool   `json:"get_point_str"`
	MetaName    string `json:"meta_name"`
}

type CommonAlertDetails struct {
	AlertDetails
	Period string `json:"period"`
	// 报警连续持续周期数
	AlertDuration int64    `json:"alert_duration"`
	Level         string   `json:"level"`
	NotifierId    string   `json:"notifier_id"`
	Channel       []string `json:"channel"`
	Recipients    []string `json:"recipients"`
	// 静默期
	SilentPeriod string `json:"silent_period"`
	Status       string `json:"status"`
	// 报警类型
	AlertType                string                      `json:"alert_type"`
	CommonAlertMetricDetails []*CommonAlertMetricDetails `json:"common_alert_metric_details"`
}

type CommonAlertMetricDetails struct {
	Comparator    string    `json:"comparator"`
	Threshold     float64   `json:"threshold"`
	WithinRange   []float64 `json:"within_range"`
	ConditionType string    `json:"condition_type"`
	ThresholdStr  string    `json:"threshold_str"`
	// metric points'value的运算方式
	Reduce                 string           `json:"reduce"`
	DB                     string           `json:"db"`
	Measurement            string           `json:"measurement"`
	MeasurementDisplayName string           `json:"measurement_display_name"`
	ResType                string           `json:"res_type"`
	Field                  string           `json:"field"`
	Groupby                string           `json:"groupby"`
	Filters                []MetricQueryTag `json:"filters"`
	FieldDescription       MetricFieldDetail
	FieldOpt               string `json:"field_opt"`
	GetPointStr            bool   `json:"get_point_str"`
}
