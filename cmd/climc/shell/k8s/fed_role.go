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

package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	options "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func init() {
	cmd := newFedResourceCmd(k8s.FederatedRoles)
	cmd.Create(new(options.FedRoleCreateOptions)).
		List(new(options.FedRoleListOptions)).
		Show(new(options.IdentOptions)).
		Delete(new(options.IdentOptions)).
		AttachCluster(new(options.FedResourceJointClusterAttachOptions)).
		DetachCluster(new(options.FedResourceJointClusterDetachOptions)).
		SyncCluster(new(options.FedResourceJointClusterDetachOptions)).
		Sync(new(options.IdentOptions)).
		Update(new(options.FedResourceUpdateOptions)).
		ShowEvent()

	cmd.ClassShow(new(options.FedApiResourecesOptions))
	cmd.ClassShow(new(options.FedClusterUsersOptions))
	cmd.ClassShow(new(options.FedClusterUserGroupsOptions))
}
