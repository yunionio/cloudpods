// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
