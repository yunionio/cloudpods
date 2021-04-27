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
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Storages).WithContextManager(&modules.Zones)
	cmd.List(&compute.StorageListOptions{})
	cmd.Update(&compute.StorageUpdateOptions{})
	cmd.Create(&compute.StorageCreateOptions{})
	cmd.Show(&options.BaseShowOptions{})
	cmd.Delete(&options.BaseIdOptions{})
	cmd.Perform("enable", &options.BaseIdOptions{})
	cmd.Perform("disable", &options.BaseIdOptions{})
	cmd.Perform("online", &options.BaseIdOptions{})
	cmd.Perform("offline", &options.BaseIdOptions{})
	cmd.Perform("cache-image", &compute.StorageCacheImageActionOptions{})
	cmd.Perform("uncache-image", &compute.StorageUncacheImageActionOptions{})
	cmd.Perform("change-owner", &options.ChangeOwnerOptions{})
	cmd.Perform("force-detach-host", &compute.StorageForceDetachHost{})
	cmd.Perform("public", &options.BasePublicOptions{})
	cmd.Perform("private", &options.BaseIdOptions{})
}
