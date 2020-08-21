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
	Level    string       `json:"level"`
	State    string       `json:"state"`
	EvalData []*EvalMatch `json:"eval_data"`
}
