package monitor

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type SuggestSysAlertListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput

	//监控规则type：Rule Type
	Type  string `json:"type"`
	ResId string `json:"res_id"`
}

type SuggestSysAlertCreateInput struct {
	apis.VirtualResourceCreateInput

	Enabled       *bool `json:"enabled"`
	MonitorConfig jsonutils.JSONObject

	//转换成ResId
	ResID   string               `json:"res_id"`
	Type    string               `json:"type"`
	Problem jsonutils.JSONObject `json:"problem"`
	Suggest string               `json:"suggest"`
	Action  string

	RuleAt time.Time
}

type SuggestSysAlertDetails struct {
	apis.VirtualResourceDetails

	//监控规则对应的json对象
	MonitorConfig jsonutils.JSONObject `json:"monitor_config"`
	//监控规则type：Rule Type
	Type    string               `json:"type"`
	Problem jsonutils.JSONObject `json:"problem"`
	Suggest string               `json:"suggest"`
	Action  string               `json:"action"`
	//Description string `width:"256" charset:"ascii" list:"user"`
	ResId string `json:"res_id"`
	// 根据规则定位时间
	RuleAt time.Time `json:"rule_at"`
}

type SuggestSysAlertUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	Enabled       *bool `json:"enabled"`
	MonitorConfig jsonutils.JSONObject

	//转换成ResId
	ResID   string               `json:"res_id"`
	Type    string               `json:"type"`
	Problem jsonutils.JSONObject `json:"problem"`
	Suggest string               `json:"suggest"`
	Action  string

	RuleAt time.Time
}
