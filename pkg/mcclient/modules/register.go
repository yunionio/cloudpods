package modules

import "yunion.io/x/onecloud/pkg/mcclient"

func registerCompute(mod BaseManagerInterface) {
	registerComputeV1(mod)
	registerComputeV2(mod)
}

func registerComputeV1(mod BaseManagerInterface) {
	_register("v1", mod)
}

func registerComputeV2(mod BaseManagerInterface) {
	mod.SetApiVersion(mcclient.V2_API_VERSION)
	_register("v2", mod)
}

func register(mod BaseManagerInterface) {
	_register("v1", mod)
}

func registerV2(mod BaseManagerInterface) {
	_register("v2", mod)
}

func Register(mod BaseManagerInterface) {
	register(mod)
}
