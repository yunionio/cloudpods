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
	type ClouduserListOptions struct {
		options.BaseListOptions
		CloudaccountId  string `help:"Cloudaccount Id"`
		CloudproviderId string `help:"Cloudprovider Id"`
		CloudpolicyId   string `help:"filter cloudusers by cloudpolicy"`
		CloudgroupId    string `help:"filter cloudusers by cloudgroup"`
	}
	R(&ClouduserListOptions{}, "cloud-user-list", "List cloud users", func(s *mcclient.ClientSession, opts *ClouduserListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Cloudusers.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cloudusers.GetColumns(s))
		return nil
	})

	type ClouduserCreateOptions struct {
		Name            string
		CloudaccountId  string   `help:"Cloudaccount Id"`
		CloudproviderId string   `help:"Cloudprovider Id"`
		OwnerId         string   `help:"Owner Id"`
		CloudpolicyIds  []string `help:"cloudpolicy ids"`
		CloudgroupIds   []string `help:"cloudgroup ids"`
		Email           string   `help:"email address"`
		MobilePhone     string   `help:"phone number"`
		IsConsoleLogin  *bool    `help:"is console login"`
		Password        string   `help:"clouduser password"`
	}

	R(&ClouduserCreateOptions{}, "cloud-user-create", "Create cloud user", func(s *mcclient.ClientSession, opts *ClouduserCreateOptions) error {
		params := jsonutils.Marshal(opts)
		result, err := modules.Cloudusers.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ClouduserIdOptions struct {
		ID string `help:"Clouduser Id"`
	}

	R(&ClouduserIdOptions{}, "cloud-user-delete", "Delete cloud user", func(s *mcclient.ClientSession, opts *ClouduserIdOptions) error {
		result, err := modules.Cloudusers.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ClouduserIdOptions{}, "cloud-user-logininfo", "Show cloud user login info", func(s *mcclient.ClientSession, opts *ClouduserIdOptions) error {
		result, err := modules.Cloudusers.GetLoginInfo(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ClouduserIdOptions{}, "cloud-user-show", "Show cloud user", func(s *mcclient.ClientSession, opts *ClouduserIdOptions) error {
		result, err := modules.Cloudusers.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ClouduserSyncOptions struct {
		ID         string `help:"Clouduser Id"`
		PolicyOnly bool   `help:"Ony sync clouduser policies for cloud"`
	}

	R(&ClouduserSyncOptions{}, "cloud-user-sync", "Sync cloud user policies", func(s *mcclient.ClientSession, opts *ClouduserSyncOptions) error {
		result, err := modules.Cloudusers.PerformAction(s, opts.ID, "sync", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ClouduserIdOptions{}, "cloud-user-syncstatus", "Sync cloud user status", func(s *mcclient.ClientSession, opts *ClouduserIdOptions) error {
		result, err := modules.Cloudusers.PerformAction(s, opts.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ClouduserPolicyOptions struct {
		ID             string `help:"Clouduser Id"`
		CLOUDPOLICY_ID string `help:"cloudpolicy Id"`
	}

	R(&ClouduserPolicyOptions{}, "cloud-user-attach-policy", "Attach policy for cloud user", func(s *mcclient.ClientSession, opts *ClouduserPolicyOptions) error {
		result, err := modules.Cloudusers.PerformAction(s, opts.ID, "attach-policy", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ClouduserPolicyOptions{}, "cloud-user-detach-policy", "Detach policy from cloud user", func(s *mcclient.ClientSession, opts *ClouduserPolicyOptions) error {
		result, err := modules.Cloudusers.PerformAction(s, opts.ID, "detach-policy", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ClouduserPasswordOptions struct {
		ID       string `help:"Clouduser Id"`
		Password string `help:"clouduser password"`
	}

	R(&ClouduserPasswordOptions{}, "cloud-user-reset-password", "Reset clouduser password", func(s *mcclient.ClientSession, opts *ClouduserPasswordOptions) error {
		result, err := modules.Cloudusers.PerformAction(s, opts.ID, "reset-password", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ClouduserChangeOwnerOptions struct {
		ID      string `help:"clouduser id"`
		USER_ID string `help:"local user id"`
	}

	R(&ClouduserChangeOwnerOptions{}, "cloud-user-change-owner", "Change clouduser owner", func(s *mcclient.ClientSession, opts *ClouduserChangeOwnerOptions) error {
		result, err := modules.Cloudusers.PerformAction(s, opts.ID, "change-owner", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ClouduserGroupOptions struct {
		ID            string `help:"clouduser id"`
		CLOUDGROUP_ID string `help:"cloudgroup id" json:"cloudgroup_id"`
	}

	R(&ClouduserGroupOptions{}, "cloud-user-join-group", "Join user to cloudgroup", func(s *mcclient.ClientSession, opts *ClouduserGroupOptions) error {
		result, err := modules.Cloudusers.PerformAction(s, opts.ID, "join-group", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ClouduserGroupOptions{}, "cloud-user-leave-group", "Leave from cloudgroup", func(s *mcclient.ClientSession, opts *ClouduserGroupOptions) error {
		result, err := modules.Cloudusers.PerformAction(s, opts.ID, "leave-group", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
