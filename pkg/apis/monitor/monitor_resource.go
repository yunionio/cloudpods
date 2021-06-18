package monitor

import (
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
)

const (
	MONITOR_RESOURCE_ALERT_STATUS_INIT     = "init"
	MONITOR_RESOURCE_ALERT_STATUS_ATTACH   = "attach"
	MONITOR_RESOURCE_ALERT_STATUS_ALERTING = "alerting"
)

type MonitorResourceCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	ResId       string `json:"res_id"`
	ResType     string `json:"res_type"`
	AlertStatus string `json:"alert_status"`
}

type MonitorResourceListInput struct {
	apis.VirtualResourceListInput
	apis.EnabledResourceBaseListInput
	compute.ManagedResourceListInput

	ResId     []string `json:"res_id"`
	ResType   string   `json:"res_type"`
	OnlyResId bool     `json:"only_res_id"`
}

type MonitorResourceDetails struct {
	apis.VirtualResourceDetails
	compute.CloudregionResourceInfo

	AttachAlertCount int64 `json:"attach_alert_count"`
}
