package llm_container

import (
	"testing"

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
	if spec.Devices[0].IsolatedDevice == nil || spec.Devices[0].IsolatedDevice.Index == nil || *spec.Devices[0].IsolatedDevice.Index != 0 {
		t.Fatalf("device index = %#v", spec.Devices[0].IsolatedDevice)
	}
}

func TestAppendContainerIsolatedDevicesById(t *testing.T) {
	dev := computeapi.SIsolatedDevice{}
	dev.Id = "gpu-1"
	spec := computeapi.ContainerSpec{}
	appendContainerIsolatedDevices(&spec, nil, nil, []computeapi.SIsolatedDevice{dev})
	if len(spec.Devices) != 1 || spec.Devices[0].IsolatedDevice == nil || spec.Devices[0].IsolatedDevice.Id != "gpu-1" {
		t.Fatalf("devices = %#v", spec.Devices)
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

func TestAppendDesktopExtraEnvs(t *testing.T) {
	envs := appendDesktopExtraEnvs(nil, map[string]string{"FOO": "bar"})
	if len(envs) != 1 || envs[0].Key != "FOO" || envs[0].Value != "bar" {
		t.Fatalf("envs = %#v", envs)
	}
}
