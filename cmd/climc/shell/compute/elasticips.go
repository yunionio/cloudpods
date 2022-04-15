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
	cmd := shell.NewResourceCmd(&modules.Elasticips).WithKeyword("eip")
	cmd.List(&compute.ElasticipListOptions{})
	cmd.Create(&compute.EipCreateOptions{})
	cmd.Delete(&options.BaseIdOptions{})
	cmd.Update(&compute.EipUpdateOptions{})
	cmd.Show(&options.BaseShowOptions{})
	cmd.Perform("purge", &options.BaseIdOptions{})
	cmd.Perform("associate", &compute.EipAssociateOptions{})
	cmd.Perform("dissociate", &compute.EipDissociateOptions{})
	cmd.Perform("sync", &options.BaseIdOptions{})
	cmd.Perform("syncstatus", &options.BaseIdOptions{})
	cmd.Perform("change-bandwidth", &compute.EipChangeBandwidthOptions{})
	cmd.Perform("change-owner", &compute.EipChangeOwnerOptions{})
}
