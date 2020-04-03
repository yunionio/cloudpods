package monitor

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type SuggestSysRuleListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput
}

type SuggestSysRuleCreateInput struct {
	apis.VirtualResourceCreateInput

	// 查询指标周期
	Period  string               `json:"period"`
	Enabled *bool                `json:"enabled"`
	Setting jsonutils.JSONObject `json:"setting"`
}

type SuggestSysRuleUpdateInput struct {
	apis.Meta

	// 查询指标周期
	Period   string               `json:"period"`
	Setting  jsonutils.JSONObject `json:"setting"`
	Enabled  *bool                `json:"enabled"`
	ExecTime time.Time            `json:"exec_time"`
}

type SuggestSysRuleDetails struct {
	apis.VirtualResourceDetails

	Name    string               `json:"name"`
	Setting jsonutils.JSONObject `json:"setting"`
	Enabled bool                 `json:"enabled"`
}
