package monitor

import "yunion.io/x/onecloud/pkg/apis"

type AlertPanelCreateInput struct {
	apis.ScopedResourceInput

	CommonMetricInputQuery
	AlertCreateInput

	Refresh     string `json:"refresh"`
	DashboardId string `json:"dashboard_id"`
}

type AlertPanelListInput struct {
	AlertListInput
	DashboardId string `json:"dashboard_id"`
}

type AlertPanelUpdateInput struct {
	apis.ScopedResourceBaseInfo
	CommonMetricInputQuery
	V1AlertUpdateInput

	Refresh string `json:"refresh"`
}

type PanelDetails struct {
	AlertDetails
	CommonAlertMetricDetails []*CommonAlertMetricDetails `json:"common_alert_metric_details"`
}
