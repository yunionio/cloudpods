package options

type MetadataListOptions struct {
	Resources []string `help:"list of resource e.g server、disk、eip、snapshot, empty will show all metadata"`
	WithSys   *bool    `help:"With sys metadata"`
	WithCloud *bool    `help:"With cloud metadata"`
}
