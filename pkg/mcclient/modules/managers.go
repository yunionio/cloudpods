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
	"yunion.io/x/onecloud/pkg/apis"
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
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_REGION, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewActionManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_LOG, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewServiceTreeManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_SERVICETREE, "", "v1", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewMonitorV2Manager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_MONITOR, "", "", columns, adminColumns),
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
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_CLOUDWATCHER, "", "v1", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewNotifyManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_NOTIFY, "", "v1", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewNotifyv2Manager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager:   *modulebase.NewBaseManager(apis.SERVICE_TYPE_NOTIFY, "", "v2", columns, adminColumns),
		Keyword:       keyword,
		KeywordPlural: keywordPlural,
	}
}

func NewJointComputeManager(keyword, keywordPlural string, columns, adminColumns []string, master, slave modulebase.Manager) modulebase.JointResourceManager {
	return modulebase.JointResourceManager{
		ResourceManager: NewComputeManager(keyword, keywordPlural, columns, adminColumns),
		Master:          master,
		Slave:           slave}
}

func NewJointCloudIdManager(keyword, keywordPlural string, columns, adminColumns []string, master, slave modulebase.Manager) modulebase.JointResourceManager {
	return modulebase.JointResourceManager{
		ResourceManager: NewCloudIdManager(keyword, keywordPlural, columns, adminColumns),
		Master:          master,
		Slave:           slave}
}

func NewJointServiceTreeManager(keyword, keywordPlural string, columns, adminColumns []string, master, slave modulebase.Manager) modulebase.JointResourceManager {
	return modulebase.JointResourceManager{
		ResourceManager: NewServiceTreeManager(keyword, keywordPlural, columns, adminColumns),
		Master:          master,
		Slave:           slave}
}

func NewIdentityManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_KEYSTONE, "", "v2.0", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewIdentityV3Manager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_KEYSTONE, "", "v3", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewImageManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_IMAGE, "", "v1", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewVNCProxyManager() modulebase.ResourceManager {
	return modulebase.ResourceManager{BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_VNCPROXY, "", "", nil, nil),
		Keyword: "vncproxy", KeywordPlural: "vncproxy"}
}

func NewITSMManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_ITSM, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewSchedulerManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_SCHEDULER, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewYunionAgentManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_YUNIONAGENT, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewYunionConfManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_YUNIONCONF, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewAutoUpdateManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_AUTOUPDATE, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewWebsocketManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_WEBSOCKET, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

// deprecate
func NewCloudmetaManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_CLOUDMETA, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewOfflineCloudmetaManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_OFFLINE_CLOUDMETA, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewAnsibleManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_ANSIBLE, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewDevtoolManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_DEVTOOL, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewCloudeventManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_CLOUDEVENT, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewCloudIdManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_CLOUDID, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}

func NewSuggestionManager(keyword, keywordPlural string, columns, adminColumns []string) modulebase.ResourceManager {
	return modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(apis.SERVICE_TYPE_SUGGESTION, "", "", columns, adminColumns),
		Keyword:     keyword, KeywordPlural: keywordPlural}
}
