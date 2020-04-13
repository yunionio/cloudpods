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
	type CloudpolicyListOptions struct {
		options.BaseListOptions

		ClouduserId  string `json:"clouduser_id"`
		CloudgroupId string `json:"cloudgroup_id"`
	}
	R(&CloudpolicyListOptions{}, "cloud-policy-list", "List cloud policies", func(s *mcclient.ClientSession, opts *CloudpolicyListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Cloudpolicies.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cloudpolicies.GetColumns(s))
		return nil
	})

	type CloudpolicyIdOptions struct {
		ID string `help:"Cloudpolicy Id"`
	}

	R(&CloudpolicyIdOptions{}, "cloud-policy-show", "Sow cloud policiy details", func(s *mcclient.ClientSession, opts *CloudpolicyIdOptions) error {
		result, err := modules.Cloudpolicies.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudpolicyGroupOptions struct {
		ID            string `help:"Cloudpolicy Id"`
		CLOUDGROUP_ID string `help:"Cloudgroup Id" json:"cloudgroup_id"`
	}

	R(&CloudpolicyGroupOptions{}, "cloud-policy-assign-group", "Assign cloud policiy to group", func(s *mcclient.ClientSession, opts *CloudpolicyGroupOptions) error {
		result, err := modules.Cloudpolicies.PerformAction(s, opts.ID, "assign-group", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudpolicyGroupOptions{}, "cloud-policy-revoke-group", "Revoke cloud policiy from group", func(s *mcclient.ClientSession, opts *CloudpolicyGroupOptions) error {
		result, err := modules.Cloudpolicies.PerformAction(s, opts.ID, "revoke-group", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
