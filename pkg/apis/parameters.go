package apis

type BaseListInput struct {
	Meta

	// 查询限制量
	// default: 20
	Limit *int `json:"limit"`
	// 查询偏移量
	// default: 0
	Offset *int `json:"offset"`
	// Name of the field to be ordered by
	OrderBy []string `json:"order_by"`
	// List Order
	// example: desc|asc
	Order string
	// Show more details
	Details *bool `json:"details"`
	// Filter results by a simple keyword search
	Search string `json:"search"`
	// Piggyback metadata information
	WithMeta *bool `json:"with_meta"`
	// Filters
	Filter []string `json:"filters"`
	// Filters with joint table col; joint_tbl.related_key(origin_key).filter_col.filter_cond(filters)
	JointFilter []string `json:"joint_filter"`
	// If true, match if any of the filters matches; otherwise, match if all of the filters match
	FilterAny *bool `json:"filter_any"`
	// Is an admin call?
	Admin *bool `json:"admin"`
	// Tenant ID or Name
	Tenant string `json:"tenant"`
	// Project domain filter
	ProjectDomain string `json:"project_domain"`
	// User ID or Name
	User string `json:"user"`
	// Show only specified fields
	Field []string `json:"field"`
	// Specify query scope, either project, domain or system
	Scope string `json:"scope"`
	// Show system resource
	System *bool `json:"system"`
	// Show only pending deleted resource
	PendingDelete *bool `json:"pending_delete"`
	// Show all resources including pending deleted
	// TODO: fix this???
	PendingDeleteAll *bool `json:"-"`
	// Show all resources including the emulated resources
	ShowEmulated *bool `json:"show_emulated"`
	// Export field keys
	ExportKeys string `json:"export_keys"`

	// TODO: support this tags
	// Tags      []string `help:"Tags info, eg: hypervisor=aliyun, os_type=Linux, os_version" json:"-"`
	// UserTags  []string `help:"UserTags info, eg: group=rd" json:"-"`
	// CloudTags []string `help:"CloudTags info, eg: price_key=cn-beijing" json:"-"`
	// List objects belonging to the cloud provider
	Manager string `json:"manager,omitempty"`
	// List objects belonging to the cloud account
	Account string `json:"account,omitempty"`
	// List objects from the provider, choices:"OneCloud|VMware|Aliyun|Qcloud|Azure|Aws|Huawei|OpenStack|Ucloud|ZStack"
	Provider []string `json:"provider,omitempty"`
	// List objects belonging to a special brand
	Brand []string `json:"brand"`
	// Cloud environment, choices:"public|private|onpremise|private_or_onpremise"
	CloudEnv string `json:"cloud_env,omitempty"`
	// List objects belonging to public cloud
	PublicCloud *bool `json:"public_cloud"`
	// List objects belonging to private cloud
	PrivateCloud *bool `json:"private_cloud"`
	// List objects belonging to on premise infrastructures
	IsOnPremise *bool `json:"is_on_premise"`
	// List objects managed by external providers
	IsManaged *bool `json:"is_managed"`
	// Marker for pagination
	PagingMarker string `json:"paging_marker"`
}
