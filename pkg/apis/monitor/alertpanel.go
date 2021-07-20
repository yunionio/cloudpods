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

import "yunion.io/x/onecloud/pkg/apis"

type AlertPanelCreateInput struct {
	apis.ScopedResourceInput
	apis.ProjectizedResourceListInput

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
