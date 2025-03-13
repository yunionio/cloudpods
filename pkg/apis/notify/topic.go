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

package notify

import "yunion.io/x/onecloud/pkg/apis"

const (
	DefaultResourceCreateDelete    = "resource create or delete"
	DefaultResourceChangeConfig    = "resource change config"
	DefaultResourceUpdate          = "resource update"
	DefaultResourceReleaseDue1Day  = "resource release due 1 day"
	DefaultResourceReleaseDue3Day  = "resource release due 3 day"
	DefaultResourceReleaseDue30Day = "resource release due 30 day"
	DefaultResourceRelease         = "resource release"
	DefaultScheduledTaskExecute    = "scheduled task execute"
	DefaultScalingPolicyExecute    = "scaling policy execute"
	DefaultSnapshotPolicyExecute   = "snapshot policy execute"
	DefaultResourceOperationFailed = "resource operation failed"
	DefaultResourceSync            = "resource sync"
	DefaultSystemExceptionEvent    = "system exception event"
	DefaultChecksumTestFailed      = "checksum test failed"
	DefaultUserLock                = "user lock"
	DefaultActionLogExceedCount    = "action log exceed count"
	DefaultSyncAccountStatus       = "cloud account sync status"
	DefaultPasswordExpireDue1Day   = "password expire due 1 day"
	DefaultPasswordExpireDue7Day   = "password expire due 7 day"
	DefaultPasswordExpire          = "password expire"
	DefaultNetOutOfSync            = "net out of sync"
	DefaultMysqlOutOfSync          = "mysql out of sync"
	DefaultServiceAbnormal         = "service abnormal"
	DefaultServerPanicked          = "server panicked"
)

type TopicUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput
	TitleCn           string
	TitleEn           string
	ContentCn         string
	ContentEn         string
	AdvanceDays       []int `json:"advance_days"`
	WebconsoleDisable *bool
	Actions           []string
	Resources         []string
}

type TopicListInput struct {
	apis.StandaloneResourceListInput
	apis.EnabledResourceBaseListInput
}

type TopicDetails struct {
	apis.StandaloneResourceDetails
	STopic

	// description: resources managed
	// example: ["server", "eip", "disk"]
	Resources   []string `json:"resource_types"`
	Actions     []string `json:"actions"`
	AdvanceDays []int    `json:"advance_days"`
}

type PerformEnableInput struct {
}

type PerformDisableInput struct {
}

type STopicGroupKeys []string
type TopicAdvanceDays []int

type STopicCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput
	Type              string           `json:"type"`
	Results           bool             `json:"results"`
	TitleCn           string           `json:"title_cn"`
	TitleEn           string           `json:"title_en"`
	ContentCn         string           `json:"content_cn"`
	ContentEn         string           `json:"content_en"`
	GroupKeys         *STopicGroupKeys `json:"group_keys"`
	AdvanceDays       []int            `json:"advance_days"`
	Resources         []string         `json:"resources"`
	Actions           []string         `json:"actions"`
	WebconsoleDisable bool             `json:"webconsole_disable"`
}
