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

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Cloudaccounts modulebase.ResourceManager
)

func init() {
	Cloudaccounts = modules.NewComputeManager("cloudaccount", "cloudaccounts",
		[]string{"ID", "Name", "Enabled", "Status", "Access_url",
			"balance", "error_count", "health_status",
			"Sync_Status", "Last_sync",
			"guest_count", "project_domain", "domain_id",
			"Provider", "Brand",
			"Enable_Auto_Sync", "Sync_Interval_Seconds",
			"Share_Mode", "is_public", "public_scope",
			"auto_create_project",
		},
		[]string{})

	modules.RegisterCompute(&Cloudaccounts)
}
