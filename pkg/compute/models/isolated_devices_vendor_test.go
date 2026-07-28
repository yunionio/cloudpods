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
)

func TestGetVendorByVendorDeviceId(t *testing.T) {
	assert.Equal(t, "HYGON", GetVendorByVendorDeviceId("1d94:6320"))
	assert.Equal(t, "NVIDIA", GetVendorByVendorDeviceId("10de:1c82"))
	assert.Equal(t, "AMD", GetVendorByVendorDeviceId("1002:6611"))
	assert.Equal(t, "abcd", GetVendorByVendorDeviceId("abcd:0001"))
}

func TestSIsolatedDeviceGetVendor(t *testing.T) {
	dev := &SIsolatedDevice{VendorDeviceId: "1d94:6320"}
	assert.Equal(t, "HYGON", dev.getVendor())
}

func TestVendorDeviceIdPrefixForFilter(t *testing.T) {
	cases := map[string]string{
		"HYGON":  "1d94:",
		"NVIDIA": "10de:",
		"1d94":   "1d94:",
		"hygon":  "1d94:",
		"Hygon":  "1d94:",
		"nvidia": "10de:",
		"1D94":   "1d94:",
	}
	for vendor, want := range cases {
		assert.Equal(t, want, vendorDeviceIdPrefixForFilter(vendor), "vendor=%s", vendor)
	}
}
