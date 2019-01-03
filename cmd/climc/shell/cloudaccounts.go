package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	type CloudaccountListOptions struct {
		options.BaseListOptions
	}
	R(&CloudaccountListOptions{}, "cloud-account-list", "List cloud accounts", func(s *mcclient.ClientSession, args *CloudaccountListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
		}
		result, err := modules.Cloudaccounts.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cloudaccounts.GetColumns(s))
		return nil
	})

	type CloudaccountCreateOptions struct {
		NAME      string `help:"Name of cloud account"`
		ACCOUNT   string `help:"Account to access the cloud account"`
		SECRET    string `help:"Secret to access the cloud account, clientId/clientScret for Azure"`
		PROVIDER  string `help:"Driver for cloud account" choices:"VMware|Aliyun|Azure|Qcloud|OpenStack|Huawei"`
		AccessURL string `helo:"hello" metavar:"Azure choices: <AzureGermanCloud、AzureChinaCloud、AzureUSGovernmentCloud、AzurePublicCloud>"`
		Desc      string `help:"Description"`
		Enabled   bool   `help:"Enabled the account automatically"`

		Import            bool `help:"Import all sub account automatically"`
		AutoSync          bool `help:"Enabled the account automatically"`
		AutoCreateProject bool `help:"Enable the account with same name project"`
	}
	R(&CloudaccountCreateOptions{}, "cloud-account-create", "Create a cloud account", func(s *mcclient.ClientSession, args *CloudaccountCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.ACCOUNT), "account")
		params.Add(jsonutils.NewString(args.SECRET), "secret")
		params.Add(jsonutils.NewString(args.PROVIDER), "provider")

		if args.Enabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		}

		if args.Import {
			params.Add(jsonutils.JSONTrue, "import")
			if args.AutoSync {
				params.Add(jsonutils.JSONTrue, "auto_sync")
			}
			if args.AutoCreateProject {
				params.Add(jsonutils.JSONTrue, "auto_create_project")
			}
		}

		if len(args.AccessURL) > 0 {
			params.Add(jsonutils.NewString(args.AccessURL), "access_url")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		result, err := modules.Cloudaccounts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountUpdateOptions struct {
		ID        string `help:"ID or Name of cloud account"`
		Name      string `help:"New name to update"`
		AccessUrl string `help:"New access url"`

		BalanceKey       string `help:"update cloud balance account key, such as Azure EA key"`
		RemoveBalanceKey bool   `help:"remove cloud blance account key"`

		Desc string `help:"Description"`
	}
	R(&CloudaccountUpdateOptions{}, "cloud-account-update", "Update a cloud account", func(s *mcclient.ClientSession, args *CloudaccountUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.AccessUrl) > 0 {
			params.Add(jsonutils.NewString(args.AccessUrl), "access_url")
		}
		if len(args.BalanceKey) > 0 {
			params.Add(jsonutils.NewString(args.BalanceKey), "balance_key")
		} else if args.RemoveBalanceKey {
			params.Add(jsonutils.NewString(""), "balance_key")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Cloudaccounts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountShowOptions struct {
		ID string `help:"ID or Name of cloud account"`
	}
	R(&CloudaccountShowOptions{}, "cloud-account-show", "Get details of a cloud account", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudaccountShowOptions{}, "cloud-account-delete", "Delete a cloud account", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudaccountShowOptions{}, "cloud-account-enable", "Enable cloud account", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudaccountShowOptions{}, "cloud-account-disable", "Disable cloud account", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CloudaccountShowOptions{}, "cloud-account-balance", "Get balance", func(s *mcclient.ClientSession, args *CloudaccountShowOptions) error {
		result, err := modules.Cloudaccounts.GetSpecific(s, args.ID, "balance", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountImportOptions struct {
		ID                string `help:"ID or Name of cloud account" json:"-"`
		AutoSync          bool   `help:"Import sub accounts with enabled status"`
		AutoCreateProject bool   `help:"Import sub account with project"`
	}
	R(&CloudaccountImportOptions{}, "cloud-account-import", "Import sub cloud account", func(s *mcclient.ClientSession, args *CloudaccountImportOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "import", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountUpdateCredentialOptions struct {
		ID      string `help:"ID or Name of cloud account"`
		ACCOUNT string `help:"new account"`
		SECRET  string `help:"new secret"`
	}
	R(&CloudaccountUpdateCredentialOptions{}, "cloud-account-update-credential", "Update credential of a cloud account", func(s *mcclient.ClientSession, args *CloudaccountUpdateCredentialOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.ACCOUNT), "account")
		params.Add(jsonutils.NewString(args.SECRET), "secret")

		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "update-credential", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CloudaccountSyncOptions struct {
		ID       string   `help:"ID or Name of cloud account"`
		Force    bool     `help:"Force sync no matter what"`
		FullSync bool     `help:"Synchronize everything"`
		Region   []string `help:"region to sync"`
		Zone     []string `help:"region to sync"`
		Host     []string `help:"region to sync"`
	}
	R(&CloudaccountSyncOptions{}, "cloud-account-sync", "Sync of a cloud account account", func(s *mcclient.ClientSession, args *CloudaccountSyncOptions) error {
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
		result, err := modules.Cloudaccounts.PerformAction(s, args.ID, "sync", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
