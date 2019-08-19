package modules

type ConfigsManager struct {
	ResourceManager
}

var (
	Configs ConfigsManager
)

func init() {
	Configs = ConfigsManager{NewNotifyManager("config", "configs",
		[]string{},
		[]string{})}

	register(&Configs)
}
