package monitor

import "yunion.io/x/onecloud/pkg/apis"

type AlertRecordListInput struct {
	apis.Meta

	apis.ScopedResourceBaseListInput
	apis.EnabledResourceBaseListInput
	apis.StatusStandaloneResourceListInput

	AlertId string   `json:"alert_id"`
	Level   string   `json:"level"`
	State   string   `json:"state"`
	ResType []string `json:"res_type"`
}

type AlertRecordDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	ResNum    int64  `json:"res_num"`
	AlertName string `json:"alert_name"`
}

type AlertRecordCreateInput struct {
	apis.StandaloneResourceCreateInput

	AlertId string `json:"alert_id"`
	// 报警级别
	Level     string       `json:"level"`
	State     string       `json:"state"`
	ResType   string       `json:"res_type"`
	EvalData  []*EvalMatch `json:"eval_data"`
	AlertRule AlertRecordRule
}

type AlertRecordRule struct {
	Metric          string `json:"metric"`
	Measurement     string `json:"measurement"`
	MeasurementDesc string `json:"measurement_desc"`
	ResType         string `json:"res_type"`
	Field           string `json:"field"`
	FieldDesc       string `json:"field_desc"`
	// 比较运算符, 比如: >, <, >=, <=
	Comparator string `json:"comparator"`
	// 报警阀值
	Threshold     string `json:"threshold"`
	Period        string `json:"period"`
	AlertDuration int64  `json:"alert_duration"`
	ConditionType string `json:"condition_type"`
}
