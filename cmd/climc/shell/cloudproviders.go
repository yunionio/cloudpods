package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	type CloudproviderListOptions struct {
		BaseListOptions
	}
	R(&CloudproviderListOptions{}, "cloud-provider-list", "List cloud providers", func(s *mcclient.ClientSession, args *CloudproviderListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		result, err := modules.Cloudproviders.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cloudproviders.GetColumns(s))
		return nil
	})

	type CloudproviderCreateOptions struct {
		NAME      string `help:"Name of cloud provider"`
		ACCOUNT   string `help:"Account to access the cloud provider"`
		SECRET    string `help:"Secret to access the cloud provider"`
		PROVIDER  string `help:"Driver for cloud provider" choices:"VMware|Aliyun"`
		AccessUrl string `help:"Access url"`
		Desc      string `help:"Description"`
		Enabled   bool   `help:"Enabled the provider automatically"`
	}
	R(&CloudproviderCreateOptions{}, "cloud-provider-create", "Create a cloud provider", func(s *mcclient.ClientSession, args *CloudproviderCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.ACCOUNT), "account")
		params.Add(jsonutils.NewString(args.SECRET), "secret")
		params.Add(jsonutils.NewString(args.PROVIDER), "provider")
		if args.Enabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		}
		if len(args.AccessUrl) > 0 {
			params.Add(jsonutils.NewString(args.AccessUrl), "access_url")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		result, err := modules.Cloudproviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
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
		ID       string   `help:"ID or Name of cloud provider"`
		Force    bool     `help:"Force sync no matter what"`
		FullSync bool     `help:"Synchronize everything"`
		Region   []string `help:"region to sync"`
		Zone     []string `help:"region to sync"`
		Host     []string `help:"region to sync"`
	}
	R(&CloudproviderSyncOptions{}, "cloud-provider-sync", "Sync of a cloud provider account", func(s *mcclient.ClientSession, args *CloudproviderSyncOptions) error {
		params := jsonutils.NewDict()
		if args.Force {
			params.Add(jsonutils.JSONTrue, "force")
		}
		if args.FullSync {
			params.Add(jsonutils.JSONTrue, "full_sync")
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
}
