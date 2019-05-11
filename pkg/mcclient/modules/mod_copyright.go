package modules

type CopyrightManager struct {
	ResourceManager
}

var (
	Copyright CopyrightManager
)

func init() {
	Copyright = CopyrightManager{NewYunionAgentManager("info", "infos",
		[]string{"copyright", "email"},
		[]string{})}
	register(&Copyright)
}
