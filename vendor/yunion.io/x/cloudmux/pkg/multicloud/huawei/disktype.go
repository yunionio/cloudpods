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

package huawei

type SDiskType struct {
	ExtraSpecs ExtraSpecs `json:"extra_specs"`
	Name       string     `json:"name"`
	QosSpecsID string     `json:"qos_specs_id"`
	Id         string     `json:"id"`
	IsPublic   bool       `json:"is_public"`
}

type ExtraSpecs struct {
	VolumeBackendName                        string `json:"volume_backend_name"`
	AvailabilityZone                         string `json:"availability-zone"`
	RESKEYAvailabilityZones                  string `json:"RESKEY:availability_zones"`
	OSVendorExtendedSoldOutAvailabilityZones string `json:"os-vendor-extended:sold_out_availability_zones"`
}
