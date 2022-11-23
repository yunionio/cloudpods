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

package options

import (
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type AlerterOptions struct {
	common_options.CommonOptions
	common_options.DBOptions

	DataProxyTimeout                               int   `help:"query data source proxy timeout" default:"30"`
	AlertingMinIntervalSeconds                     int64 `help:"alerting min schedule frequency" default:"10"`
	AlertingMaxAttempts                            int   `help:"alerting engine max attempt" default:"3"`
	AlertingEvaluationTimeoutSeconds               int64 `help:"alerting evaluation timeout" default:"5"`
	AlertingNotificationTimeoutSeconds             int64 `help:"alerting notification timeout" default:"30"`
	InitScopeSuggestConfigIntervalSeconds          int   `help:"internal to init scope suggest configs" default:"900"`
	InitAlertResourceAdminRoleUsersIntervalSeconds int   `help:"internal to init alert resource admin role users " default:"3600"`
	MonitorResourceSyncIntervalSeconds             int   `help:"internal to sync monitor resource,unit: h " default:"1"`

	APISyncIntervalSeconds  int `default:"3600"`
	APIRunDelayMilliseconds int `default:"5000"`
	APIListBatchSize        int `default:"1024"`

	WorkerCheckInterval int `default:"180"`

	AutoMigrationMustPair bool `default:"false" help:"result of auto migration source guests and target hosts must be paired"`
}

var (
	Options AlerterOptions
)
