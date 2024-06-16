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
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Vpcs).WithContextManager(&modules.Cloudregions)
	cmd.List(&options.VpcListOptions{})
	cmd.Create(&options.VpcCreateOptions{})
	cmd.Show(&options.VpcIdOptions{})
	cmd.Delete(&options.VpcIdOptions{})
	cmd.Update(&options.VpcUpdateOptions{})
	cmd.Perform("status", &options.VpcStatusOptions{})
	cmd.Perform("purge", &options.VpcIdOptions{})
	cmd.Perform("sync", &options.VpcIdOptions{})
	cmd.Perform("syncstatus", &options.VpcIdOptions{})
	cmd.Perform("private", &options.VpcIdOptions{})
	cmd.Perform("public", &baseoptions.BasePublicOptions{})
	cmd.Perform("change-owner", &options.VpcChangeOwnerOptions{})
	cmd.Get("vpc-change-owner-candidate-domains", &options.VpcIdOptions{})
	cmd.Get("topology", &options.VpcIdOptions{})

	R(&baseoptions.ResourceMetadataOptions{}, "vpc-set-user-metadata", "Set metadata of a vpc", func(s *mcclient.ClientSession, opts *baseoptions.ResourceMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.Vpcs.PerformAction(s, opts.ID, "user-metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
