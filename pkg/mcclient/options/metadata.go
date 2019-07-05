package options

type MetadataListOptions struct {
	Resources []string `help:"list of resource e.g server、disk、eip、snapshot, empty will show all metadata"`
	SysMeta   *bool    `help:"Show sys metadata only"`
	CloudMeta *bool    `help:"Show cloud metadata olny"`
	UserMeta  *bool    `help:"Show user metadata olny"`
	Admin     *bool    `help:"Show all metadata"`

	WithSysMeta   *bool `help:"Show sys metadata"`
	WithCloudMeta *bool `help:"Show cloud metadata"`
	WithUserMeta  *bool `help:"Show user metadata"`
}

type TagListOptions MetadataListOptions
