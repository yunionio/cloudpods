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

package compute

type SGMapItem struct {
	SGDeviceName    string `json:"sg_device_name"`
	HostNumber      int    `json:"host_number"`
	Bus             int    `json:"bus"`
	SCSIId          int    `json:"scsi_id"`
	Lun             int    `json:"lun"`
	Type            int    `json:"type"`
	LinuxDeviceName string `json:"linux_device_name"`
}

const (
	IPMIUserPrivUser  = "USER"
	IPMIUSERPrivAdmin = "ADMINISTRATOR"
)

type IPMIUser struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Callin   bool   `json:"callin"`
	LinkAuth bool   `json:"link_auth"`
	IPMIMsg  bool   `json:"ipmi_msg"`
	Priv     string `json:"priv"`
}
