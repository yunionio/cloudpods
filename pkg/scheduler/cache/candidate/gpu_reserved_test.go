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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGpuReservedResourceHints(t *testing.T) {
	r := GpuReservedResource{
		CPUCount:      64,
		MemorySizeMB:  65536,
		StorageSizeMB: 819200,
	}

	assert.Equal(t, "msg", r.AppendStorageHint("msg", true))
	assert.Equal(t, "msg, reserved for GPU: 800GB", r.AppendStorageHint("msg", false))
	rZero := GpuReservedResource{}
	assert.Equal(t, "msg", rZero.AppendStorageHint("msg", false))

	assert.Equal(t,
		"no enough storage, free=1024, reserved for GPU: 800GB",
		r.AppendStorageHint("no enough storage, free=1024", false),
	)
	assert.Equal(t, "reserved for GPU: 64GB", r.AppendMemoryHint("", false))
	assert.Equal(t, "reserved for GPU: 64 vCPU", r.AppendCpuHint("", false))
}

func TestHostDescGetGpuReservedResource(t *testing.T) {
	desc := &HostDesc{
		GuestReservedResource: NewReservedResource(8, 8192, 102400),
	}
	r := desc.GetGpuReservedResource()
	assert.Equal(t, int64(8), r.CPUCount)
	assert.Equal(t, int64(8192), r.MemorySizeMB)
	assert.Equal(t, int64(102400), r.StorageSizeMB)

	nilDesc := &HostDesc{}
	assert.Equal(t, GpuReservedResource{}, nilDesc.GetGpuReservedResource())
}

func TestGetGpuReservedResourceFromGetter(t *testing.T) {
	desc := &HostDesc{
		GuestReservedResource: NewReservedResource(16, 16384, 204800),
	}
	getter := newHostGetter(desc)
	r := GetGpuReservedResourceFromGetter(getter)
	assert.Equal(t, int64(16), r.CPUCount)
	assert.Equal(t, GpuReservedResource{}, GetGpuReservedResourceFromGetter(nil))
}
