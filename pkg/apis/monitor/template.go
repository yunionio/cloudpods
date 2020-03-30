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

type NotificationTemplateCreateInput struct {
	Content string `json:"content"`
}

type NotificationTemplateConfig struct {
	Title   string      `json:"title"`
	Name    string      `json:"name"`
	Matches []EvalMatch `json:"matches"`
	// PrevAlertState AlertStateType `json:"prev_alert_state"`
	// State AlertStateType `json:"state"`
	NoDataFound bool   `json:"no_data"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Level       string `json:"level"`
	IsRecovery  bool   `json:"is_recovery"`
}
