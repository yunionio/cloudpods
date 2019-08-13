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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type CloudproviderListOptions struct {
		options.BaseListOptions

		Usable bool `help:"Vpc & Network usable"`

		HasObjectStorage bool `help:"filter cloudproviders that has object storage"`
		NoObjectStorage  bool `help:"filter cloudproviders that has no object storage"`
	}
	R(&CloudproviderListOptions{}, "cloud-provider-list", "List cloud providers", func(s *mcclient.ClientSession, args *CloudproviderListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}

			if args.Usable {
				params.Add(jsonutils.NewBool(true), "usable")
			}

			if args.HasObjectStorage {
				params.Add(jsonutils.JSONTrue, "has_object_storage")
			} else if args.NoObjectStorage {
				params.Add(jsonutils.JSONFalse, "has_object_storage")
			}
		}
		result, err := modules.Cloudproviders.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cloudproviders.GetColumns(s))
		return nil
	})

	type CloudproviderUpdateOptions struct {
		ID        string `help:"ID or Name of cloud provider"`
		Name      string `help:"New name to update"`
		AccessUrl string `help:"New access url"`
		Desc      string `help:"Description"`
	}
	R(&CloudproviderUpdateOptions{}, "cloud-provider-update", "Update a cloud provider", func(s *mcclient.ClientSession, args *CloudproviderUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.AccessUrl) > 0 {
			params.Add(jsonutils.NewString(args.AccessUrl), "access_url")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudproviders.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudproviderChangeProjectOptions struct {
		ID     string `help:"ID or Name of cloud provider"`
		TENANT string `help:"ID or Name of tenant"`
	}
	R(&CloudproviderChangeProjectOptions{}, "cloud-provider-change-project", "Change project for provider", func(s *mcclient.ClientSession, args *CloudproviderChangeProjectOptions) error {
		result, err := modules.Cloudproviders.PerformAction(s, args.ID, "change-project", jsonutils.Marshal(map[string]string{"project": args.TENANT}))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudproviderShowOptions struct {
		ID string `help:"ID or Name of cloud provider"`
	}
	R(&CloudproviderShowOptions{}, "cloud-provider-show", "Get details of a cloud provider", func(s *mcclient.ClientSession, args *CloudproviderShowOptions) error {
		result, err := modules.Cloudproviders.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudproviderShowOptions{}, "cloud-provider-delete", "Delete a cloud provider", func(s *mcclient.ClientSession, args *CloudproviderShowOptions) error {
		result, err := modules.Cloudproviders.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudproviderShowOptions{}, "cloud-provider-enable", "Enable cloud provider", func(s *mcclient.ClientSession, args *CloudproviderShowOptions) error {
		result, err := modules.Cloudproviders.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudproviderShowOptions{}, "cloud-provider-disable", "Disable cloud provider", func(s *mcclient.ClientSession, args *CloudproviderShowOptions) error {
		result, err := modules.Cloudproviders.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudproviderShowOptions{}, "cloud-provider-balance", "Get balance", func(s *mcclient.ClientSession, args *CloudproviderShowOptions) error {
		result, err := modules.Cloudproviders.GetSpecific(s, args.ID, "balance", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudproviderUpdateCredentialOptions struct {
		ID      string `help:"ID or Name of cloud provider"`
		ACCOUNT string `help:"new account"`
		SECRET  string `help:"new secret"`
	}
	R(&CloudproviderUpdateCredentialOptions{}, "cloud-provider-update-credential", "Update credential of a cloud provider", func(s *mcclient.ClientSession, args *CloudproviderUpdateCredentialOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.ACCOUNT), "account")
		params.Add(jsonutils.NewString(args.SECRET), "secret")

		result, err := modules.Cloudproviders.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudproviderSyncOptions struct {
		ID          string   `help:"ID or Name of cloud provider"`
		Force       bool     `help:"Force sync no matter what"`
		FullSync    bool     `help:"Synchronize everything"`
		ProjectSync bool     `help:"Auto sync project info"`
		Region      []string `help:"region to sync"`
		Zone        []string `help:"region to sync"`
		Host        []string `help:"region to sync"`
	}
	R(&CloudproviderSyncOptions{}, "cloud-provider-sync", "Sync of a cloud provider account", func(s *mcclient.ClientSession, args *CloudproviderSyncOptions) error {
		params := jsonutils.NewDict()
		if args.Force {
			params.Add(jsonutils.JSONTrue, "force")
		}
		if args.FullSync {
			params.Add(jsonutils.JSONTrue, "full_sync")
		}
		if args.ProjectSync {
			params.Add(jsonutils.JSONTrue, "project_sync")
		}
		if len(args.Region) > 0 {
			params.Add(jsonutils.NewStringArray(args.Region), "region")
		}
		if len(args.Zone) > 0 {
			params.Add(jsonutils.NewStringArray(args.Zone), "zone")
		}
		if len(args.Host) > 0 {
			params.Add(jsonutils.NewStringArray(args.Host), "host")
		}
		result, err := modules.Cloudproviders.PerformAction(s, args.ID, "sync", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudproviderClientRCOptions struct {
		ID string `help:"ID or name of cloud provider"`
	}
	R(&CloudproviderClientRCOptions{}, "cloud-provider-clirc", "Get client RC file of the cloud provider", func(s *mcclient.ClientSession, args *CloudproviderClientRCOptions) error {
		result, err := modules.Cloudproviders.GetSpecific(s, args.ID, "clirc", nil)
		if err != nil {
			return err
		}
		rc := make(map[string]string)
		err = result.Unmarshal(&rc)
		if err != nil {
			return err
		}
		for k, v := range rc {
			fmt.Printf("export %s='%s'\n", k, v)
		}
		return nil
	})

	type CloudproviderStorageClassesOptions struct {
		ID          string `help:"ID or Name of cloud provider" json:"-"`
		Cloudregion string `help:"cloud region name or Id"`
	}
	R(&CloudproviderStorageClassesOptions{}, "cloud-provider-storage-classes", "Get list of supported storage classes of a cloud provider", func(s *mcclient.ClientSession, args *CloudproviderStorageClassesOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Cloudproviders.GetSpecific(s, args.ID, "storage-classes", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
