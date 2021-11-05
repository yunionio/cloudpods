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

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type GlobalVpcListOptions struct {
		options.BaseListOptions
	}
	R(&GlobalVpcListOptions{}, "global-vpc-list", "List global vpcs", func(s *mcclient.ClientSession, args *GlobalVpcListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.GlobalVpcs.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.GlobalVpcs.GetColumns(s))
		return nil
	})

	type GlobalVpcShowOptions struct {
		ID string `help:"ID or Name of globalvpc"`
	}
	R(&GlobalVpcShowOptions{}, "global-vpc-show", "Show details of a global vpc", func(s *mcclient.ClientSession, args *GlobalVpcShowOptions) error {
		result, err := modules.GlobalVpcs.GetById(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type GlobalVpcPublicOptions struct {
		ID            string   `help:"ID or name of global vpc" json:"-"`
		Scope         string   `help:"sharing scope" choices:"system|domain"`
		SharedDomains []string `help:"share to domains"`
	}
	R(&GlobalVpcPublicOptions{}, "global-vpc-public", "Make global vpc public", func(s *mcclient.ClientSession, args *GlobalVpcPublicOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.GlobalVpcs.PerformAction(s, args.ID, "public", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type GlobalVpcPrivateOptions struct {
		ID string `help:"ID or name of global vpc" json:"-"`
	}
	R(&GlobalVpcPrivateOptions{}, "global-vpc-private", "Make global vpc private", func(s *mcclient.ClientSession, args *GlobalVpcPrivateOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.GlobalVpcs.PerformAction(s, args.ID, "private", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&GlobalVpcShowOptions{}, "global-vpc-change-owner-candidate-domains", "Show candiate domains of a global vpc for changing owner", func(s *mcclient.ClientSession, args *GlobalVpcShowOptions) error {
		result, err := modules.GlobalVpcs.GetSpecific(s, args.ID, "change-owner-candidate-domains", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type GlobalVpcChangeOwnerOptions struct {
		ID            string `help:"ID or name of vpc" json:"-"`
		ProjectDomain string `json:"project_domain" help:"target domain"`
	}
	R(&GlobalVpcChangeOwnerOptions{}, "global-vpc-change-owner", "Change owner domain of a global vpc", func(s *mcclient.ClientSession, args *GlobalVpcChangeOwnerOptions) error {
		if len(args.ProjectDomain) == 0 {
			return fmt.Errorf("empty project_domain")
		}
		params := jsonutils.Marshal(args)
		ret, err := modules.GlobalVpcs.PerformAction(s, args.ID, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
