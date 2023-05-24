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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

type ClircShowOpt struct {
	options.BaseIdOptions
	FollowOutputFormat bool `help:"follow output format"`
}

func init() {
	cmd := shell.NewResourceCmd(&modules.Cloudproviders).WithKeyword("cloud-provider")
	cmd.List(&compute.CloudproviderListOptions{})
	cmd.Update(&compute.CloudproviderUpdateOptions{})
	cmd.Show(&options.BaseIdOptions{})
	cmd.Delete(&options.BaseIdOptions{})
	cmd.Perform("change-project", &compute.CloudproviderChangeProjectOptions{})
	cmd.Perform("enable", &options.BaseIdOptions{})
	cmd.Perform("disable", &options.BaseIdOptions{})
	cmd.Perform("sync", &compute.CloudproviderSyncOptions{})
	cmd.Perform("project-mapping", &compute.ClouproviderProjectMappingOptions{})
	cmd.Perform("set-syncing", &compute.ClouproviderSetSyncingOptions{})

	cmd.GetWithCustomOptionShow("clirc", func(result jsonutils.JSONObject, opt shell.IGetOpt) {
		rc := make(map[string]string)
		err := result.Unmarshal(&rc)
		if err != nil {
			log.Errorf("Unmarshal error: %v", err)
			return
		}
		if opt.(*ClircShowOpt).FollowOutputFormat {
			shell.PrintObject(result)
		} else {
			for k, v := range rc {
				fmt.Printf(`export %s='%s'`+"\n", k, v)
			}
		}
	}, &ClircShowOpt{})
	cmd.Get("storage-classes", &compute.CloudproviderStorageClassesOptions{})
	cmd.Get("change-owner-candidate-domains", &options.BaseIdOptions{})
	cmd.Get("balance", &options.BaseIdOptions{})
}
