package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type MonitorResourceAlertListOptions struct {
	options.BaseListOptions
	MonitorResourceId string `help:"ID of monitor resource" json:"monitor_resource_id"`
	AlertId           string `help:"ID  of alert" json:"alert_id"`
	Alerting          bool   `help:"search alerting resource" json:"alerting"`
	SendState         string `json:"send_state"`
}

func (o *MonitorResourceAlertListOptions) GetMasterOpt() string {
	return o.MonitorResourceId
}

func (o *MonitorResourceAlertListOptions) GetSlaveOpt() string {
	return o.AlertId
}

func (o *MonitorResourceAlertListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}
