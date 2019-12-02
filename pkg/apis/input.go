package apis

type DomainizedResourceCreateInput struct {
	// description: the owner domain name or id
	// required: false
	Domain   string `json:"project_domain"`

	// description: the owner domain name or id, alias field of domain
	// required: false
	DomainId string `json:"domain_id"`
}

type ProjectizedResourceCreateInput struct {
	DomainizedResourceCreateInput

	// description: the owner project name or id
	// required: false
	Project   string `json:"project"`

	// description: the owner project name or id, alias field of project
	// required: false
	ProjectId string `json:"project_id"`
}

type SharableVirtualResourceCreateInput struct {
	VirtualResourceCreateInput

	// description: indicate the resource is a public resource
	// required: false
	IsPublic    *bool  `json:"is_public"`

	// description: indicate the shared scope for a public resource, which can be domain or system or none
	// required: false
	PublicScope string `json:"public_scope"`
}

type VirtualResourceCreateInput struct {
	StatusStandaloneResourceCreateInput
	ProjectizedResourceCreateInput

	// description: indicate the resource is a system resource, which is not visible to user
	// required: false
	IsSystem *bool `json:"is_system"`
}

type EnabledStatusStandaloneResourceCreateInput struct {
	StatusStandaloneResourceCreateInput

	// description: indicate the resource is enabled/disabled by administrator
	// required: false
	Enabled *bool `json:"enabled"`
}

type StatusStandaloneResourceCreateInput struct {
	StandaloneResourceCreateInput

	// description: the status of the resource
	// required: false
	Status string `json:"status"`
}

type StandaloneResourceCreateInput struct {
	ResourceBaseCreateInput

	// description: resource name, required if generated_name is not given
	// unique: true
	// required: false
	// example: test-network
	Name         string `json:"name"`

	// description: generated resource name, given a pattern to generate name, required if name is not given
	// unique: false
	// required: false
	// example: test###
	GenerateName string `json:"generate_name"`

	// description: resource description
	// required: false
	// example: test create network
	Description  string `json:"description"`

	// description: the resource is an emulated resource
	// required: false
	IsEmulated   *bool  `json:"is_emulated"`
}

type JoinResourceBaseCreateInput struct {
	ResourceBaseCreateInput
}

type ResourceBaseCreateInput struct {
	ModelBaseCreateInput
}

type ModelBaseCreateInput struct {
	Meta
}
