// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitor

import (
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
	Setting   *AlertSetting
	PanelDetails
}

type AlertClonePanelInput struct {
	PanelId        string `json:"panel_id"`
	ClonePanelName string `json:"clone_panel_name"`
}

type AlertCloneDashboardInput struct {
	CloneName string `json:"clone_name"`
}

type AlertPanelSetOrderInput struct {
	Order []struct {
		PanelId string `json:"panel_id"`
		Index   int    `json:"index"`
	} `json:"order"`
}
