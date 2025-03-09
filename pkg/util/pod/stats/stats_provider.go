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

package stats

import "yunion.io/x/onecloud/pkg/util/pod/cadvisor"

type ContainerStatsProvider interface {
	ListPodStats() ([]PodStats, error)
	ListPodStatsAndUpdateCPUNanoCoreUsage() ([]PodStats, error)
	ListPodCPUAndMemoryStats() ([]PodStats, error)
	ImageFsStats() (FsStats, error)
	ImageFsDevice() (string, error)
}

type StatsProvider struct {
	cadvisor cadvisor.Interface
	ContainerStatsProvider
}

func NewCRIStatsProvider(
	cadvisor cadvisor.Interface,
) *StatsProvider {
	return nil
}

func newStatsProvider(
	cadvisor cadvisor.Interface,
) *StatsProvider {
	return &StatsProvider{
		cadvisor: cadvisor,
	}
}
