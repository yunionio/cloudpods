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

package multicloud

import "yunion.io/x/cloudmux/pkg/apis"

type SHostBase struct {
	SResourceBase
	STagBase
}

func (host *SHostBase) GetCpuCmtbound() float32 {
	return 0.0
}

func (host *SHostBase) GetMemCmtbound() float32 {
	return 0.0
}

func (host *SHostBase) GetReservedMemoryMb() int {
	return 0
}

func (host *SHostBase) GetSchedtags() ([]string, error) {
	return nil, nil
}

func (host *SHostBase) GetOvnVersion() string {
	return ""
}

func (host *SHostBase) GetCpuArchitecture() string {
	return apis.OS_ARCH_X86_64
}
