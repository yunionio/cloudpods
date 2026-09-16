// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain the copy of the License at
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
)

func TestIluvatarDevNode(t *testing.T) {
	assert.Equal(t, "/dev/iluvatar0", iluvatarDevNode(0))
	assert.Equal(t, "/dev/iluvatar7", iluvatarDevNode(7))
}

func TestResolveIluvatarDeviceMinor(t *testing.T) {
	exists := func(p string) bool { return p == "/dev/iluvatar3" }
	assert.Equal(t, 3, resolveIluvatarDeviceMinor(3, exists))
	assert.Equal(t, -1, resolveIluvatarDeviceMinor(0, exists))
	assert.Equal(t, -1, resolveIluvatarDeviceMinor(0, nil))
}

func TestCollectIluvatarCommonDevicePaths(t *testing.T) {
	assert.Nil(t, collectIluvatarCommonDevicePaths(nil))
	assert.Nil(t, collectIluvatarCommonDevicePaths(func(string) bool { return false }))
	assert.Equal(t, []string{"/dev/itrctl"}, collectIluvatarCommonDevicePaths(func(p string) bool {
		return p == "/dev/itrctl"
	}))
}

func TestIluvatarDeviceSpec(t *testing.T) {
	dev := iluvatarDeviceSpec("/dev/iluvatar2")
	require.NotNil(t, dev)
	assert.Equal(t, "/dev/iluvatar2", dev.HostPath)
	assert.Equal(t, "/dev/iluvatar2", dev.ContainerPath)
	assert.Equal(t, "rwm", dev.Permissions)
}

func TestBuildIluvatarExtraConfigures(t *testing.T) {
	exists := func(p string) bool { return p == "/usr/local/corex-4.4.0" }
	envs, mounts := buildIluvatarExtraConfigures([]string{"0", "1", "2", "3"}, "/usr/local/corex-4.4.0", exists)
	require.Len(t, envs, 3)
	assert.Equal(t, "IX_VISIBLE_DEVICES", envs[0].Key)
	assert.Equal(t, "0,1,2,3", envs[0].Value)
	assert.Equal(t, "COREX_HOME", envs[1].Key)
	assert.Equal(t, "/usr/local/corex-4.4.0", envs[1].Value)
	assert.Equal(t, "LD_LIBRARY_PATH", envs[2].Key)
	assert.Equal(t, "/usr/local/corex-4.4.0/lib64", envs[2].Value)

	require.Len(t, mounts, 2)
	assert.Equal(t, "/usr/local/corex-4.4.0", mounts[0].HostPath)
	assert.Equal(t, "/usr/local/corex-4.4.0", mounts[0].ContainerPath)
	assert.True(t, mounts[0].Readonly)
	assert.Equal(t, "/usr/local/corex-4.4.0", mounts[1].HostPath)
	assert.Equal(t, "/usr/local/corex", mounts[1].ContainerPath)
	assert.True(t, mounts[1].Readonly)

	envs, mounts = buildIluvatarExtraConfigures(nil, "/usr/local/corex-4.4.0", exists)
	assert.Nil(t, envs)
	assert.Nil(t, mounts)
}

func TestBuildIluvatarRuntimeMountsAliasOnly(t *testing.T) {
	exists := func(p string) bool { return p == "/usr/local/corex" }
	mounts := buildIluvatarRuntimeMounts("/usr/local/corex", exists)
	require.Len(t, mounts, 1)
	assert.Equal(t, "/usr/local/corex", mounts[0].HostPath)
	assert.Equal(t, "/usr/local/corex", mounts[0].ContainerPath)
}

func TestBuildIluvatarRuntimeMountsMissingHome(t *testing.T) {
	assert.Nil(t, buildIluvatarRuntimeMounts("/usr/local/corex-4.4.0", func(string) bool { return false }))
}

func TestParseIxsmiTableTwoGpus(t *testing.T) {
	input := `| 0    Iluvatar BI-V150S           | 00000000:26:00.0     | 500MHz    1600MHz    |
| N/A  35C   P0       N/A / N/A    | 68MiB / 32768MiB     | 0%        Default    |
| 1    Iluvatar BI-V150S           | 00000000:29:00.0     | 500MHz    1600MHz    |
| N/A  33C   P0       53W / 450W   | 68MiB / 32768MiB     | 0%        Default    |`
	gpus := parseIxsmiTable(input)
	require.Len(t, gpus, 2)
	assert.Equal(t, 0, gpus[0].Index)
	assert.Equal(t, "Iluvatar BI-V150S", gpus[0].Name)
	assert.Equal(t, "00000000:26:00.0", gpus[0].BusId)
	assert.Equal(t, 32768, gpus[0].MemorySizeMB)
	assert.Equal(t, "Default", gpus[0].ComputeMode)
	assert.Equal(t, 1, gpus[1].Index)
}

func TestIluvatarPCIAddrCandidates(t *testing.T) {
	cands := iluvatarPCIAddrCandidates("00000000:26:00.0")
	assert.Equal(t, []string{"00000000:26:00.0", "0000:26:00.0", "26:00.0"}, cands)
}
