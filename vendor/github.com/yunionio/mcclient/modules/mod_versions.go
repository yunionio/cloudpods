package modules

type VersionManager struct {
	ResourceManager
}

var (
	Version VersionManager
)

func init() {
	Version = VersionManager{NewYunionAgentManager("version", "versions",
		[]string{},
		[]string{})}
	register(&Version)
}
