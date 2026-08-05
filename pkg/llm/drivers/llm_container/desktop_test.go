package llm_container

import (
	"testing"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

func TestAppendContainerIsolatedDevicesFromSku(t *testing.T) {
	devices := api.Devices{{Model: "GeForce RTX 4090"}}
	sku := &models.SLLMSku{}
	sku.Devices = &devices
	spec := computeapi.ContainerSpec{}
	appendContainerIsolatedDevices(&spec, nil, sku, nil)
	if len(spec.Devices) != 1 {
		t.Fatalf("devices len = %d", len(spec.Devices))
	}
	iso := spec.Devices[0].IsolatedDevice
	if iso == nil || iso.Index == nil || *iso.Index != 0 {
		t.Fatalf("device index = %#v", iso)
	}
	if iso.GuestIsolatedDeviceIndex != 0 {
		t.Fatalf("guest_isolated_device_index = %d", iso.GuestIsolatedDeviceIndex)
	}
}

func TestAppendContainerIsolatedDevicesById(t *testing.T) {
	devices := api.Devices{{Model: "GeForce RTX 4090"}}
	sku := &models.SLLMSku{}
	sku.Devices = &devices
	dev := computeapi.SIsolatedDevice{}
	dev.Id = "gpu-1"
	spec := computeapi.ContainerSpec{}
	appendContainerIsolatedDevices(&spec, nil, sku, []computeapi.SIsolatedDevice{dev})
	if len(spec.Devices) != 1 || spec.Devices[0].IsolatedDevice == nil || spec.Devices[0].IsolatedDevice.Id != "gpu-1" {
		t.Fatalf("devices = %#v", spec.Devices)
	}
	if spec.Devices[0].IsolatedDevice.GuestIsolatedDeviceIndex != 0 {
		t.Fatalf("guest_isolated_device_index = %d", spec.Devices[0].IsolatedDevice.GuestIsolatedDeviceIndex)
	}
}

func TestAppendContainerIsolatedDevicesEmptySkuIgnoresBound(t *testing.T) {
	dev := computeapi.SIsolatedDevice{}
	dev.Id = "gpu-stale"
	spec := computeapi.ContainerSpec{}
	appendContainerIsolatedDevices(&spec, nil, nil, []computeapi.SIsolatedDevice{dev})
	if len(spec.Devices) != 0 {
		t.Fatalf("expected no devices when effective devices empty, got %#v", spec.Devices)
	}
	sku := &models.SLLMSku{}
	empty := api.Devices{}
	sku.Devices = &empty
	spec2 := computeapi.ContainerSpec{}
	appendContainerIsolatedDevices(&spec2, nil, sku, []computeapi.SIsolatedDevice{dev})
	if len(spec2.Devices) != 0 {
		t.Fatalf("expected no devices for empty sku devices, got %#v", spec2.Devices)
	}
}

func TestDesktopUiTitle(t *testing.T) {
	if got := desktopUiTitle(&api.LLMImageDesktopConfig{UiTitle: "Custom"}); got != "Custom" {
		t.Fatalf("ui_title = %q", got)
	}
	if got := desktopUiTitle(nil); got != "Cloudpods Desktop" {
		t.Fatalf("default ui_title = %q", got)
	}
}

func TestDesktopHasIsolatedGPU(t *testing.T) {
	if desktopHasIsolatedGPU(nil, nil, nil) {
		t.Fatal("expected false without devices")
	}
	bound := computeapi.SIsolatedDevice{}
	bound.Id = "gpu-stale"
	if desktopHasIsolatedGPU(nil, nil, []computeapi.SIsolatedDevice{bound}) {
		t.Fatal("expected false when only stale bound devices and empty effective devices")
	}
	devices := api.Devices{{Model: "GeForce RTX 4090"}}
	sku := &models.SLLMSku{}
	sku.Devices = &devices
	if !desktopHasIsolatedGPU(nil, sku, nil) {
		t.Fatal("expected true when sku has devices")
	}
}

func TestDesktopGPUWaylandEnvs(t *testing.T) {
	envs := desktopGPUWaylandEnvs()
	m := make(map[string]string, len(envs))
	for _, e := range envs {
		m[e.Key] = e.Value
	}
	if m["PIXELFLUX_WAYLAND"] != "true" {
		t.Fatalf("PIXELFLUX_WAYLAND = %q", m["PIXELFLUX_WAYLAND"])
	}
	if m["DRINODE"] != desktopDefaultDRINode || m["DRI_NODE"] != desktopDefaultDRINode {
		t.Fatalf("dri envs = %#v", m)
	}
}

func TestDesktopWebtopCommonEnvsSelkiesSidebar(t *testing.T) {
	desktopEnvs := desktopWebtopCommonEnvs("llm-1", "Title", true)
	agentEnvs := desktopWebtopCommonEnvs("llm-2", "Title", false)
	assertSelkiesSidebar := func(t *testing.T, envs []*commonapi.ContainerKeyValue, apps, gamepads string) {
		t.Helper()
		m := make(map[string]string, len(envs))
		for _, e := range envs {
			m[e.Key] = e.Value
		}
		if m["SELKIES_UI_SIDEBAR_SHOW_APPS"] != apps {
			t.Fatalf("SELKIES_UI_SIDEBAR_SHOW_APPS = %q, want %q", m["SELKIES_UI_SIDEBAR_SHOW_APPS"], apps)
		}
		if m["SELKIES_UI_SIDEBAR_SHOW_GAMEPADS"] != gamepads {
			t.Fatalf("SELKIES_UI_SIDEBAR_SHOW_GAMEPADS = %q, want %q", m["SELKIES_UI_SIDEBAR_SHOW_GAMEPADS"], gamepads)
		}
	}
	assertSelkiesSidebar(t, desktopEnvs, "True", "True")
	assertSelkiesSidebar(t, agentEnvs, "False", "False")
}

func TestAppendDesktopExtraEnvs(t *testing.T) {
	envs := appendDesktopExtraEnvs(nil, map[string]string{"FOO": "bar"})
	if len(envs) != 1 || envs[0].Key != "FOO" || envs[0].Value != "bar" {
		t.Fatalf("envs = %#v", envs)
	}
}
