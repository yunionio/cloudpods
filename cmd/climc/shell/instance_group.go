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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type InstanceGroupListOptions struct {
		options.BaseListOptions

		ServiceType string `help:"Service Type"`
		ParentId    string `help:"Parent ID"`
		ZoneId      string `help:"Zone ID"`
		Server      string `help:"Guest ID or Name"`
	}

	R(&InstanceGroupListOptions{}, "instance-group-list", "List instance group", func(s *mcclient.ClientSession,
		args *InstanceGroupListOptions) error {

		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.InstanceGroup.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.InstanceGroup.GetColumns(s))
		return nil
	})

	type InstanceGroupShowOptions struct {
		ID string `help:"ID or Name of instance group"`
	}
	R(&InstanceGroupShowOptions{}, "instance-group-show", "Show details of a instance group",
		func(s *mcclient.ClientSession, args *InstanceGroupShowOptions) error {

			result, err := modules.InstanceGroup.GetById(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})

	type InstanceGroupCreateOptions struct {
		NAME string `help:"name of instance group"`

		ZoneId          string `help:"zone id" json:"zone_id"`
		ServiceType     string `help:"service type"`
		ParentId        string `help:"parent id"`
		SchedStrategy   string `help:"scheduler strategy"`
		Granularity     string `help:"the upper limit number of guests with this group in a host"`
		ForceDispersion bool   `help:"force to make guest dispersion"`
	}

	R(&InstanceGroupCreateOptions{}, "instance-group-create", "Create a instance group",
		func(s *mcclient.ClientSession, args *InstanceGroupCreateOptions) error {
			params, err := options.StructToParams(args)
			if err != nil {
				return err
			}
			result, err := modules.InstanceGroup.Create(s, params)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		},
	)

	R(&InstanceGroupShowOptions{}, "instance-group-delete", "delete a instance group",
		func(s *mcclient.ClientSession, args *InstanceGroupShowOptions) error {
			result, err := modules.InstanceGroup.Delete(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		},
	)

	type InstanceGroupUpdateOptions struct {
		ID              string `help:"ID or Name of servers to update" json:"-"`
		Name            string `help:"New name to change"`
		Granularity     string `help:"the upper limit number of guests with this group in a host"`
		ForceDispersion string `help:"force to make guest dispersion" choices:"yes|no" json:"-"`
	}

	R(&InstanceGroupUpdateOptions{}, "instance-group-update", "update a instance group",
		func(s *mcclient.ClientSession, args *InstanceGroupUpdateOptions) error {
			params, err := options.StructToParams(args)
			if err != nil {
				return err
			}
			if args.ForceDispersion == "yes" {
				params.Set("force_dispersion", jsonutils.JSONTrue)
			} else {
				params.Set("force_dispersion", jsonutils.JSONFalse)
			}
			ret, err := modules.InstanceGroup.Update(s, args.ID, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

}
