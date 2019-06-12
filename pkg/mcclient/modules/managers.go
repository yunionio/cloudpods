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

/*
添加新manager注意事项：
1. version字段   -- 在endpoint中注册的url如果携带版本。例如http://x.x.x.x/api/v1，那么必须标注对应version字段。否者可能导致yunionapi报资源not found的错误。
*/

func NewResourceManager(serviceType string, keyword, keywordPlural string,
	columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  serviceType},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewComputeManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "compute"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewActionManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "log"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewMonitorManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			version:      "v1",
			serviceType:  "servicetree"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewCloudwatcherManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			version:      "v1",
			serviceType:  "cloudwatcher"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewNotifyManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			version:      "v1",
			serviceType:  "notify"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewJointComputeManager(keyword, keywordPlural string, columns, adminColumns []string, master, slave Manager) JointResourceManager {
	return JointResourceManager{
		ResourceManager: NewComputeManager(keyword, keywordPlural, columns, adminColumns),
		Master:          master,
		Slave:           slave}
}

func NewJointMonitorManager(keyword, keywordPlural string, columns, adminColumns []string, master, slave Manager) JointResourceManager {
	return JointResourceManager{
		ResourceManager: NewMonitorManager(keyword, keywordPlural, columns, adminColumns),
		Master:          master,
		Slave:           slave}
}

func NewIdentityManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			endpointType: "adminURL",
			version:      "v2.0",
			serviceType:  "identity"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewIdentityV3Manager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			endpointType: "adminURL",
			version:      "v3",
			serviceType:  "identity"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewImageManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			endpointType: "",
			version:      "v1",
			serviceType:  "image"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewVNCProxyManager() ResourceManager {
	return ResourceManager{BaseManager: BaseManager{serviceType: "vncproxy"},
		Keyword: "vncproxy", KeywordPlural: "vncproxy"}
}

func NewITSMManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "itsm"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewSchedulerManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "scheduler"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewMeterManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			version:      "v1",
			serviceType:  "meter"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewMeterAlertManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "meteralert"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewYunionAgentManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "yunionagent"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewYunionConfManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "yunionconf"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewAutoUpdateManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "autoupdate"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewWebsocketManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "websocket"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewCloudmetaManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "cloudmeta"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}

func NewAnsibleManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "ansible"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}
