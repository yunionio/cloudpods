package options

type VersionListOptions struct {
	Region string
}

type VersionGetOptions struct {
	Service string `choices:"cloud" default:"cloud"`
}
