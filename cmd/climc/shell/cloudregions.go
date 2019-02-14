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

		City string `help:"List regions in the specified city"`
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

	type CloudregionCityListOptions struct {
		Manager  string `help:"List objects belonging to the cloud provider"`
		Account  string `help:"List objects belonging to the cloud account"`
		Provider string `help:"List objects from the provider" choices:"VMware|Aliyun|Qcloud|Azure|Aws|Huawei|Openstack"`

		Private   *bool `help:"show private cloud regions only" json:"is_private"`
		Public    *bool `help:"show public cloud regions only" json:"is_public"`
		Usable    *bool `help:"List regions where networks are usable"`
		UsableVpc *bool `help:"List regions where VPC are usable"`
	}
	R(&CloudregionCityListOptions{}, "cloud-region-cities", "List cities where cloud region resides", func(s *mcclient.ClientSession, args *CloudregionCityListOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		results, err := modules.Cloudregions.GetRegionCities(s, params)
		if err != nil {
			return err
		}
		listResult := modules.ListResult{}
		listResult.Data, err = results.GetArray()
		if err != nil {
			return err
		}
		printList(&listResult, nil)
		return nil
	})

	R(&CloudregionCityListOptions{}, "cloud-region-providers", "List cities where cloud region resides", func(s *mcclient.ClientSession, args *CloudregionCityListOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		results, err := modules.Cloudregions.GetRegionProviders(s, params)
		if err != nil {
			return err
		}
		listResult := modules.ListResult{}
		listResult.Data, err = results.GetArray()
		if err != nil {
			return err
		}
		printList(&listResult, nil)
		return nil
	})

	type CloudregionCreateOptions struct {
		Id          string  `help:"ID of the region"`
		NAME        string  `help:"Name of the region"`
		Provider    string  `help:"Cloud provider"`
		Desc        string  `help:"Description" json:"description" token:"desc"`
		Latitude    float32 `help:"region geographical location - latitude"`
		Longitude   float32 `help:"region geographical location - longitude"`
		City        string  `help:"region geograpical location - city, e.g. Beijing, Frankfurt"`
		CountryCode string  `help:"region geographical location - ISO country code, e.g. CN"`
	}
	R(&CloudregionCreateOptions{}, "cloud-region-create", "Create a cloud region", func(s *mcclient.ClientSession, args *CloudregionCreateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
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
		ID          string  `help:"ID or name of the region to update" json:"-"`
		Name        string  `help:"New name of the region"`
		Desc        string  `help:"Description of the region" json:"description" token:"desc"`
		Latitude    float32 `help:"region geographical location - latitude"`
		Longitude   float32 `help:"region geographical location - longitude"`
		City        string  `help:"region geograpical location - city, e.g. Beijing, Frankfurt"`
		CountryCode string  `help:"region geographical location - ISO country code, e.g. CN"`
	}
	R(&CloudregionUpdateOptions{}, "cloud-region-update", "Update a cloud region", func(s *mcclient.ClientSession, args *CloudregionUpdateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
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
