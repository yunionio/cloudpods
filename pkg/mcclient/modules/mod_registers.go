package modules

type RegisterManager struct {
	ResourceManager
}

var (
	AccountRegister RegisterManager
)

func init() {
	AccountRegister = RegisterManager{NewYunionAgentManager("register", "registers",
		[]string{},
		[]string{})}
	register(&AccountRegister)
}
