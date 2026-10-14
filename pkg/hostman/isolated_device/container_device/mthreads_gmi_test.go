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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMthreadsGmiS5000(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "s5000-8gpu.txt"))
	require.NoError(t, err)

	gpus := parseMthreadsGmi(string(raw))
	require.Len(t, gpus, 9)

	physical := make([]*parsedMthreadsGPU, 0, 8)
	for _, gpu := range gpus {
		if isMthreadsPhysical(gpu) {
			physical = append(physical, gpu)
		}
	}
	require.Len(t, physical, 8)

	wantBus := []string{
		"00000000:18:00.0",
		"00000000:3a:00.0",
		"00000000:4b:00.0",
		"00000000:5c:00.0",
		"00000000:9a:00.0",
		"00000000:ba:00.0",
		"00000000:ca:00.0",
		"00000000:da:00.0",
	}
	wantTemp := []int{45, 38, 44, 36, 45, 37, 45, 38}
	for i, gpu := range physical {
		assert.Equal(t, i, gpu.Index)
		assert.Equal(t, "MTT S5000", gpu.Name)
		assert.Equal(t, wantBus[i], gpu.BusId)
		assert.Equal(t, 0, gpu.GPUUtil)
		assert.Equal(t, 0, gpu.MemoryUsedMB)
		assert.Equal(t, 81920, gpu.MemorySizeMB)
		assert.Equal(t, wantTemp[i], gpu.TempC)
		assert.Equal(t, "P0", gpu.Perf)
		assert.Equal(t, "YES", gpu.MPCCapable)
		assert.Equal(t, "On-die, EDC", gpu.ECCMode)
	}

	virt := gpus[8]
	assert.Equal(t, 8, virt.Index)
	assert.Equal(t, "Virt", virt.DeviceType)
	assert.False(t, isMthreadsPhysical(virt))
	assert.Equal(t, 10, virt.GPUUtil)
	assert.Equal(t, 1024, virt.MemoryUsedMB)
}

func TestBuildMthreadsExtraConfigures(t *testing.T) {
	envs, mounts := buildMthreadsExtraConfigures([]string{"0", "1", "2", "3"})
	assert.Nil(t, mounts)
	require.Len(t, envs, 2)
	got := map[string]string{}
	for _, env := range envs {
		got[env.Key] = env.Value
	}
	assert.Equal(t, "0,1,2,3", got["MTHREADS_VISIBLE_DEVICES"])
	assert.Equal(t, "all", got["MTHREADS_DRIVER_CAPABILITIES"])

	envs, mounts = buildMthreadsExtraConfigures(nil)
	assert.Nil(t, envs)
	assert.Nil(t, mounts)
}

func TestBuildMthreadsHamiQosEnvs(t *testing.T) {
	envs := buildMthreadsHamiQosEnvs(2048, 25)
	require.Len(t, envs, 2)
	got := map[string]string{}
	for _, env := range envs {
		got[env.Key] = env.Value
	}
	assert.Equal(t, "2147483648", got["MTHREADS_QOS_MEMORY_LIMIT"])
	assert.Equal(t, "25", got["MTHREADS_QOS_COMPUTING_POWER_WEIGHT"])

	zero := buildMthreadsHamiQosEnvs(0, 25)
	assert.Nil(t, zero)

	defaultWeight := buildMthreadsHamiQosEnvs(512, 0)
	require.Len(t, defaultWeight, 2)
	assert.Equal(t, "1", defaultWeight[1].Value)
}

func TestMthreadsDevNode(t *testing.T) {
	assert.Equal(t, "/dev/mtgpu.0", mthreadsDevNode(0))
	assert.Equal(t, "/dev/mtgpu.7", mthreadsDevNode(7))
}
