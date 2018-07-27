package modules

func registerCompute(mod BaseManagerInterface) {
	_register("v1", mod)
	_register("v2", mod)
}

func register(mod BaseManagerInterface) {
	_register("v1", mod)
}

func Register(mod BaseManagerInterface) {
	register(mod)
}
