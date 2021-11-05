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

package identity

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func init() {
	type RegionListOptions struct {
		Limit  int64  `help:"Limit, default 0, i.e. no limit" default:"20"`
		Offset int64  `help:"Offset, default 0, i.e. no offset"`
		Search string `help:"Search by name"`
	}
	R(&RegionListOptions{}, "region-list", "List regions", func(s *mcclient.ClientSession, args *RegionListOptions) error {
		query := jsonutils.NewDict()
		if args.Limit > 0 {
			query.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			query.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		if len(args.Search) > 0 {
			query.Add(jsonutils.NewString(args.Search), "id__icontains")
		}
		result, err := modules.Regions.List(s, query)
		if err != nil {
			return err
		}
		printList(result, modules.Regions.GetColumns(s))
		return nil
	})

	type RegionShowOptions struct {
		REGION string `help:"ID of region"`
		Zone   string `help:"ID of region"`
	}
	R(&RegionShowOptions{}, "region-show", "Show details of region", func(s *mcclient.ClientSession, args *RegionShowOptions) error {
		ID := mcclient.RegionID(args.REGION, args.Zone)
		result, err := modules.Regions.Get(s, ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&RegionShowOptions{}, "region-delete", "Delete region", func(s *mcclient.ClientSession, args *RegionShowOptions) error {
		ID := mcclient.RegionID(args.REGION, args.Zone)
		_, err := modules.Regions.Delete(s, ID, nil)
		if err != nil {
			return err
		}
		return nil
	})

	type RegionCreateOptions struct {
		REGION string `help:"ID of the region"`
		Zone   string `help:"ID of the zone"`
		Name   string `help:"Name of the region"`
		Desc   string `help:"Description"`
	}
	R(&RegionCreateOptions{}, "region-create", "Create a region", func(s *mcclient.ClientSession, args *RegionCreateOptions) error {
		params := jsonutils.NewDict()
		ID := mcclient.RegionID(args.REGION, args.Zone)
		params.Add(jsonutils.NewString(ID), "id")
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Zone) > 0 {
			params.Add(jsonutils.NewString(args.REGION), "parent_region_id")
		}
		region, err := modules.Regions.Create(s, params)
		if err != nil {
			return err
		}
		printObject(region)
		return nil
	})

	type RegionUpdateOptions struct {
		REGION string `help:"ID of region"`
		Zone   string `help:"Zone"`
		Name   string `help:"New name of the region"`
		Desc   string `help:"New description of the region"`
	}
	R(&RegionUpdateOptions{}, "region-update", "Update a region", func(s *mcclient.ClientSession, args *RegionUpdateOptions) error {
		ID := mcclient.RegionID(args.REGION, args.Zone)
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if params.Size() == 0 {
			return fmt.Errorf("No data to update")
		}
		region, err := modules.Regions.Patch(s, ID, params)
		if err != nil {
			return err
		}
		printObject(region)
		return nil
	})
}
