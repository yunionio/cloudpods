package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CloudregionListOptions struct {
		options.BaseListOptions
		Private   *bool `help:"show private cloud regions only" json:"is_private"`
		Public    *bool `help:"show public cloud regions only" json:"is_public"`
		Usable    *bool `help:"List regions where networks are usable"`
		UsableVpc *bool `help:"List regions where VPC are usable"`
	}
	R(&CloudregionListOptions{}, "cloud-region-list", "List cloud regions", func(s *mcclient.ClientSession, opts *CloudregionListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Cloudregions.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cloudregions.GetColumns(s))
		return nil
	})

	type CloudregionCreateOptions struct {
		Id       string `help:"ID of the region"`
		NAME     string `help:"Name of the region"`
		Provider string `help:"Cloud provider"`
		Desc     string `help:"Description"`
	}
	R(&CloudregionCreateOptions{}, "cloud-region-create", "Create a cloud region", func(s *mcclient.ClientSession, args *CloudregionCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Id) > 0 {
			params.Add(jsonutils.NewString(args.Id), "id")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		results, err := modules.Cloudregions.Create(s, params)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	type CloudregionShowOptions struct {
		ID string `help:"ID or name of the region"`
	}
	R(&CloudregionShowOptions{}, "cloud-region-show", "Show a cloud region", func(s *mcclient.ClientSession, args *CloudregionShowOptions) error {
		results, err := modules.Cloudregions.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	R(&CloudregionShowOptions{}, "cloud-region-delete", "Delete a cloud region", func(s *mcclient.ClientSession, args *CloudregionShowOptions) error {
		results, err := modules.Cloudregions.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	type CloudregionUpdateOptions struct {
		ID   string `help:"ID or name of the region"`
		Name string `help:"New name of the region"`
		Desc string `help:"Description of the region"`
	}
	R(&CloudregionUpdateOptions{}, "cloud-region-update", "Update a cloud region", func(s *mcclient.ClientSession, args *CloudregionUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		results, err := modules.Cloudregions.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	type CloudregionSetDefaultVpcOptions struct {
		ID  string `help:"ID or name of the region"`
		VPC string `help:"ID or name of VPC to make default"`
	}
	R(&CloudregionSetDefaultVpcOptions{}, "cloud-region-set-default-vpc", "Set default vpc for a region", func(s *mcclient.ClientSession, args *CloudregionSetDefaultVpcOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.VPC), "vpc")
		result, err := modules.Cloudregions.PerformAction(s, args.ID, "default-vpc", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
