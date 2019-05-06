package modules

type CopyRightManager struct {
	ResourceManager
}

var (
	CopyRight CopyRightManager
)

func init() {
	CopyRight = CopyRightManager{NewYunionAgentManager("info", "infos",
		[]string{"copyright", "email"},
		[]string{})}
	register(&CopyRight)
}
