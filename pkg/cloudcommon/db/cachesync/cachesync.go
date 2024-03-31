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

package cachesync

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

var tenantCacheSyncWorkerMan *appsrv.SWorkerManager

func init() {
	tenantCacheSyncWorkerMan = appsrv.NewWorkerManagerIgnoreOverflow("tenant_cache_sync_worker", 1, 1, true, true)
	// tenantCacheSyncWorkerMan.EnableCancelPreviousIdenticalTask()
}

func StartTenantCacheSync(intvalSeconds int) {
	newResourceChangeManager(identity_modules.Projects, intvalSeconds)
	newResourceChangeManager(identity_modules.Domains, intvalSeconds)
	newResourceChangeManager(identity_modules.UsersV3, intvalSeconds)
}
