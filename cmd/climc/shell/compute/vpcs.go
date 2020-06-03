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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type VpcListOptions struct {
		options.BaseListOptions

		Usable             *bool  `help:"Filter usable vpcs"`
		Region             string `help:"ID or Name of region" json:"-"`
		Globalvpc          string `help:"Filter by globalvpc"`
		ExternalAccessMode string `help:"Filter by external access mode" choices:"distgw|eip|eip-distgw"`
	}
	R(&VpcListOptions{}, "vpc-list", "List VPCs", func(s *mcclient.ClientSession, opts *VpcListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		var result *modulebase.ListResult
		if len(opts.Region) > 0 {
			result, err = modules.Vpcs.ListInContext(s, params, &modules.Cloudregions, opts.Region)
		} else {
			result, err = modules.Vpcs.List(s, params)
		}
		if err != nil {
			return err
		}

		if len(opts.ExportFile) > 0 {
			exportList(result, opts.ExportFile, opts.ExportKeys, opts.ExportTexts, modules.Vpcs.GetColumns(s))
		} else {
			printList(result, modules.Vpcs.GetColumns(s))
		}
		return nil
	})

	type VpcCreateOptions struct {
		REGION             string `help:"ID or name of the region where the VPC is created"`
		Id                 string `help:"ID of the new VPC"`
		NAME               string `help:"Name of the VPC"`
		CIDR               string `help:"CIDR block"`
		Default            bool   `help:"default VPC for the region" default:"false"`
		Desc               string `help:"Description of the VPC"`
		Manager            string `help:"ID or Name of Cloud provider"`
		ExternalAccessMode string `help:"Filter by external access mode" choices:"distgw|eip|eip-distgw" default:"eip-distgw"`
	}
	R(&VpcCreateOptions{}, "vpc-create", "Create a VPC", func(s *mcclient.ClientSession, args *VpcCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.CIDR), "cidr_block")
		if len(args.Id) > 0 {
			params.Add(jsonutils.NewString(args.Id), "id")
		}
		if len(args.ExternalAccessMode) > 0 {
			params.Add(jsonutils.NewString(args.ExternalAccessMode), "external_access_mode")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Default {
			params.Add(jsonutils.JSONTrue, "is_default")
		}
		if len(args.Manager) > 0 {
			params.Add(jsonutils.NewString(args.Manager), "manager")
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
		ID                 string `help:"ID or name of the VPC"`
		Name               string `help:"New name of the VPC"`
		Desc               string `help:"Description of the VPC"`
		ExternalAccessMode string `help:"Filter by external access mode" choices:"distgw|eip|eip-distgw"`
	}
	R(&VpcUpdateOptions{}, "vpc-update", "Update a VPC", func(s *mcclient.ClientSession, args *VpcUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.ExternalAccessMode) > 0 {
			params.Add(jsonutils.NewString(args.ExternalAccessMode), "external_access_mode")
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

	R(&VpcUpdateStatusOptions{}, "vpc-purge", "Purge a managed VPC, not delete the remote entity", func(s *mcclient.ClientSession, args *VpcUpdateStatusOptions) error {
		result, err := modules.Vpcs.PerformAction(s, args.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&VpcUpdateStatusOptions{}, "vpc-sync", "Synchronize the status of a vpc", func(s *mcclient.ClientSession, args *VpcUpdateStatusOptions) error {
		result, err := modules.Vpcs.PerformAction(s, args.ID, "sync", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&VpcUpdateStatusOptions{}, "vpc-syncstatus", "Sync the status of a vpc", func(s *mcclient.ClientSession, args *VpcUpdateStatusOptions) error {
		result, err := modules.Vpcs.PerformAction(s, args.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type VpcPublicOptions struct {
		ID            string   `help:"ID or name of vpc" json:"-"`
		Scope         string   `help:"sharing scope" choices:"system|domain"`
		SharedDomains []string `help:"share to domains"`
	}
	R(&VpcPublicOptions{}, "vpc-public", "Make vpc public", func(s *mcclient.ClientSession, args *VpcPublicOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Vpcs.PerformAction(s, args.ID, "public", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type VpcPrivateOptions struct {
		ID string `help:"ID or name of vpc" json:"-"`
	}
	R(&VpcPrivateOptions{}, "vpc-private", "Make vpc private", func(s *mcclient.ClientSession, args *VpcPrivateOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Vpcs.PerformAction(s, args.ID, "private", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&VpcShowOptions{}, "vpc-change-owner-candidate-domains", "Show candiate domains of a vpc for changing owner", func(s *mcclient.ClientSession, args *VpcShowOptions) error {
		result, err := modules.Vpcs.GetSpecific(s, args.ID, "change-owner-candidate-domains", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type VpcChangeOwnerOptions struct {
		ID            string `help:"ID or name of vpc" json:"-"`
		ProjectDomain string `json:"project_domain" help:"target domain"`
	}
	R(&VpcChangeOwnerOptions{}, "vpc-change-owner", "Change owner domain of vpc", func(s *mcclient.ClientSession, args *VpcChangeOwnerOptions) error {
		if len(args.ProjectDomain) == 0 {
			return fmt.Errorf("empty project_domain")
		}
		params := jsonutils.Marshal(args)
		ret, err := modules.Vpcs.PerformAction(s, args.ID, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
