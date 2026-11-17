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

package models

import (
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestBuildIsolatedDeviceConfigsSharingModes(t *testing.T) {
	devs := api.Devices{
		{Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_UNLIMITED},
		{Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_HAMI, MemoryMb: 8192},
	}
	out, err := BuildIsolatedDeviceConfigs(&devs, 0)
	if err != nil {
		t.Fatalf("BuildIsolatedDeviceConfigs: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d", len(out))
	}
	if out[0].SharingMode != computeapi.DEVICE_SHARING_MODE_UNLIMITED {
		t.Fatalf("dev0 SharingMode = %q", out[0].SharingMode)
	}
	if out[1].SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI || out[1].MemoryRequest != 8192 {
		t.Fatalf("dev1 = %#v", out[1])
	}
}

func TestBuildIsolatedDeviceConfigsHAMISplitClaim(t *testing.T) {
	devs := api.Devices{{Model: "A100"}, {Model: "A100"}}
	claim := 40960
	out, err := BuildIsolatedDeviceConfigs(&devs, claim)
	if err != nil {
		t.Fatalf("BuildIsolatedDeviceConfigs: %v", err)
	}
	perDev := (claim + len(devs) - 1) / len(devs)
	for i := range out {
		if out[i].SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI {
			t.Fatalf("dev%d SharingMode = %q", i, out[i].SharingMode)
		}
		if out[i].MemoryRequest != perDev {
			t.Fatalf("dev%d MemoryRequest = %d want %d", i, out[i].MemoryRequest, perDev)
		}
	}
}

func TestBuildIsolatedDeviceConfigsEmpty(t *testing.T) {
	out, err := BuildIsolatedDeviceConfigs(nil, 0)
	if err != nil || out != nil {
		t.Fatalf("nil devices: out=%v err=%v", out, err)
	}
	empty := api.Devices{}
	out, err = BuildIsolatedDeviceConfigs(&empty, 0)
	if err != nil || out != nil {
		t.Fatalf("empty devices: out=%v err=%v", out, err)
	}
}

func TestIsolatedDevicesNeedSync(t *testing.T) {
	desired := []*computeapi.IsolatedDeviceConfig{
		{Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_HAMI, MemoryRequest: 8192},
	}
	boundSame := []computeapi.SIsolatedDevice{
		{Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_HAMI},
	}
	if isolatedDevicesNeedSync(desired, boundSame) {
		t.Fatal("same model/sharing_mode should not need sync")
	}

	boundUnlimited := []computeapi.SIsolatedDevice{
		{Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_UNLIMITED},
	}
	if !isolatedDevicesNeedSync(desired, boundUnlimited) {
		t.Fatal("UNLIMITED -> HAMI should need sync")
	}

	if !isolatedDevicesNeedSync(desired, nil) {
		t.Fatal("empty bound should need sync when desired non-empty")
	}
	if !isolatedDevicesNeedSync(nil, boundUnlimited) {
		t.Fatal("empty desired should need sync when bound non-empty")
	}
	if isolatedDevicesNeedSync(nil, nil) {
		t.Fatal("both empty should not need sync")
	}
}

func TestBuildSchedulableIsolatedDevices(t *testing.T) {
	devs := api.Devices{{Model: "A100"}, {Model: "A100"}}
	claim := 40960
	iso, perDev, err := buildSchedulableIsolatedDevices(&devs, isolatedDeviceVramPlan{HamiClaimMb: claim})
	if err != nil {
		t.Fatalf("buildSchedulableIsolatedDevices: %v", err)
	}
	if len(iso) != 2 {
		t.Fatalf("len = %d want 2", len(iso))
	}
	wantPer := (claim + len(devs) - 1) / len(devs)
	if perDev != wantPer {
		t.Fatalf("perDev = %d want %d", perDev, wantPer)
	}
	for i, d := range iso {
		if d.DevType != computeapi.GPU_TYPE {
			t.Fatalf("dev%d DevType = %q", i, d.DevType)
		}
		if d.SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI {
			t.Fatalf("dev%d SharingMode = %q", i, d.SharingMode)
		}
		if d.MemoryMb != wantPer || d.MemoryRequest != wantPer {
			t.Fatalf("dev%d MemoryMb=%d MemoryRequest=%d want %d", i, d.MemoryMb, d.MemoryRequest, wantPer)
		}
	}
}

func TestBuildSchedulableIsolatedDevicesEmpty(t *testing.T) {
	iso, perDev, err := buildSchedulableIsolatedDevices(nil, isolatedDeviceVramPlan{})
	if err != nil || iso != nil || perDev != 0 {
		t.Fatalf("nil devices: iso=%v perDev=%d err=%v", iso, perDev, err)
	}
	empty := api.Devices{}
	iso, perDev, err = buildSchedulableIsolatedDevices(&empty, isolatedDeviceVramPlan{})
	if err != nil || iso != nil || perDev != 0 {
		t.Fatalf("empty devices: iso=%v perDev=%d err=%v", iso, perDev, err)
	}
}

func TestSkuForecastDisks(t *testing.T) {
	if skuForecastDisks(nil) != nil {
		t.Fatal("nil volumes")
	}
	empty := api.Volumes{}
	if skuForecastDisks(&empty) != nil {
		t.Fatal("empty volumes")
	}
	vols := api.Volumes{{SizeMB: 10240}, {SizeMB: 20480}}
	disks := skuForecastDisks(&vols)
	if len(disks) != 2 || disks[0].SizeMb != 10240 || disks[1].SizeMb != 20480 {
		t.Fatalf("disks = %#v", disks)
	}
	if disks[0].DiskType != "data" || disks[1].Index != 1 {
		t.Fatalf("disk fields = %#v", disks)
	}
}

func TestBuildIsolatedDeviceConfigsDefaultHAMIGpu(t *testing.T) {
	devs := api.Devices{{Model: "A100"}}
	claim := 20480
	out, err := BuildIsolatedDeviceConfigs(&devs, claim)
	if err != nil {
		t.Fatalf("BuildIsolatedDeviceConfigs: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len = %d", len(out))
	}
	if out[0].DevType != computeapi.GPU_TYPE {
		t.Fatalf("DevType = %q", out[0].DevType)
	}
	if out[0].SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI {
		t.Fatalf("SharingMode = %q", out[0].SharingMode)
	}
	if out[0].MemoryMb != claim || out[0].MemoryRequest != claim {
		t.Fatalf("MemoryMb=%d MemoryRequest=%d want %d", out[0].MemoryMb, out[0].MemoryRequest, claim)
	}
}

func TestGroupIsolatedDeviceAttachConfigs(t *testing.T) {
	desired := []*computeapi.IsolatedDeviceConfig{
		{Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_HAMI, MemoryRequest: 8192},
		{Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_HAMI, MemoryRequest: 8192},
		{Model: "A100", SharingMode: computeapi.DEVICE_SHARING_MODE_UNLIMITED},
	}
	groups := groupIsolatedDeviceAttachConfigs(desired)
	if len(groups) != 2 {
		t.Fatalf("groups len = %d want 2", len(groups))
	}
	if groups[0].Count != 2 || groups[0].SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI || groups[0].MemoryRequest != 8192 {
		t.Fatalf("group0 = %#v", groups[0])
	}
	if groups[1].Count != 1 || groups[1].SharingMode != computeapi.DEVICE_SHARING_MODE_UNLIMITED {
		t.Fatalf("group1 = %#v", groups[1])
	}
}

func TestWholeCardScheduleMemoryUsesUtilizationNotKV(t *testing.T) {
	// weight*1.2+2GiB fits in 2x16311 MiB at 0.9 (GPUStack). Adding the KV
	// reserve and splitting evenly asks each card for 18519 MiB, which a
	// 16311 MiB RTX 5060 Ti cannot satisfy.
	const (
		modelClaimMb = 28846
		kvReserveMb  = 8192
		cardMb       = 16311
	)
	withKVPerCard := (modelClaimMb + kvReserveMb + 1) / 2
	if withKVPerCard != 18519 || withKVPerCard <= cardMb {
		t.Fatalf("fixture per-card KV split = %d, want 18519 > %d", withKVPerCard, cardMb)
	}
	want := wholeCardScheduleMemoryMb(modelClaimMb, 2, 0.9)
	if want >= cardMb {
		t.Fatalf("GPUStack floor %d should fit in %d MiB", want, cardMb)
	}
	if want != 16026 {
		t.Fatalf("ceil(28846/(2*0.9)) = %d, want 16026", want)
	}

	devs := api.Devices{
		{Model: "GeForce RTX 5060 Ti", SharingMode: computeapi.DEVICE_SHARING_MODE_UNLIMITED},
		{Model: "GeForce RTX 5060 Ti", SharingMode: computeapi.DEVICE_SHARING_MODE_UNLIMITED},
	}
	plan := isolatedDeviceVramPlan{
		HamiClaimMb:          modelClaimMb + kvReserveMb,
		WholeCardClaimMb:     modelClaimMb,
		WholeCardUtilization: 0.9,
	}
	out, err := buildIsolatedDeviceConfigs(&devs, plan)
	if err != nil {
		t.Fatalf("buildIsolatedDeviceConfigs: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d", len(out))
	}
	for i, d := range out {
		if d.MemoryMb != want || d.MemoryRequest != want {
			t.Fatalf("dev%d MemoryMb=%d MemoryRequest=%d want %d", i, d.MemoryMb, d.MemoryRequest, want)
		}
	}
}

func TestWholeCardExplicitMemoryMbWins(t *testing.T) {
	devs := api.Devices{{
		Model:       "GeForce RTX 5060 Ti",
		SharingMode: computeapi.DEVICE_SHARING_MODE_UNLIMITED,
		MemoryMb:    8192,
	}}
	out, err := buildIsolatedDeviceConfigs(&devs, isolatedDeviceVramPlan{
		WholeCardClaimMb:     28846,
		WholeCardUtilization: 0.9,
	})
	if err != nil {
		t.Fatalf("buildIsolatedDeviceConfigs: %v", err)
	}
	if out[0].MemoryMb != 8192 {
		t.Fatalf("explicit memory_mb = %d, want 8192", out[0].MemoryMb)
	}
}

func TestHAMISplitIgnoresGpuMemoryUtilization(t *testing.T) {
	devs := api.Devices{{Model: "A100"}, {Model: "A100"}}
	claim := 40960
	out, err := buildIsolatedDeviceConfigs(&devs, isolatedDeviceVramPlan{
		HamiClaimMb:          claim,
		WholeCardClaimMb:     10000,
		WholeCardUtilization: 0.9,
	})
	if err != nil {
		t.Fatalf("buildIsolatedDeviceConfigs: %v", err)
	}
	want := (claim + len(devs) - 1) / len(devs)
	for i, d := range out {
		if d.SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI {
			t.Fatalf("dev%d SharingMode = %q", i, d.SharingMode)
		}
		if d.MemoryMb != want || d.MemoryRequest != want {
			t.Fatalf("dev%d MemoryMb=%d want %d (HAMI must not divide by utilization)", i, d.MemoryMb, want)
		}
	}
}
