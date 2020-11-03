package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type AlertDashBoardCreateInput struct {
	apis.Meta
	apis.ScopedResourceCreateInput
	apis.StandaloneResourceCreateInput

	Refresh string `json:"refresh"`
}

type AlertDashBoardListInput struct {
	AlertListInput
}

type AlertDashBoardDetails struct {
	AlertDetails
	AlertPanelDetails []AlertPanelDetail `json:"alert_panel_details"`
}

type AlertPanelDetail struct {
	PanelName string `json:"panel_name"`
	PanelId   string `json:"panel_id"`
	Refresh   string `json:"refresh"`
	Setting   jsonutils.JSONObject
	PanelDetails
}
