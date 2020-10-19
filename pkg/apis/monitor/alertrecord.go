package monitor

import "yunion.io/x/onecloud/pkg/apis"

type AlertRecordListInput struct {
	apis.Meta

	apis.ScopedResourceBaseListInput
	apis.EnabledResourceBaseListInput
	apis.StatusStandaloneResourceListInput

	AlertId string `json:"alert_id"`
	Level   string `json:"level"`
	State   string `json:"state"`
}

type AlertRecordDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	ResNum int64 `json:"res_num"`
}

type AlertRecordCreateInput struct {
	apis.StandaloneResourceCreateInput

	AlertId string `json:"alert_id"`
	// 报警级别
	Level     string       `json:"level"`
	State     string       `json:"state"`
	EvalData  []*EvalMatch `json:"eval_data"`
	AlertRule AlertRecordRule
}

type AlertRecordRule struct {
	Metric          string `json:"metric"`
	MeasurementDesc string `json:"measurement_desc"`
	FieldDesc       string `json:"field_desc"`
	// 比较运算符, 比如: >, <, >=, <=
	Comparator string `json:"comparator"`
	// 报警阀值
	Threshold string `json:"threshold"`
	Period    string `json:"period"`
}
