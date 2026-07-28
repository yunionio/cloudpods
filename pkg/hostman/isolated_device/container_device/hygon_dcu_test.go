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

	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
)

func TestParseHySmiDeviceIndices(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect []int
	}{
		{
			name: "default table",
			input: `HCU     Temp     AvgPwr     Perf     PwrCap     VRAM%      HCU%      Mode
0       61.0C    108.0W     auto     400.0W     0%         0.0%      Normal
1       62.0C    112.0W     auto     400.0W     0%         0.0%      Normal`,
			expect: []int{0, 1},
		},
		{
			name: "gpu bracket format",
			input: `GPU[0]    : VRAM Total Memory (B): 34342961152
GPU[1]    : VRAM Total Memory (B): 34342961152`,
			expect: []int{0, 1},
		},
		{
			name: "hcu vram meminfo format",
			input: `HCU[0]          : vram Total Memory (MiB): 65520
HCU[0]          : vram Total Used Memory (MiB): 2
HCU[1]          : vram Total Memory (MiB): 65520
HCU[7]          : vram Total Memory (MiB): 65520`,
			expect: []int{0, 1, 7},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseHySmiDeviceIndices(c.input)
			assert.Equal(t, c.expect, got)
		})
	}
}

func TestParseHySmiMemInfoVram(t *testing.T) {
	t.Run("gpu bytes format", func(t *testing.T) {
		input := `GPU[0]    : VRAM Total Memory (B): 34342961152
GPU[1]    : VRAM Total Memory (B): 34342961152`
		memMap := parseHySmiMemInfoVram(input)
		assert.Greater(t, memMap[0], 30000)
		assert.Greater(t, memMap[1], 30000)
	})
	t.Run("hcu vram meminfo format", func(t *testing.T) {
		input := `HCU[0]          : vram Total Memory (MiB): 65520
HCU[0]          : vram Total Used Memory (MiB): 2
HCU[1]          : vram Total Memory (MiB): 65520
HCU[7]          : vram Total Memory (MiB): 65520`
		memMap := parseHySmiMemInfoVram(input)
		assert.Equal(t, 65520, memMap[0])
		assert.Equal(t, 65520, memMap[1])
		assert.Equal(t, 65520, memMap[7])
		assert.NotContains(t, memMap, 2) // should not pick up "Used Memory" value
	})
}

func TestHygonModelNameFromPCIDevice(t *testing.T) {
	t.Run("from lspci model name", func(t *testing.T) {
		line := `09:00.0 "Display controller [0380]" "Chengdu Haiguang IC Design Co., Ltd. [1d94]" "Z100 [K100_AI] [6320]" -rxx "Chengdu Haiguang IC Design Co., Ltd. [1d94]" "Z100 [K100_AI] [6320]"`
		pciDev := isolated_device.NewPCIDevice2(line)
		assert.Equal(t, "K100_AI", hygonModelNameFromPCIDevice(pciDev))
	})
	t.Run("fallback to device name", func(t *testing.T) {
		pciDev := &isolated_device.PCIDevice{
			DeviceName: "Z100 [K100_AI]",
		}
		assert.Equal(t, "Z100 [K100_AI]", hygonModelNameFromPCIDevice(pciDev))
	})
	t.Run("fallback to default", func(t *testing.T) {
		assert.Equal(t, "Hygon DCU", hygonModelNameFromPCIDevice(nil))
		assert.Equal(t, "Hygon DCU", hygonModelNameFromPCIDevice(&isolated_device.PCIDevice{}))
	})
}

func TestParseHySmiVdeviceIndices(t *testing.T) {
	input := `Virtual Device 0:
 Actual Device: 0
 Compute units: 5
 Global memory: 4294967296 bytes
Virtual Device 1:
 Actual Device: 0
 Compute units: 15
 Global memory: 8589934592 bytes`
	indices := parseHySmiVdeviceIndices(input, 0)
	assert.Equal(t, []int{0, 1}, indices)
}

func TestHygonRenderPathFromLinkWithRemote(t *testing.T) {
	assert.Equal(t, "/dev/dri/renderD128", hygonRenderPathFromLinkWithRemote("../renderD128", false))
	assert.Equal(t, "/dev/dri/renderD128", hygonRenderPathFromLinkWithRemote("/dev/dri/renderD128", true))
}

func TestBuildHygonRenderPathMapFromEntry(t *testing.T) {
	entryName := "pci-0000:03:00.0-render"
	pciAddr, err := getGPUPCIAddr(entryName)
	assert.NoError(t, err)
	assert.Equal(t, "0000:03:00.0", pciAddr)
	renderPath := hygonRenderPathFromLinkWithRemote("/dev/dri/renderD128", true)
	assert.Equal(t, "/dev/dri/renderD128", renderPath)
}

func TestHygonRenderPathForHCUIndex(t *testing.T) {
	assert.Equal(t, "/dev/dri/renderD128", hygonRenderPathForHCUIndex(0))
	assert.Equal(t, "/dev/dri/renderD129", hygonRenderPathForHCUIndex(1))
	assert.Equal(t, "/dev/dri/renderD135", hygonRenderPathForHCUIndex(7))
}

func TestBuildHygonRenderPathToPCIMapFromEntries(t *testing.T) {
	// Sample by-path entries from Hygon node m06r2n19
	entries := []struct {
		entryName  string
		linkPath   string
		renderPath string
		pciAddr    string
	}{
		{"pci-0000:09:00.0-render", "../renderD128", "/dev/dri/renderD128", "0000:09:00.0"},
		{"pci-0000:36:00.0-render", "../renderD129", "/dev/dri/renderD129", "0000:36:00.0"},
		{"pci-0000:55:00.0-render", "../renderD130", "/dev/dri/renderD130", "0000:55:00.0"},
		{"pci-0000:77:00.0-render", "../renderD131", "/dev/dri/renderD131", "0000:77:00.0"},
		{"pci-0000:85:00.0-render", "../renderD132", "/dev/dri/renderD132", "0000:85:00.0"},
		{"pci-0000:b5:00.0-render", "../renderD133", "/dev/dri/renderD133", "0000:b5:00.0"},
		{"pci-0000:d5:00.0-render", "../renderD134", "/dev/dri/renderD134", "0000:d5:00.0"},
		{"pci-0000:f5:00.0-render", "../renderD135", "/dev/dri/renderD135", "0000:f5:00.0"},
	}

	pciToRender := map[string]string{}
	for _, e := range entries {
		pciAddr, err := getGPUPCIAddr(e.entryName)
		assert.NoError(t, err)
		assert.Equal(t, e.pciAddr, pciAddr)
		renderPath := hygonRenderPathFromLinkWithRemote(e.linkPath, false)
		assert.Equal(t, e.renderPath, renderPath)
		pciToRender[pciAddr] = renderPath
	}

	renderToPCI := buildHygonRenderPathToPCIMap(pciToRender)
	assert.Len(t, renderToPCI, 8)

	for hcuIdx := 0; hcuIdx < 8; hcuIdx++ {
		renderPath := hygonRenderPathForHCUIndex(hcuIdx)
		pciAddr, ok := renderToPCI[renderPath]
		assert.True(t, ok, "HCU[%d] render path %s should exist in map", hcuIdx, renderPath)
		assert.Equal(t, entries[hcuIdx].pciAddr, pciAddr)
	}
}
