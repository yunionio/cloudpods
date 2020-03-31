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

package modules

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

/*
添加新manager注意事项：
1. version字段   -- 在endpoint中注册的url如果携带版本。例如http://x.x.x.x/api/v1，那么必须标注对应version字段。否者可能导致yunionapi报资源not found的错误。
*/

func NewResourceManager(serviceType string, keyword, keywordPlural string,
	columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(serviceType, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewComputeManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("compute", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewActionManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("log", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewMonitorManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("servicetree", "", "v1", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewMonitorV2Manager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("monitor", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewJointMonitorV2Manager(keyword, keywordPlural string, columns, adminColumns []string, master, slave modulebase.Manager) modulebase.JointResourceManager {
	return modulebase.JointResourceManager{
		ResourceManager: NewMonitorV2Manager(keyword, keywordPlural, columns, adminColumns),
		Master:          master,
		Slave:           slave}
}

func NewCloudwatcherManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("cloudwatcher", "", "v1", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewNotifyManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("notify", "", "v1", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewJointComputeManager(keyword, keywordPlural string, columns, adminColumns []string, master, slave modulebase.Manager) modulebase.JointResourceManager {
	return modulebase.JointResourceManager{
		ResourceManager: NewComputeManager(keyword, keywordPlural, columns, adminColumns),
		Master:          master,
		Slave:           slave}
}

func NewJointMonitorManager(keyword, keywordPlural string, columns, adminColumns []string, master, slave modulebase.Manager) modulebase.JointResourceManager {
	return modulebase.JointResourceManager{
		ResourceManager: NewMonitorManager(keyword, keywordPlural, columns, adminColumns),
		Master:          master,
		Slave:           slave}
}

func NewIdentityManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("identity", "", "v2.0", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewIdentityV3Manager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("identity", "", "v3", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewImageManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("image", "", "v1", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewVNCProxyManager() modulebase.ResourceManager {
	return modulebase.ResourceManager{BaseManager: *modulebase.NewBaseManager("vncproxy", "", "", nil, nil),
		Keyword: "vncproxy", KeywordPlural: "vncproxy"}
}

func NewITSMManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("itsm", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewSchedulerManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("scheduler", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewMeterManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("meter", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewYunionAgentManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("yunionagent", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewYunionConfManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("yunionconf", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewAutoUpdateManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("autoupdate", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewWebsocketManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("websocket", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

// deprecate
func NewCloudmetaManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("cloudmeta", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewOfflineCloudmetaManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("offlinecloudmeta", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewAnsibleManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("ansible", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewDevtoolManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("devtool", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewCloudeventManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager("cloudevent", "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}
