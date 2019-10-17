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
	//"fmt"
	//"strings"

	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	printAnsiblePlaybookObject := func(obj jsonutils.JSONObject) {
		dict := obj.(*jsonutils.JSONDict)
		pbJson, err := dict.Get("playbook")
		if err != nil {
			printObject(obj)
			return
		}
		pbStr := pbJson.YAMLString()
		dict.Set("playbook", jsonutils.NewString(pbStr))
		printObject(obj)
	}
	R(&options.AnsiblePlaybookCreateOptions{}, "ansibleplaybook-create", "Create ansible playbook", func(s *mcclient.ClientSession, opts *options.AnsiblePlaybookCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		log.Errorf("create playbook params: %+v", params)
		os.Exit(1)
		apb, err := modules.AnsiblePlaybooks.Create(s, params)
		if err != nil {
			return err
		}
		printAnsiblePlaybookObject(apb)
		return nil
	})
	R(&options.AnsiblePlaybookIdOptions{}, "ansibleplaybook-show", "Show ansible playbook", func(s *mcclient.ClientSession, opts *options.AnsiblePlaybookIdOptions) error {
		apb, err := modules.AnsiblePlaybooks.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printAnsiblePlaybookObject(apb)
		return nil
	})
	R(&options.AnsiblePlaybookListOptions{}, "ansibleplaybook-list", "List ansible playbooks", func(s *mcclient.ClientSession, opts *options.AnsiblePlaybookListOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		apbs, err := modules.AnsiblePlaybooks.List(s, params)
		if err != nil {
			return err
		}
		printList(apbs, modules.AnsiblePlaybooks.GetColumns(s))
		return nil
	})
	R(&options.AnsiblePlaybookIdOptions{}, "ansibleplaybook-delete", "Delete ansible playbook", func(s *mcclient.ClientSession, opts *options.AnsiblePlaybookIdOptions) error {
		apb, err := modules.AnsiblePlaybooks.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printAnsiblePlaybookObject(apb)
		return nil
	})
	R(&options.AnsiblePlaybookUpdateOptions{}, "ansibleplaybook-update", "Update ansible playbook", func(s *mcclient.ClientSession, opts *options.AnsiblePlaybookUpdateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		apb, err := modules.AnsiblePlaybooks.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printAnsiblePlaybookObject(apb)
		return nil
	})
	R(&options.AnsiblePlaybookIdOptions{}, "ansibleplaybook-run", "Run ansible playbook", func(s *mcclient.ClientSession, opts *options.AnsiblePlaybookIdOptions) error {
		apb, err := modules.AnsiblePlaybooks.PerformAction(s, opts.ID, "run", nil)
		if err != nil {
			return err
		}
		printAnsiblePlaybookObject(apb)
		return nil
	})
	R(&options.AnsiblePlaybookIdOptions{}, "ansibleplaybook-stop", "Stop ansible playbook", func(s *mcclient.ClientSession, opts *options.AnsiblePlaybookIdOptions) error {
		apb, err := modules.AnsiblePlaybooks.PerformAction(s, opts.ID, "stop", nil)
		if err != nil {
			return err
		}
		printAnsiblePlaybookObject(apb)
		return nil
	})
}
