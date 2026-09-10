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
	"github.com/stretchr/testify/require"

	"yunion.io/x/onecloud/pkg/hostman/options"
)

func TestTHeadPpuDevNodeUsesIndexNotMinor(t *testing.T) {
	assert.Equal(t, "/dev/alixpu_ppu0", tHeadPpuDevNode(0))
	assert.Equal(t, "/dev/alixpu_ppu3", tHeadPpuDevNode(3))
	assert.NotEqual(t, "/dev/alixpu_ppu4", tHeadPpuDevNode(3))
}

func TestCollectTHeadPpuCommonDevicePaths(t *testing.T) {
	assert.Nil(t, collectTHeadPpuCommonDevicePaths(nil))
	assert.Empty(t, collectTHeadPpuCommonDevicePaths(func(string) bool { return false }))
	assert.Equal(t, []string{
		"/dev/alixpu",
		"/dev/alixpu_ctl",
		"/dev/alixpu_sep",
	}, collectTHeadPpuCommonDevicePaths(func(string) bool { return true }))
	assert.Equal(t, []string{"/dev/alixpu", "/dev/alixpu_sep"}, collectTHeadPpuCommonDevicePaths(func(p string) bool {
		return p != "/dev/alixpu_ctl"
	}))
}

func TestParsePpuSmiQueryCSV(t *testing.T) {
	input := `index, name, uuid, pci.bus_id, memory.total [MiB]
0, PPU-ZW810E, GPU-019e2226-4211-0208-0000-000000ab261d, 00000000:a8:00.0, 98304 MiB
1, PPU-ZW810E, GPU-019e2226-84c1-0200-0000-0000c020e701, 00000000:a7:00.0, 98304`
	gpus := parsePpuSmiQueryCSV(input)
	require.Len(t, gpus, 2)
	assert.Equal(t, 0, gpus[0].Index)
	assert.Equal(t, "PPU-ZW810E", gpus[0].Name)
	assert.Equal(t, "GPU-019e2226-4211-0208-0000-000000ab261d", gpus[0].UUID)
	assert.Equal(t, "00000000:a8:00.0", gpus[0].BusId)
	assert.Equal(t, 98304, gpus[0].MemorySizeMB)
	assert.Equal(t, 1, gpus[1].Index)
}

func TestParsePpuSmiList(t *testing.T) {
	input := `PPU 0: PPU-ZW810E (UUID: GPU-019ea108-c110-0828-0000-0000c07e1a46)
PPU 1: PPU (UUID: GPU-019ea108-c120-040c-0000-0000c0267f1e)`
	uuids := parsePpuSmiList(input)
	assert.Equal(t, "GPU-019ea108-c110-0828-0000-0000c07e1a46", uuids[0])
	assert.Equal(t, "GPU-019ea108-c120-040c-0000-0000c0267f1e", uuids[1])
}

func TestTHeadPpuPCIAddrCandidates(t *testing.T) {
	cands := tHeadPpuPCIAddrCandidates("00000000:a8:00.0")
	assert.Equal(t, []string{"00000000:a8:00.0", "0000:a8:00.0", "a8:00.0"}, cands)
}

func TestBuildTHeadPpuExtraConfigures(t *testing.T) {
	exists := func(p string) bool {
		return p == "/usr/local/PPU_SDK" || p == "/usr/local/PPU_SDK/lib"
	}
	envs, mounts := buildTHeadPpuExtraConfigures([]string{"0", "3"}, "/usr/local/PPU_SDK", exists)
	require.Len(t, envs, 3)
	assert.Equal(t, "CUDA_VISIBLE_DEVICES", envs[0].Key)
	assert.Equal(t, "0,3", envs[0].Value)
	assert.Equal(t, "NVIDIA_VISIBLE_DEVICES", envs[1].Key)
	assert.Equal(t, "0,3", envs[1].Value)
	assert.Equal(t, "LD_LIBRARY_PATH", envs[2].Key)
	assert.Equal(t, "/usr/local/PPU_SDK/lib", envs[2].Value)
	require.Len(t, mounts, 1)
	assert.Equal(t, "/usr/local/PPU_SDK", mounts[0].HostPath)
	assert.True(t, mounts[0].Readonly)

	envs, mounts = buildTHeadPpuExtraConfigures(nil, "/usr/local/PPU_SDK", exists)
	assert.Nil(t, envs)
	assert.Nil(t, mounts)
}

func TestParseTHeadPpuNodeIndex(t *testing.T) {
	idx, ok := parseTHeadPpuNodeIndex("alixpu_ppu3")
	assert.True(t, ok)
	assert.Equal(t, 3, idx)
	_, ok = parseTHeadPpuNodeIndex("alixpu")
	assert.False(t, ok)
}

func TestTHeadPpuSmiPathDefault(t *testing.T) {
	orig := options.HostOptions.THeadPpuSmiPath
	defer func() { options.HostOptions.THeadPpuSmiPath = orig }()
	options.HostOptions.THeadPpuSmiPath = ""
	assert.Equal(t, "/usr/local/bin/ppu-smi", tHeadPpuSmiPath())
	options.HostOptions.THeadPpuSmiPath = "/opt/bin/ppu-smi"
	assert.Equal(t, "/opt/bin/ppu-smi", tHeadPpuSmiPath())
}
