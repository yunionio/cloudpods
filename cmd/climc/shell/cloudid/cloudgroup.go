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

package cloudid

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CloudgroupListOptions struct {
		options.BaseListOptions

		ClouduserId   string `json:"clouduser_id"`
		CloudpolicyId string `json:"cloudpolicy_id"`
	}
	R(&CloudgroupListOptions{}, "cloud-group-list", "List cloud groups", func(s *mcclient.ClientSession, opts *CloudgroupListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Cloudgroups.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cloudgroups.GetColumns(s))
		return nil
	})

	type CloudgroupCreateOptions struct {
		NAME           string   `json:"name"`
		PROVIDER       string   `json:"provider" choices:"Google|Aliyun|Aws|Huawei|Qcloud"`
		CloudpolicyIds []string `json:"cloudpolicy_ids"`
		Desc           string   `json:"description"`
	}

	R(&CloudgroupCreateOptions{}, "cloud-group-create", "Create cloud group", func(s *mcclient.ClientSession, opts *CloudgroupCreateOptions) error {
		params := jsonutils.Marshal(opts)
		result, err := modules.Cloudgroups.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudgroupIdOptions struct {
		ID string `help:"Cloudgroup Id"`
	}

	R(&CloudgroupIdOptions{}, "cloud-group-delete", "Delete cloud group", func(s *mcclient.ClientSession, opts *CloudgroupIdOptions) error {
		result, err := modules.Cloudgroups.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudgroupIdOptions{}, "cloud-group-show", "Show cloud group", func(s *mcclient.ClientSession, opts *CloudgroupIdOptions) error {
		result, err := modules.Cloudgroups.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudgroupIdOptions{}, "cloud-group-syncstatus", "Sync cloud group status", func(s *mcclient.ClientSession, opts *CloudgroupIdOptions) error {
		result, err := modules.Cloudgroups.PerformAction(s, opts.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudgroupPolicyOptions struct {
		ID             string `help:"Cloudgroup Id"`
		CLOUDPOLICY_ID string `help:"Cloudpolicy Id"`
	}

	R(&CloudgroupPolicyOptions{}, "cloud-group-attach-policy", "Attach policy for cloud group", func(s *mcclient.ClientSession, opts *CloudgroupPolicyOptions) error {
		result, err := modules.Cloudgroups.PerformAction(s, opts.ID, "attach-policy", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudgroupPolicyOptions{}, "cloud-group-detach-policy", "Detach policy from cloud group", func(s *mcclient.ClientSession, opts *CloudgroupPolicyOptions) error {
		result, err := modules.Cloudgroups.PerformAction(s, opts.ID, "detach-policy", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudgroupUserOptions struct {
		ID           string `help:"Cloudgroup Id"`
		CLOUDUSER_ID string `help:"Clouduser Id"`
	}

	R(&CloudgroupUserOptions{}, "cloud-group-add-user", "Add user to cloud group", func(s *mcclient.ClientSession, opts *CloudgroupUserOptions) error {
		result, err := modules.Cloudgroups.PerformAction(s, opts.ID, "add-user", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudgroupUserOptions{}, "cloud-group-remove-user", "Remove user from cloud group", func(s *mcclient.ClientSession, opts *CloudgroupUserOptions) error {
		result, err := modules.Cloudgroups.PerformAction(s, opts.ID, "remove-user", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudgroupPoliciesOptions struct {
		ID             string   `help:"Cloudgroup Id"`
		CloudpolicyIds []string `json:"cloudpolicy_ids"`
	}

	R(&CloudgroupPoliciesOptions{}, "cloud-group-set-policies", "Set cloudpolicies for cloud group", func(s *mcclient.ClientSession, opts *CloudgroupPoliciesOptions) error {
		result, err := modules.Cloudgroups.PerformAction(s, opts.ID, "set-policies", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudgroupUsersOptions struct {
		ID           string   `help:"Cloudgroup Id"`
		ClouduserIds []string `json:"clouduser_ids"`
	}

	R(&CloudgroupUsersOptions{}, "cloud-group-set-users", "Set users for cloud group", func(s *mcclient.ClientSession, opts *CloudgroupUsersOptions) error {
		result, err := modules.Cloudgroups.PerformAction(s, opts.ID, "set-users", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
