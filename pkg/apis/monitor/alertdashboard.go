package monitor

import "yunion.io/x/onecloud/pkg/apis"

type AlertDashBoardCreateInput struct {
	apis.ScopedResourceInput

	CommonMetricInputQuery
	AlertCreateInput

	Refresh string `json:"refresh"`
}

type AlertDashBoardListInput struct {
	AlertListInput
}

type AlertDashBoardDetails struct {
	AlertDetails

	CommonAlertMetricDetails []*CommonAlertMetricDetails `json:"common_alert_metric_details"`
}

type AlertDashBoardUpdateInput struct {
	CommonMetricInputQuery
	V1AlertUpdateInput

	Refresh string `json:"refresh"`
}
