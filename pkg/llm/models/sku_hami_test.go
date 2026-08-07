package models

import (
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/utils/vram"
)

func TestNormalizeLLMSkuDeviceDefaults(t *testing.T) {
	dev := api.Device{}
	normalizeLLMSkuDevice(&dev)
	if dev.DevType != computeapi.GPU_TYPE {
		t.Fatalf("DevType = %q, want %q", dev.DevType, computeapi.GPU_TYPE)
	}
	if dev.SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI {
		t.Fatalf("SharingMode = %q, want %q", dev.SharingMode, computeapi.DEVICE_SHARING_MODE_HAMI)
	}
}

func TestNormalizeLLMSkuDeviceLegacyTypes(t *testing.T) {
	cases := []struct {
		name            string
		in              api.Device
		wantDevType     string
		wantSharingMode string
		wantVendor      string
	}{
		{
			name:            "NVIDIA_GPU_SHARE",
			in:              api.Device{DevType: computeapi.CONTAINER_DEV_NVIDIA_GPU_SHARE},
			wantDevType:     computeapi.GPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_UNLIMITED,
			wantVendor:      "NVIDIA",
		},
		{
			name:            "NVIDIA_MPS",
			in:              api.Device{DevType: computeapi.CONTAINER_DEV_NVIDIA_MPS},
			wantDevType:     computeapi.GPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_MPS,
			wantVendor:      "NVIDIA",
		},
		{
			name:            "NVIDIA_GPU",
			in:              api.Device{DevType: computeapi.CONTAINER_DEV_NVIDIA_GPU},
			wantDevType:     computeapi.GPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
			wantVendor:      "NVIDIA",
		},
		{
			name:            "NVIDIA_HAMI",
			in:              api.Device{DevType: computeapi.CONTAINER_DEV_NVIDIA_HAMI},
			wantDevType:     computeapi.GPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_HAMI,
			wantVendor:      "NVIDIA",
		},
		{
			name:            "HYGON_DCU",
			in:              api.Device{DevType: computeapi.CONTAINER_DEV_HYGON_DCU},
			wantDevType:     computeapi.GPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
			wantVendor:      "HYGON",
		},
		{
			name:            "HYGON_DCU_HAMI",
			in:              api.Device{DevType: computeapi.CONTAINER_DEV_HYGON_DCU_HAMI},
			wantDevType:     computeapi.GPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_HAMI,
			wantVendor:      "HYGON",
		},
		{
			name:            "ASCEND_NPU",
			in:              api.Device{DevType: computeapi.CONTAINER_DEV_ASCEND_NPU},
			wantDevType:     computeapi.NPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
			wantVendor:      "ASCEND",
		},
		{
			name:            "ASCEND_NPU_HAMI",
			in:              api.Device{DevType: computeapi.CONTAINER_DEV_ASCEND_NPU_HAMI},
			wantDevType:     computeapi.NPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_HAMI,
			wantVendor:      "ASCEND",
		},
		{
			name:            "ASCEND vendor empty DevType defaults to NPU",
			in:              api.Device{Vendor: "ASCEND", Model: "910B2"},
			wantDevType:     computeapi.NPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_HAMI,
			wantVendor:      "ASCEND",
		},
		{
			name:            "ASCEND vendor corrects stored GPU to NPU",
			in:              api.Device{DevType: computeapi.GPU_TYPE, Vendor: "ascend", Model: "910B2"},
			wantDevType:     computeapi.NPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_HAMI,
			wantVendor:      "ASCEND",
		},
		{
			name:            "explicit sharing_mode preserved",
			in:              api.Device{DevType: computeapi.CONTAINER_DEV_NVIDIA_GPU_SHARE, SharingMode: computeapi.DEVICE_SHARING_MODE_MPS},
			wantDevType:     computeapi.GPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_MPS,
			wantVendor:      "NVIDIA",
		},
		{
			name:            "explicit vendor preserved and canonicalized",
			in:              api.Device{DevType: computeapi.GPU_TYPE, Vendor: "hygon", Model: "BW"},
			wantDevType:     computeapi.GPU_TYPE,
			wantSharingMode: computeapi.DEVICE_SHARING_MODE_HAMI,
			wantVendor:      "HYGON",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dev := tc.in
			normalizeLLMSkuDevice(&dev)
			if dev.DevType != tc.wantDevType {
				t.Fatalf("DevType = %q, want %q", dev.DevType, tc.wantDevType)
			}
			if dev.SharingMode != tc.wantSharingMode {
				t.Fatalf("SharingMode = %q, want %q", dev.SharingMode, tc.wantSharingMode)
			}
			if dev.Vendor != tc.wantVendor {
				t.Fatalf("Vendor = %q, want %q", dev.Vendor, tc.wantVendor)
			}
		})
	}
}

func TestNormalizeLLMSkuDevicesAllowsHAMIWithoutStoredVram(t *testing.T) {
	devs := api.Devices{{Model: "A100"}}
	if err := normalizeLLMSkuDevices(&devs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if devs[0].SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI {
		t.Fatalf("SharingMode = %q", devs[0].SharingMode)
	}
}

func TestEffectiveMaxModelLen(t *testing.T) {
	if got := effectiveMaxModelLen(nil); got != api.LLM_DEFAULT_CONTEXT_TOKENS {
		t.Fatalf("nil sku → %d", got)
	}
	sku := &SLLMSku{LLMType: string(api.LLM_CONTAINER_VLLM)}
	if got := effectiveMaxModelLen(sku); got != api.LLM_DEFAULT_CONTEXT_TOKENS {
		t.Fatalf("default → %d", got)
	}
	sku.BackendParameters = []string{"--max-model-len=4096"}
	if got := effectiveMaxModelLen(sku); got != 4096 {
		t.Fatalf("backend param → %d, want 4096", got)
	}
}

func TestEstimateVramClaimMbFromInstantModels(t *testing.T) {
	models := map[string]SInstantModel{
		"a": {WeightSizeBytes: 1024 * 1024 * 100},  // 100 MiB weights
		"b": {WeightSizeBytes: 1024 * 1024 * 1000}, // 1000 MiB weights
	}
	maxLen := api.LLM_DEFAULT_CONTEXT_TOKENS
	got := EstimateVramClaimMbFromInstantModels(string(api.LLM_CONTAINER_VLLM), []string{"a", "b"}, models, maxLen)
	want := vram.EstimateClaimMbWithContext(1024*1024*1000, string(api.LLM_CONTAINER_VLLM), maxLen)
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
	if EstimateVramClaimMbFromInstantModels(string(api.LLM_CONTAINER_VLLM), nil, models, maxLen) != 0 {
		t.Fatal("empty model ids should yield 0")
	}

	base := vram.EstimateClaimMb(1024*1024*1000, string(api.LLM_CONTAINER_VLLM))
	if want <= base {
		t.Fatalf("claim WithContext (%d) should exceed model-only EstimateClaimMb (%d)", want, base)
	}
}

func TestGetLLMBasePodCreateInputHAMIFields(t *testing.T) {
	devs := api.Devices{{Model: "A100"}}
	sku := &SLLMSkuBase{
		Cpu:     4,
		Memory:  8192,
		Volumes: &api.Volumes{{SizeMB: 10240}},
		Devices: &devs,
	}
	llm := &SLLMBase{}
	input := &api.LLMBaseCreateInput{}
	input.Name = "llm-hami-test"
	input.AutoStart = true
	input.ProjectId = "proj-test"
	input.Nets = []*computeapi.NetworkConfig{{Network: "net1"}}

	out, err := GetLLMBasePodCreateInput(nil, nil, input, llm, sku, 20480, "")
	if err != nil {
		t.Fatalf("GetLLMBasePodCreateInput: %v", err)
	}
	if len(out.IsolatedDevices) != 1 {
		t.Fatalf("IsolatedDevices len = %d", len(out.IsolatedDevices))
	}
	dev := out.IsolatedDevices[0]
	if dev.DevType != computeapi.GPU_TYPE {
		t.Fatalf("DevType = %q", dev.DevType)
	}
	if dev.SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI {
		t.Fatalf("SharingMode = %q", dev.SharingMode)
	}
	if dev.MemoryMb != 20480 || dev.MemoryRequest != 20480 {
		t.Fatalf("MemoryMb=%d MemoryRequest=%d", dev.MemoryMb, dev.MemoryRequest)
	}
}

func TestGetLLMBasePodCreateInputHAMIRequiresClaim(t *testing.T) {
	devs := api.Devices{{Model: "A100"}}
	sku := &SLLMSkuBase{
		Cpu:     4,
		Memory:  8192,
		Volumes: &api.Volumes{{SizeMB: 10240}},
		Devices: &devs,
	}
	llm := &SLLMBase{}
	input := &api.LLMBaseCreateInput{}
	input.Name = "llm-hami-test"
	input.AutoStart = true
	input.ProjectId = "proj-test"
	input.Nets = []*computeapi.NetworkConfig{{Network: "net1"}}

	if _, err := GetLLMBasePodCreateInput(nil, nil, input, llm, sku, 0, ""); err == nil {
		t.Fatal("expected error when HAMI with zero vram claim")
	}
}

func TestGetLLMBasePodCreateInputHAMIManualMemoryMb(t *testing.T) {
	devs := api.Devices{{Model: "A100", MemoryMb: 8192}}
	sku := &SLLMSkuBase{
		Cpu:     4,
		Memory:  8192,
		Volumes: &api.Volumes{{SizeMB: 10240}},
		Devices: &devs,
	}
	llm := &SLLMBase{}
	input := &api.LLMBaseCreateInput{}
	input.Name = "llm-hami-manual-mem"
	input.AutoStart = true
	input.ProjectId = "proj-test"
	input.Nets = []*computeapi.NetworkConfig{{Network: "net1"}}

	// Manual memory_mb allows claim=0.
	out, err := GetLLMBasePodCreateInput(nil, nil, input, llm, sku, 0, "")
	if err != nil {
		t.Fatalf("GetLLMBasePodCreateInput: %v", err)
	}
	if len(out.IsolatedDevices) != 1 {
		t.Fatalf("IsolatedDevices len = %d", len(out.IsolatedDevices))
	}
	dev := out.IsolatedDevices[0]
	if dev.MemoryMb != 8192 || dev.MemoryRequest != 8192 {
		t.Fatalf("MemoryMb=%d MemoryRequest=%d want 8192", dev.MemoryMb, dev.MemoryRequest)
	}

	// Manual memory_mb wins over claim split.
	out2, err := GetLLMBasePodCreateInput(nil, nil, input, llm, sku, 40960, "")
	if err != nil {
		t.Fatalf("GetLLMBasePodCreateInput with claim: %v", err)
	}
	if out2.IsolatedDevices[0].MemoryRequest != 8192 {
		t.Fatalf("manual memory should win over claim, got %d", out2.IsolatedDevices[0].MemoryRequest)
	}
}

func TestGetLLMBasePodCreateInputHAMIMixedMemoryFallback(t *testing.T) {
	devs := api.Devices{
		{Model: "A100", MemoryMb: 10240},
		{Model: "A100"},
	}
	sku := &SLLMSkuBase{
		Cpu:     4,
		Memory:  8192,
		Volumes: &api.Volumes{{SizeMB: 10240}},
		Devices: &devs,
	}
	llm := &SLLMBase{}
	input := &api.LLMBaseCreateInput{}
	input.Name = "llm-hami-mixed"
	input.AutoStart = true
	input.ProjectId = "proj-test"
	input.Nets = []*computeapi.NetworkConfig{{Network: "net1"}}

	claim := 40960
	perDevFromClaim := (claim + len(devs) - 1) / len(devs)
	out, err := GetLLMBasePodCreateInput(nil, nil, input, llm, sku, claim, "")
	if err != nil {
		t.Fatalf("GetLLMBasePodCreateInput: %v", err)
	}
	if out.IsolatedDevices[0].MemoryRequest != 10240 {
		t.Fatalf("device0 MemoryRequest = %d want 10240", out.IsolatedDevices[0].MemoryRequest)
	}
	if out.IsolatedDevices[1].MemoryRequest != perDevFromClaim {
		t.Fatalf("device1 MemoryRequest = %d want %d", out.IsolatedDevices[1].MemoryRequest, perDevFromClaim)
	}
}

func TestLLMPodIsolatedDeviceConfigFromSKU(t *testing.T) {
	devs := api.Devices{
		{Model: "A100"},
		{DevType: computeapi.CONTAINER_DEV_NVIDIA_GPU_SHARE, Model: "A100"},
	}
	vramClaimMb := 40960
	perDev := (vramClaimMb + len(devs) - 1) / len(devs)

	out := make([]*computeapi.IsolatedDeviceConfig, 0, len(devs))
	for i := range devs {
		normalizeLLMSkuDevice(&devs[i])
		out = append(out, &computeapi.IsolatedDeviceConfig{
			DevType:       devs[i].DevType,
			SharingMode:   devs[i].SharingMode,
			Vendor:        devs[i].Vendor,
			Model:         devs[i].Model,
			DevicePath:    devs[i].DevicePath,
			MemoryMb:      perDev,
			MemoryRequest: perDev,
			SmUtilLimit:   devs[i].SmUtilLimit,
		})
	}
	if out[0].DevType != computeapi.GPU_TYPE || out[0].SharingMode != computeapi.DEVICE_SHARING_MODE_HAMI {
		t.Fatalf("device0 = %#v", out[0])
	}
	if out[1].DevType != computeapi.GPU_TYPE || out[1].SharingMode != computeapi.DEVICE_SHARING_MODE_UNLIMITED {
		t.Fatalf("device1 = %#v", out[1])
	}
	if out[1].Vendor != "NVIDIA" {
		t.Fatalf("device1 Vendor = %q, want NVIDIA", out[1].Vendor)
	}
	if out[0].MemoryRequest != perDev || out[1].MemoryRequest != perDev {
		t.Fatalf("MemoryRequest = %d,%d want %d", out[0].MemoryRequest, out[1].MemoryRequest, perDev)
	}
}

func TestBuildIsolatedDeviceMemoryParamsDefaultsHAMI(t *testing.T) {
	params := buildIsolatedDeviceMemoryParams(api.Device{Model: "A100"})
	if params.Contains("unused") {
		t.Fatal("HAMI params should not set unused")
	}
	devTypes, err := params.GetArray("dev_type")
	if err != nil || len(devTypes) != 1 {
		t.Fatalf("dev_type = %v err=%v", params, err)
	}
	if s, _ := devTypes[0].GetString(); s != computeapi.GPU_TYPE {
		t.Fatalf("dev_type = %q", s)
	}
	modes, err := params.GetArray("sharing_mode")
	if err != nil || len(modes) != 1 {
		t.Fatalf("sharing_mode = %v err=%v", params, err)
	}
	if s, _ := modes[0].GetString(); s != computeapi.DEVICE_SHARING_MODE_HAMI {
		t.Fatalf("sharing_mode = %q", s)
	}
}

func TestBuildIsolatedDeviceMemoryParamsExclusiveUsesUnused(t *testing.T) {
	params := buildIsolatedDeviceMemoryParams(api.Device{
		DevType:     computeapi.GPU_TYPE,
		SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
		Model:       "A100",
	})
	if !params.Contains("unused") {
		t.Fatal("exclusive params should set unused")
	}
}

func TestBuildIsolatedDeviceMemoryParamsVendor(t *testing.T) {
	params := buildIsolatedDeviceMemoryParams(api.Device{
		DevType:     computeapi.GPU_TYPE,
		SharingMode: computeapi.DEVICE_SHARING_MODE_EXCLUSIVE,
		Model:       "BW",
		Vendor:      "HYGON",
	})
	vendors, err := params.GetArray("vendor")
	if err != nil || len(vendors) != 1 {
		t.Fatalf("vendor = %v err=%v", params, err)
	}
	if s, _ := vendors[0].GetString(); s != "HYGON" {
		t.Fatalf("vendor = %q", s)
	}
}
