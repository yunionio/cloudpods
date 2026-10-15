package compute

import "strings"

// sGPU memory buckets in MiB. The driver rounds a requested limit up to one of these.
var mthreadsSgpuMemoryBucketsMiB = []int{
	512,
	1 * 1024,
	2 * 1024,
	4 * 1024,
	8 * 1024,
	16 * 1024,
	32 * 1024,
	48 * 1024,
	64 * 1024,
	80 * 1024,
}

const (
	MthreadsSgpuMaxInstances       = 16
	MthreadsSgpuWeightMin          = 1
	MthreadsSgpuWeightMax          = 99999
	mthreadsSgpuBytesPerMiB  int64 = 1024 * 1024
)

// AlignMthreadsSgpuMemoryMiB rounds a requested size up to the next sGPU bucket.
// Zero stays zero. A request above the largest bucket is left unchanged.
func AlignMthreadsSgpuMemoryMiB(request int) int {
	if request <= 0 {
		return 0
	}
	for _, bucket := range mthreadsSgpuMemoryBucketsMiB {
		if request <= bucket {
			return bucket
		}
	}
	return request
}

// MthreadsSgpuMemoryLimitBytes is the byte value for MTHREADS_QOS_MEMORY_LIMIT.
func MthreadsSgpuMemoryLimitBytes(memoryMiB int) int64 {
	return int64(AlignMthreadsSgpuMemoryMiB(memoryMiB)) * mthreadsSgpuBytesPerMiB
}

// IsMthreadsHami reports a sliced Moore Threads GPU. Vendor may be the name or the PCI id.
func IsMthreadsHami(vendor, sharingMode string) bool {
	if sharingMode != DEVICE_SHARING_MODE_HAMI {
		return false
	}
	vendor = strings.TrimSpace(vendor)
	return strings.EqualFold(vendor, "MTHREADS") || strings.EqualFold(vendor, MTHREADS_VENDOR_ID)
}

// CountMthreadsHamiDevices counts sGPU slices. More than one in a guest is rejected.
func CountMthreadsHamiDevices(devs []*IsolatedDeviceConfig) int {
	n := 0
	for _, dev := range devs {
		if dev != nil && IsMthreadsHami(dev.Vendor, dev.SharingMode) {
			n++
		}
	}
	return n
}

// MthreadsSgpuComputingPowerWeight maps an SM percent to the sGPU time-slice weight.
// Zero uses the driver default of 1.
func MthreadsSgpuComputingPowerWeight(smUtilLimit int) int {
	if smUtilLimit < MthreadsSgpuWeightMin {
		return MthreadsSgpuWeightMin
	}
	if smUtilLimit > MthreadsSgpuWeightMax {
		return MthreadsSgpuWeightMax
	}
	return smUtilLimit
}
