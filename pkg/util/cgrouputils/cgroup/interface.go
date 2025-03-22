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

package cgroup

const (
	CGROUP_V1 = "cgroup_v1"
	CGROUP_V2 = "cgroup_v2"
)

type ICGroupTask interface {
	InitTask(hand ICGroupTask, cpuShares int, pid, name string)
	SetPid(string)
	SetName(string)
	SetWeight(coreNum int)
	SetHand(hand ICGroupTask)
	GetParam(name string) string

	CustomConfig(key, value string) bool
	GetStaticConfig() map[string]string
	GetConfig() map[string]string
	Module() string
	RemoveTask() bool
	SetTask() bool
	Configure() bool
	TaskIsExist() bool

	Init() bool
}
