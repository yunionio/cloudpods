package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

func init() {
	type VpcListOptions struct {
		BaseListOptions
		Region string `help:"ID or Name of region"`
	}
	R(&VpcListOptions{}, "vpc-list", "List VPCs", func(s *mcclient.ClientSession, args *VpcListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		var result *modules.ListResult
		var err error
		if len(args.Region) > 0 {
			result, err = modules.Vpcs.ListInContext(s, params, &modules.Cloudregions, args.Region)
		} else {
			result, err = modules.Vpcs.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Vpcs.GetColumns(s))
		return nil
	})

	type VpcCreateOptions struct {
		REGION string `help:"ID or name of the region where the VPC is created"`
		Id     string `help:"ID of the new VPC"`
		NAME   string `help:"Name of the VPC"`
		Desc   string `help:"Description of the VPC"`
	}
	R(&VpcCreateOptions{}, "vpc-create", "Create a VPC", func(s *mcclient.ClientSession, args *VpcCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Id) > 0 {
			params.Add(jsonutils.NewString(args.Id), "id")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		results, err := modules.Vpcs.CreateInContext(s, params, &modules.Cloudregions, args.REGION)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	type VpcShowOptions struct {
		ID string `help:"ID or name of the region"`
	}
	R(&VpcShowOptions{}, "vpc-show", "Show a VPC", func(s *mcclient.ClientSession, args *VpcShowOptions) error {
		results, err := modules.Vpcs.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	R(&VpcShowOptions{}, "vpc-delete", "Delete a VPC", func(s *mcclient.ClientSession, args *VpcShowOptions) error {
		results, err := modules.Vpcs.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	type VpcUpdateOptions struct {
		ID   string `help:"ID or name of the VPC"`
		Name string `help:"New name of the VPC"`
		Desc string `help:"Description of the VPC"`
	}
	R(&VpcUpdateOptions{}, "vpc-update", "Update a VPC", func(s *mcclient.ClientSession, args *VpcUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		results, err := modules.Vpcs.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	type VpcUpdateStatusOptions struct {
		ID string `help:"ID or name of the VPC"`
	}
	R(&VpcUpdateStatusOptions{}, "vpc-available", "Make vpc status available", func(s *mcclient.ClientSession, args *VpcUpdateStatusOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString("available"), "status")
		result, err := modules.Vpcs.PerformAction(s, args.ID, "status", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&VpcUpdateStatusOptions{}, "vpc-pending", "Make vpc status pending", func(s *mcclient.ClientSession, args *VpcUpdateStatusOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString("pending"), "status")
		result, err := modules.Vpcs.PerformAction(s, args.ID, "status", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
