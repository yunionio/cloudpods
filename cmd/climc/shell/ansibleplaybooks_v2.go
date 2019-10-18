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

package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type AnsiblePlaybookV2IdOptions struct {
		ID string `help:"name/id of the playbook"`
	}

	type AnsiblePlaybookV2ListOptions struct {
		options.BaseListOptions
	}

	R(&AnsiblePlaybookV2IdOptions{}, "ansibleplaybookv2-show", "Show ansible playbook", func(s *mcclient.ClientSession, opts *AnsiblePlaybookV2IdOptions) error {
		apb, err := modules.AnsiblePlaybooksV2.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(apb)
		return nil
	})
	R(&AnsiblePlaybookV2ListOptions{}, "ansibleplaybookv2-list", "List ansible playbooks", func(s *mcclient.ClientSession, opts *AnsiblePlaybookV2ListOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		apbs, err := modules.AnsiblePlaybooksV2.List(s, params)
		if err != nil {
			return err
		}
		printList(apbs, modules.AnsiblePlaybooksV2.GetColumns(s))
		return nil
	})
}
