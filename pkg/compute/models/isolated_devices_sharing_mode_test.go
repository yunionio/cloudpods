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

	"github.com/stretchr/testify/assert"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestFilterDevicesBySharingMode(t *testing.T) {
	devs := []SIsolatedDevice{
		{Model: "GeForce RTX 4090", DevType: api.GPU_TYPE, SharingMode: api.DEVICE_SHARING_MODE_EXCLUSIVE},
		{Model: "GeForce RTX 4090", DevType: api.GPU_TYPE, SharingMode: api.DEVICE_SHARING_MODE_HAMI},
		{Model: "GeForce RTX 4090", DevType: api.GPU_TYPE, SharingMode: api.DEVICE_SHARING_MODE_MPS},
	}

	t.Run("empty sharing mode keeps all", func(t *testing.T) {
		got := filterDevicesBySharingMode(devs, "")
		assert.Equal(t, 3, len(got))
	})

	t.Run("HAMI selects only HAMI", func(t *testing.T) {
		got := filterDevicesBySharingMode(devs, api.DEVICE_SHARING_MODE_HAMI)
		assert.Equal(t, 1, len(got))
		assert.Equal(t, api.DEVICE_SHARING_MODE_HAMI, got[0].SharingMode)
	})

	t.Run("EXCLUSIVE selects only EXCLUSIVE", func(t *testing.T) {
		got := filterDevicesBySharingMode(devs, api.DEVICE_SHARING_MODE_EXCLUSIVE)
		assert.Equal(t, 1, len(got))
		assert.Equal(t, api.DEVICE_SHARING_MODE_EXCLUSIVE, got[0].SharingMode)
	})

	t.Run("unmatched mode returns empty", func(t *testing.T) {
		got := filterDevicesBySharingMode(devs, api.DEVICE_SHARING_MODE_UNLIMITED)
		assert.Equal(t, 0, len(got))
	})
}

func TestIsolatedDeviceRestoreKey(t *testing.T) {
	key := isolatedDeviceRestoreKey(api.GPU_TYPE, api.DEVICE_SHARING_MODE_HAMI, "GeForce RTX 4090")
	devType, sharingMode, model, ok := parseIsolatedDeviceRestoreKey(key)
	assert.True(t, ok)
	assert.Equal(t, api.GPU_TYPE, devType)
	assert.Equal(t, api.DEVICE_SHARING_MODE_HAMI, sharingMode)
	assert.Equal(t, "GeForce RTX 4090", model)

	_, _, _, ok = parseIsolatedDeviceRestoreKey("bad-key")
	assert.False(t, ok)
}
