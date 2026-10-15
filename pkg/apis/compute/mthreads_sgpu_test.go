package compute

import "testing"

func TestAlignMthreadsSgpuMemoryMiB(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 0},
		{-1, 0},
		{500, 512},
		{512, 512},
		{1000, 1024},
		{40960, 48 * 1024},
		{81920, 80 * 1024},
		{90000, 90000},
	}
	for _, tc := range cases {
		if got := AlignMthreadsSgpuMemoryMiB(tc.in); got != tc.want {
			t.Fatalf("AlignMthreadsSgpuMemoryMiB(%d)=%d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestMthreadsSgpuMemoryLimitBytes(t *testing.T) {
	if got := MthreadsSgpuMemoryLimitBytes(2048); got != 2147483648 {
		t.Fatalf("bytes=%d, want 2147483648", got)
	}
	if got := MthreadsSgpuMemoryLimitBytes(0); got != 0 {
		t.Fatalf("zero bytes=%d", got)
	}
}

func TestCountMthreadsHamiDevices(t *testing.T) {
	devs := []*IsolatedDeviceConfig{
		{Vendor: "MTHREADS", SharingMode: DEVICE_SHARING_MODE_HAMI},
		{Vendor: MTHREADS_VENDOR_ID, SharingMode: DEVICE_SHARING_MODE_HAMI},
		{Vendor: "MTHREADS", SharingMode: DEVICE_SHARING_MODE_EXCLUSIVE},
		{Vendor: "NVIDIA", SharingMode: DEVICE_SHARING_MODE_HAMI},
	}
	if got := CountMthreadsHamiDevices(devs); got != 2 {
		t.Fatalf("count=%d, want 2", got)
	}
}

func TestMthreadsSgpuComputingPowerWeight(t *testing.T) {
	if got := MthreadsSgpuComputingPowerWeight(0); got != 1 {
		t.Fatalf("weight=%d, want 1", got)
	}
	if got := MthreadsSgpuComputingPowerWeight(25); got != 25 {
		t.Fatalf("weight=%d, want 25", got)
	}
	if got := MthreadsSgpuComputingPowerWeight(100000); got != 99999 {
		t.Fatalf("weight=%d, want 99999", got)
	}
}
