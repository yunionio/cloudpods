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

import "yunion.io/x/onecloud/pkg/apis"

type SDBInstanceCreateInput struct {
	apis.Meta

	Name              string
	Description       string
	DisableDelete     *bool
	NetworkId         string
	Address           string
	MasterInstanceId  string
	SecgroupId        string
	Zone1             string
	Zone2             string
	Zone3             string
	ZoneId            string
	CloudregionId     string
	Cloudregion       string
	VpcId             string
	ManagerId         string
	NetworkExternalId string
	BillingType       string
	BillingCycle      string
	InstanceType      string
	Engine            string
	EngineVersion     string
	Category          string
	StorageType       string
	DiskSizeGB        int
	Password          string

	VcpuCount  int
	VmemSizeMb int
	Provider   string
}

type SDBInstanceChangeConfigInput struct {
	apis.Meta

	InstanceType string
	VCpuCount    int
	VmemSizeMb   int
	StorageType  string
	DiskSizeGB   int
	Category     string
}

type SDBInstanceRecoveryConfigInput struct {
	apis.Meta

	DBInstancebackup   string
	DBInstancebackupId string            `json:"dbinstancebackup_id"`
	Databases          map[string]string `json:"allowempty"`
}
