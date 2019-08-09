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
	type ElasticipListOptions struct {
		Region string `help:"List eips in cloudregion"`

		Usable                    *bool  `help:"List all zones that is usable"`
		UsableEipForAssociateType string `help:"With associate id filter which eip can associate"`
		UsableEipForAssociateId   string `help:"With associate type filter which eip can associate"`

		options.BaseListOptions
	}
	R(&ElasticipListOptions{}, "eip-list", "List elastic IPs", func(s *mcclient.ClientSession, opts *ElasticipListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
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
		Bandwidth  int    `help:"Bandwidth in Mbps"`
		Ip         string `help:"IP address of the EIP"`
		Network    string `help:"Network of the EIP"`
		ChargeType string `help:"bandwidth charge type" choices:"traffic|bandwidth"`
	}
	R(&EipCreateOptions{}, "eip-create", "Create an EIP", func(s *mcclient.ClientSession, args *EipCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.MANAGER), "manager")
		params.Add(jsonutils.NewString(args.REGION), "region")
		params.Add(jsonutils.NewString(args.NAME), "name")
		if args.Bandwidth != 0 {
			params.Add(jsonutils.NewInt(int64(args.Bandwidth)), "bandwidth")
		}

		if len(args.ChargeType) > 0 {
			params.Add(jsonutils.NewString(args.ChargeType), "charge_type")
		}

		if len(args.Network) > 0 {
			params.Add(jsonutils.NewString(args.Network), "network")
		}

		if len(args.Ip) > 0 {
			params.Add(jsonutils.NewString(args.Ip), "ip")
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

		EnableAutoDellocate  bool `help:"enable automatically dellocate when dissociate from instance"`
		DisableAutoDellocate bool `help:"disable automatically dellocate when dissociate from instance"`
	}
	R(&EipUpdateOptions{}, "eip-update", "Update EIP properties", func(s *mcclient.ClientSession, args *EipUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.EnableAutoDellocate {
			params.Add(jsonutils.JSONTrue, "auto_dellocate")
		} else if args.DisableAutoDellocate {
			params.Add(jsonutils.JSONFalse, "auto_dellocate")
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

	type EipDissociateOptions struct {
		ID         string `help:"ID or name of EIP"`
		AutoDelete bool   `help:"automatically delete the dissociate EIP" json:"auto_delete,omitfalse"`
	}
	R(&EipDissociateOptions{}, "eip-dissociate", "Dissociate an EIP from an instance", func(s *mcclient.ClientSession, args *EipDissociateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Elasticips.PerformAction(s, args.ID, "dissociate", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type EipSingleOptions struct {
		ID string `help:"ID or name of EIP"`
	}
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
		ChargeType string `help:"bandwidth charge type" choices:"traffic|bandwidth"`
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
		result, err := modules.Elasticips.Get(s, args.ID, nil)
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

	type EipPurgeOptions struct {
		ID string `help:"ID or name of EIP"`
	}
	R(&EipPurgeOptions{}, "eip-purge", "Purge EIP db records", func(s *mcclient.ClientSession, args *EipPurgeOptions) error {
		result, err := modules.Elasticips.PerformAction(s, args.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type EipChangeOwnerOptions struct {
		ID      string `help:"EIP to change owner"`
		PROJECT string `help:"Project ID or change"`
		RawId   bool   `help:"User raw ID, instead of name"`
	}
	R(&EipChangeOwnerOptions{}, "eip-change-owner", "Change owner porject of a eip", func(s *mcclient.ClientSession, opts *EipChangeOwnerOptions) error {
		params := jsonutils.NewDict()
		if opts.RawId {
			projid, err := modules.Projects.GetId(s, opts.PROJECT, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(projid), "tenant")
			params.Add(jsonutils.JSONTrue, "raw_id")
		} else {
			params.Add(jsonutils.NewString(opts.PROJECT), "tenant")
		}
		srv, err := modules.Elasticips.PerformAction(s, opts.ID, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

}
