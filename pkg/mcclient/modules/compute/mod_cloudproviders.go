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
	Cloudproviders modulebase.ResourceManager
)

func init() {
	Cloudproviders = modules.NewComputeManager("cloudprovider", "cloudproviders",
		[]string{"ID", "Name", "Enabled", "Status", "Access_url", "Account",
			"Sync_Status", "Last_sync", "Last_sync_end_at",
			"health_status",
			"Provider", "guest_count", "host_count", "vpc_count",
			"storage_count", "storage_cache_count", "eip_count",
			"tenant_id", "tenant"},
		[]string{})

	modules.RegisterCompute(&Cloudproviders)
}
