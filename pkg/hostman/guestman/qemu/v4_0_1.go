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

package qemu

func init() {
	RegisterCmd(
		newCmd_4_0_1_x86_64(),
		newCmd_4_0_1_aarch64(),
	)
}

func newCmd_4_0_1_x86_64() QemuCommand {
	return newBaseCommand(
		Version_4_0_1,
		Arch_x86_64,
		newOpt_4_0_1_x86_64())
}

type opt_401_x86_64 struct {
	*baseOptions_x86_64
}

func newOpt_4_0_1_x86_64() QemuOptions {
	return &opt_401_x86_64{
		baseOptions_x86_64: newBaseOptions_x86_64(),
	}
}

func (o opt_401_x86_64) Nodefconfig() string {
	return "-no-user-config"
}

func (o opt_401_x86_64) NoKVMPitReinjection() string {
	// https://www.qemu.org/docs/master/about/removed-features.html#no-kvm-pit-reinjection-removed-in-3-0
	// -no-kvm-pit-reinjection (removed in 3.0)
	return ""
}

func newCmd_4_0_1_aarch64() QemuCommand {
	return newBaseCommand(
		Version_4_0_1,
		Arch_aarch64,
		newOpt_4_0_1_aarch64())
}

type opt_401_aarch64 struct {
	*baseOptions_aarch64
}

func newOpt_4_0_1_aarch64() QemuOptions {
	return &opt_401_aarch64{
		baseOptions_aarch64: newBaseOptions_aarch64(),
	}
}

func (o opt_401_aarch64) Nodefconfig() string {
	return "-no-user-config"
}
