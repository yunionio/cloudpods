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

package candidate

import "fmt"

// GpuReservedResource describes resources reserved on a host for GPU workloads.
type GpuReservedResource struct {
	CPUCount      int64
	MemorySizeMB  int64
	StorageSizeMB int64
}

// IGpuReservedResourceGetter is implemented by host candidate getters.
type IGpuReservedResourceGetter interface {
	GetGpuReservedResource() GpuReservedResource
}

func (h *HostDesc) GetGpuReservedResource() GpuReservedResource {
	if h.GuestReservedResource == nil {
		return GpuReservedResource{}
	}
	return GpuReservedResource{
		CPUCount:      h.GuestReservedResource.CPUCount,
		MemorySizeMB:  h.GuestReservedResource.MemorySize,
		StorageSizeMB: h.GuestReservedResource.StorageSize,
	}
}

func (h *hostGetter) GetGpuReservedResource() GpuReservedResource {
	return h.h.GetGpuReservedResource()
}

func (r GpuReservedResource) storageHint(useRsvd bool) string {
	if useRsvd || r.StorageSizeMB <= 0 {
		return ""
	}
	return fmt.Sprintf("reserved for GPU: %dGB", r.StorageSizeMB/1024)
}

func (r GpuReservedResource) memoryHint(useRsvd bool) string {
	if useRsvd || r.MemorySizeMB <= 0 {
		return ""
	}
	return fmt.Sprintf("reserved for GPU: %dGB", r.MemorySizeMB/1024)
}

func (r GpuReservedResource) cpuHint(useRsvd bool) string {
	if useRsvd || r.CPUCount <= 0 {
		return ""
	}
	return fmt.Sprintf("reserved for GPU: %d vCPU", r.CPUCount)
}

// AppendStorageHint appends GPU reserved storage info to msg when applicable.
func (r GpuReservedResource) AppendStorageHint(msg string, useRsvd bool) string {
	return appendGpuHint(msg, r.storageHint(useRsvd))
}

// AppendMemoryHint appends GPU reserved memory info to msg when applicable.
func (r GpuReservedResource) AppendMemoryHint(msg string, useRsvd bool) string {
	return appendGpuHint(msg, r.memoryHint(useRsvd))
}

// AppendCpuHint appends GPU reserved CPU info to msg when applicable.
func (r GpuReservedResource) AppendCpuHint(msg string, useRsvd bool) string {
	return appendGpuHint(msg, r.cpuHint(useRsvd))
}

func appendGpuHint(msg, hint string) string {
	if hint == "" {
		return msg
	}
	if msg == "" {
		return hint
	}
	return msg + ", " + hint
}

// GetGpuReservedResourceFromGetter returns GPU reserved resources from a candidate getter.
func GetGpuReservedResourceFromGetter(getter interface{}) GpuReservedResource {
	if g, ok := getter.(IGpuReservedResourceGetter); ok {
		return g.GetGpuReservedResource()
	}
	return GpuReservedResource{}
}
