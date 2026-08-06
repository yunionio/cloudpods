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

package container_device

import (
	"testing"

	"github.com/stretchr/testify/assert"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
)

func TestInitHygonCoreUsage(t *testing.T) {
	assert.Equal(t, "0000000000000000", initHygonCoreUsage(60))
}

func TestAllocHygonCoreUsage(t *testing.T) {
	tot := initHygonCoreUsage(60)
	mask, remains, err := allocHygonCoreUsage(tot, 4)
	assert.NoError(t, err)
	assert.Equal(t, 0, remains)
	assert.NotEmpty(t, mask)

	merged, err := addHygonCoreUsage(tot, mask)
	assert.NoError(t, err)
	assert.Equal(t, mask, merged)
}

func TestHygonPciBusIdFromAddr(t *testing.T) {
	assert.Equal(t, "0000:09:00.0", hygonPciBusIdFromAddr("09:00.0"))
	assert.Equal(t, "0000:09:00.0", hygonPciBusIdFromAddr("0000:09:00.0"))
}

func TestIsHygonVDcuRequest(t *testing.T) {
	dev := &hostapi.ContainerDevice{
		IsolatedDevice: &hostapi.ContainerIsolatedDevice{
			MemoryLimit: 8192,
		},
	}
	assert.True(t, isHygonVDcuRequest([]*hostapi.ContainerDevice{dev}, 65520))
	assert.False(t, isHygonVDcuRequest([]*hostapi.ContainerDevice{dev, dev}, 65520))
	dev.IsolatedDevice.MemoryLimit = 65520
	assert.False(t, isHygonVDcuRequest([]*hostapi.ContainerDevice{dev}, 65520))
}

func TestHygonVgpuCacheDirName(t *testing.T) {
	name := hygonVgpuCacheDirName("guest1", "ctr1", 0, 1, 2, "abc", "def")
	assert.Equal(t, "guest1_ctr1_0_1_2_abc_def", name)
}
