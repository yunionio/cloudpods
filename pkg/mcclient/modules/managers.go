package modules

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

func NewYunionAgentManager(keyword, keywordPlural string, columns, adminColumns []string) ResourceManager {
	return ResourceManager{
		BaseManager: BaseManager{columns: columns,
			adminColumns: adminColumns,
			serviceType:  "yunionagent"},
		Keyword: keyword, KeywordPlural: keywordPlural}
}
