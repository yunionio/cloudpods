package shell

import (
	"fmt"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
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
