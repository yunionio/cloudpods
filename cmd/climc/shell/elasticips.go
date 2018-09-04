package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ElasticipListOptions struct {
		Manager string `help:"Show servers imported from manager"`
		Region  string `help:"Show servers in cloudregion"`

		options.BaseListOptions
	}
	R(&ElasticipListOptions{}, "eip-list", "List elastic IPs", func(s *mcclient.ClientSession, args *ElasticipListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
		}
		if len(args.Manager) > 0 {
			params.Add(jsonutils.NewString(args.Manager), "manager")
		}
		if len(args.Region) > 0 {
			params.Add(jsonutils.NewString(args.Region), "region")
		}
		results, err := modules.Elasticips.List(s, params)
		if err != nil {
			return err
		}
		printList(results, modules.Elasticips.GetColumns(s))
		return nil
	})

	type EipCreateOptions struct {
		MANAGER    string `help:"cloud provider"`
		REGION     string `help:"cloud region in which EIP is allocated"`
		NAME       string `help:"name of the EIP"`
		BW         int    `help:"Bandwidth in Mbps"`
		ChargeType string `help:"bandwidth charge type, either traffic or bandwidth" choices:"traffic|bandwidth"`
	}
	R(&EipCreateOptions{}, "eip-create", "Create an EIP", func(s *mcclient.ClientSession, args *EipCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.MANAGER), "manager")
		params.Add(jsonutils.NewString(args.REGION), "region")
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewInt(int64(args.BW)), "bandwidth")

		if len(args.ChargeType) > 0 {
			params.Add(jsonutils.NewString(args.ChargeType), "charge_type")
		}

		result, err := modules.Elasticips.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type EipDeleteOptions struct {
		ID string `help:"ID or name of EIP"`
	}
	R(&EipDeleteOptions{}, "eip-delete", "Delete an EIP", func(s *mcclient.ClientSession, args *EipDeleteOptions) error {
		result, err := modules.Elasticips.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type EipUpdateOptions struct {
		ID   string `help:"ID or name of EIP"`
		Name string `help:"New name of EIP"`
		Desc string `help:"New description of EIP"`
	}
	R(&EipUpdateOptions{}, "eip-update", "Update EIP properties", func(s *mcclient.ClientSession, args *EipUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		result, err := modules.Elasticips.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type EipAssociateOptions struct {
		ID           string `help:"ID or name of EIP"`
		INSTANCEID   string `help:"ID of instance the eip associated with"`
		InstanceType string `default:"server" help:"Instance type that the eip associated with, default is server" choices:"server"`
	}
	R(&EipAssociateOptions{}, "eip-associate", "Associate an EIP to an instance", func(s *mcclient.ClientSession, args *EipAssociateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.InstanceType), "instance_type")
		params.Add(jsonutils.NewString(args.INSTANCEID), "instance_id")
		result, err := modules.Elasticips.PerformAction(s, args.ID, "associate", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type EipSingleOptions struct {
		ID string `help:"ID or name of EIP"`
	}
	R(&EipSingleOptions{}, "eip-dissociate", "Dissociate an EIP from an instance", func(s *mcclient.ClientSession, args *EipSingleOptions) error {
		result, err := modules.Elasticips.PerformAction(s, args.ID, "dissociate", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&EipSingleOptions{}, "eip-sync", "Synchronize status of an EIP", func(s *mcclient.ClientSession, args *EipSingleOptions) error {
		result, err := modules.Elasticips.PerformAction(s, args.ID, "sync", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerCreateEipOptions struct {
		ID         string `help:"server ID or name"`
		BW         int    `help:"EIP bandwidth in Mbps"`
		ChargeType string `help:"bandwidth charge type, either traffic or bandwidth" choices:"traffic|bandwidth"`
	}
	R(&ServerCreateEipOptions{}, "server-create-eip", "allocate an EIP and associate EIP to server", func(s *mcclient.ClientSession, args *ServerCreateEipOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewInt(int64(args.BW)), "bandwidth")

		if len(args.ChargeType) > 0 {
			params.Add(jsonutils.NewString(args.ChargeType), "charge_type")
		}

		result, err := modules.Servers.PerformAction(s, args.ID, "create-eip", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type EipShowOptions struct {
		ID string `help:"ID or name of EIP"`
	}
	R(&EipShowOptions{}, "eip-show", "show details of an EIP", func(s *mcclient.ClientSession, args *EipShowOptions) error {
		result, err := modules.Servers.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type EipChangeBandwidthOptions struct {
		ID string `help:"ID or name of the EIP"`
		BW int    `help:"new bandwidth of EIP"`
	}
	R(&EipChangeBandwidthOptions{}, "eip-change-bandwidth", "Change maximal bandwidth of EIP", func(s *mcclient.ClientSession, args *EipChangeBandwidthOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewInt(int64(args.BW)), "bandwidth")
		result, err := modules.Elasticips.PerformAction(s, args.ID, "change-bandwidth", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
