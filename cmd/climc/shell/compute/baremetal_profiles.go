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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	baremetalmodules "yunion.io/x/onecloud/pkg/mcclient/modules/compute/baremetal"
	baremetaloptions "yunion.io/x/onecloud/pkg/mcclient/options/compute/baremetal"
)

func init() {
	cmd := shell.NewResourceCmd(&baremetalmodules.BaremetalProfiles)
	cmd.List(new(baremetaloptions.BaremetalProfileListOptions))
	cmd.Show(new(baremetaloptions.BaremetalProfileIdOptions))
	cmd.Delete(new(baremetaloptions.BaremetalProfileIdOptions))

	type BaremetalProfileMatchOpts struct {
		OEMNAME string
		MODEL   string
	}
	R(&BaremetalProfileMatchOpts{}, "baremetal-profile-match", "find match baremetal profile by oemname and model", func(s *mcclient.ClientSession, args *BaremetalProfileMatchOpts) error {
		specs, err := baremetalmodules.BaremetalProfiles.GetMatchProfiles(s, args.OEMNAME, args.MODEL)
		if err != nil {
			return errors.Wrap(err, "GetMatchProfiles")
		}
		listResults := printutils.ListResult{}
		for _, spec := range specs {
			listResults.Data = append(listResults.Data, jsonutils.Marshal(spec))
		}
		printList(&listResults, nil)
		return nil
	})
}
