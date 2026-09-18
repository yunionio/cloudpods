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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"yunion.io/x/onecloud/pkg/hostman/options"
)

func TestKunlunxinXpuDevNodeUsesIndex(t *testing.T) {
	assert.Equal(t, "/dev/xpu0", kunlunxinXpuDevNode(0))
	assert.Equal(t, "/dev/xpu7", kunlunxinXpuDevNode(7))
}

func TestCollectKunlunxinXpuCommonDevicePaths(t *testing.T) {
	assert.Nil(t, collectKunlunxinXpuCommonDevicePaths(nil))
	assert.Empty(t, collectKunlunxinXpuCommonDevicePaths(func(string) bool { return false }))
	assert.Equal(t, []string{"/dev/xpuctrl"}, collectKunlunxinXpuCommonDevicePaths(func(string) bool { return true }))
}

func TestParseXpuSmiTable(t *testing.T) {
	input := `Thu Sep 10 08:18:37 2026       
+-----------------------------------------------------------------------------+
| XPU-SMI               Driver Version: 5.19.0.0     XPU-RT Version: 5.19.0   |
|-------------------------------+----------------------+----------------------+
| XPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
| Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | XPU-Util  Compute M. |
|                               |             L3-Usage |            SR-IOV M. |
|===============================+======================+======================|
|   0  P800 OAM           N/A   | 00000000:03:00.0 N/A |                    0 |
| N/A   35C  N/A     87W / 400W |      0MiB / 98304MiB |      0%      Default |
|                               |      0MiB /    96MiB |             Disabled |
+-------------------------------+----------------------+----------------------+
|   1  P800 OAM           N/A   | 00000000:05:00.0 N/A |                    0 |
| N/A   37C  N/A     84W / 400W |      0MiB / 98304MiB |      0%      Default |
|                               |      0MiB /    96MiB |             Disabled |
+-------------------------------+----------------------+----------------------+
|   7  P800 OAM           N/A   | 00000000:A5:00.0 N/A |                    0 |
| N/A   35C  N/A     85W / 400W |      0MiB / 98304MiB |      0%      Default |
|                               |      0MiB /    96MiB |             Disabled |
+-------------------------------+----------------------+----------------------+
`
	gpus := parseXpuSmiTable(input)
	require.Len(t, gpus, 3)
	assert.Equal(t, 0, gpus[0].Index)
	assert.Equal(t, "P800 OAM", gpus[0].Name)
	assert.Equal(t, "00000000:03:00.0", gpus[0].BusId)
	assert.Equal(t, "00000000:03:00.0", gpus[0].UUID)
	assert.Equal(t, 98304, gpus[0].MemorySizeMB)
	assert.Equal(t, 1, gpus[1].Index)
	assert.Equal(t, "00000000:05:00.0", gpus[1].BusId)
	assert.Equal(t, 7, gpus[2].Index)
	assert.Equal(t, "00000000:A5:00.0", gpus[2].BusId)
}

func TestKunlunxinXpuPCIAddrCandidates(t *testing.T) {
	cands := kunlunxinXpuPCIAddrCandidates("00000000:a8:00.0")
	assert.Equal(t, []string{"00000000:a8:00.0", "0000:a8:00.0", "a8:00.0"}, cands)
}

func TestBuildKunlunxinXpuExtraConfigures(t *testing.T) {
	exists := func(p string) bool {
		return p == "/usr/local/xpu" || p == "/usr/local/xpu/lib"
	}
	envs, mounts := buildKunlunxinXpuExtraConfigures([]string{"0", "3"}, "/usr/local/xpu", "/usr/local/bin/xpu-smi", exists, nil)
	require.Len(t, envs, 2)
	assert.Equal(t, "XPU_VISIBLE_DEVICES", envs[0].Key)
	assert.Equal(t, "0,3", envs[0].Value)
	assert.Equal(t, "LD_LIBRARY_PATH", envs[1].Key)
	assert.Equal(t, "/usr/local/xpu/lib", envs[1].Value)
	require.Len(t, mounts, 1)
	assert.Equal(t, "/usr/local/xpu", mounts[0].HostPath)
	assert.True(t, mounts[0].Readonly)

	envs, mounts = buildKunlunxinXpuExtraConfigures(nil, "/usr/local/xpu", "/usr/local/bin/xpu-smi", exists, nil)
	assert.Nil(t, envs)
	assert.Nil(t, mounts)
}

func TestBuildKunlunxinXpuRuntimeMountsWithSmiAndMlLib(t *testing.T) {
	smi := "/usr/local/bin/xpu-smi"
	ml := "/lib/x86_64-linux-gnu/libxpunvidia-ml.so.1"
	exists := func(p string) bool {
		switch p {
		case "/usr/local/xpu", "/usr/local/xpu/lib", smi, ml:
			return true
		default:
			return false
		}
	}
	mounts := buildKunlunxinXpuRuntimeMounts("/usr/local/xpu", smi, exists, nil)
	require.Len(t, mounts, 3)
	assert.Equal(t, "/usr/local/xpu", mounts[0].HostPath)
	assert.Equal(t, smi, mounts[1].HostPath)
	assert.Equal(t, ml, mounts[2].HostPath)
	for _, m := range mounts {
		assert.Equal(t, m.HostPath, m.ContainerPath)
		assert.True(t, m.Readonly)
	}
}

func TestCollectSymlinkMountPathsReadlinkF(t *testing.T) {
	link := "/lib/x86_64-linux-gnu/libxpunvidia-ml.so.1"
	final := "/lib/x86_64-linux-gnu/libxpunvidia-ml.so.1.0.0"
	readlink := func(p string) (string, error) {
		if p == link {
			return final, nil // RemoteReadlink: readlink -f
		}
		return "", os.ErrInvalid
	}
	paths := collectSymlinkMountPaths(link, readlink)
	assert.Equal(t, []string{link, final}, paths)

	assert.Equal(t, []string{link}, collectSymlinkMountPaths(link, nil))
	same := func(p string) (string, error) { return p, nil }
	assert.Equal(t, []string{link}, collectSymlinkMountPaths(link, same))
}

func TestBuildKunlunxinXpuRuntimeMountsFollowsSymlink(t *testing.T) {
	smi := "/usr/local/bin/xpu-smi"
	link := "/lib/x86_64-linux-gnu/libxpunvidia-ml.so.1"
	real := "/lib/x86_64-linux-gnu/libxpunvidia-ml.so.1.0.0"
	exists := func(p string) bool {
		switch p {
		case "/usr/local/xpu", smi, link, real:
			return true
		default:
			return false
		}
	}
	readlink := func(p string) (string, error) {
		if p == link {
			return real, nil
		}
		return "", os.ErrInvalid
	}
	mounts := buildKunlunxinXpuRuntimeMounts("/usr/local/xpu", smi, exists, readlink)
	require.Len(t, mounts, 4)
	assert.Equal(t, "/usr/local/xpu", mounts[0].HostPath)
	assert.Equal(t, smi, mounts[1].HostPath)
	assert.Equal(t, link, mounts[2].HostPath)
	assert.Equal(t, real, mounts[3].HostPath)
}

func TestParseKunlunxinXpuNodeIndex(t *testing.T) {
	idx, ok := parseKunlunxinXpuNodeIndex("xpu3")
	assert.True(t, ok)
	assert.Equal(t, 3, idx)
	_, ok = parseKunlunxinXpuNodeIndex("xpuctrl")
	assert.False(t, ok)
}

func TestKunlunxinXpuSmiPathDefault(t *testing.T) {
	orig := options.HostOptions.KunlunxinXpuSmiPath
	defer func() { options.HostOptions.KunlunxinXpuSmiPath = orig }()
	options.HostOptions.KunlunxinXpuSmiPath = ""
	assert.Equal(t, "/usr/local/bin/xpu-smi", kunlunxinXpuSmiPath())
	options.HostOptions.KunlunxinXpuSmiPath = "/opt/bin/xpu-smi"
	assert.Equal(t, "/opt/bin/xpu-smi", kunlunxinXpuSmiPath())
}
