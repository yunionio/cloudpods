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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.ElasticCache).WithKeyword("elastic-cache")
	cmd.List(&compute.ElasticCacheListOptions{})
	cmd.Show(&compute.ElasticCacheIdOption{})
	cmd.Create(&compute.ElasticCacheCreateOptions{})
	cmd.Delete(&compute.ElasticCacheIdOption{})
	cmd.Perform("restart", &compute.ElasticCacheIdOption{})
	cmd.Perform("flush-instance", &compute.ElasticCacheIdOption{})
	cmd.Perform("syncstatus", &compute.ElasticCacheIdOption{})
	cmd.Perform("sync", &compute.ElasticCacheIdOption{})
	cmd.Perform("change-spec", &compute.ElasticCacheChangeSpecOptions{})
	cmd.Perform("set-maintenance-time", &compute.ElasticCacheMainteananceTimeOptions{})
	cmd.Perform("allocate-public-connection", &compute.ElasticCacheIdOption{})
	cmd.PerformWithKeyword("enable-auth", "update-auth-mode", &compute.ElasticCacheEnableAuthOptions{})
	cmd.PerformWithKeyword("disable-auth", "update-auth-mode", &compute.ElasticCacheDisableAuthOptions{})

	type ElasticCacheAccountResetPasswordOptions struct {
		compute.ElasticCacheIdOptions
		PASSWORD string `help:"elastic cache account password."`
	}

	R(&ElasticCacheAccountResetPasswordOptions{}, "elastic-cache-account-reset-password", "Reset elastisc cache instance account password", func(s *mcclient.ClientSession, opts *ElasticCacheAccountResetPasswordOptions) error {
		params := jsonutils.NewDict()
		params.Set("password", jsonutils.NewString(opts.PASSWORD))
		result, err := modules.ElasticCacheAccount.PerformAction(s, opts.ID, "reset-password", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	type ElasticCacheBaseListOptions struct {
		options.BaseListOptions
		ElasticcacheId string `help:"elastic cache id"`
	}

	R(&ElasticCacheBaseListOptions{}, "elastic-cache-account-list", "List elastisc cache account", func(s *mcclient.ClientSession, opts *ElasticCacheBaseListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCacheAccount.List(s, params)
		if err != nil {
			return err
		}

		printList(result, nil)
		return nil
	})

	R(&compute.ElasticCacheAccountCreateOptions{}, "elastic-cache-account-create", "Create elastisc cache account", func(s *mcclient.ClientSession, opts *compute.ElasticCacheAccountCreateOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCacheAccount.Create(s, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&compute.ElasticCacheIdOptions{}, "elastic-cache-account-delete", "Delete elastisc cache account", func(s *mcclient.ClientSession, opts *compute.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCacheAccount.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&compute.ElasticCacheBackupCreateOptions{}, "elastic-cache-backup-create", "Create elastisc cache backup", func(s *mcclient.ClientSession, opts *compute.ElasticCacheBackupCreateOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCacheBackup.Create(s, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.BaseListOptions{}, "elastic-cache-backup-list", "List elastisc cache backup", func(s *mcclient.ClientSession, opts *options.BaseListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCacheBackup.List(s, params)
		if err != nil {
			return err
		}

		printList(result, nil)
		return nil
	})

	R(&compute.ElasticCacheIdOptions{}, "elastic-cache-backup-delete", "Delete elastisc cache backup", func(s *mcclient.ClientSession, opts *compute.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCacheBackup.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&compute.ElasticCacheIdOptions{}, "elastic-cache-backup-restore", "Restore elastisc cache backup", func(s *mcclient.ClientSession, opts *compute.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCacheBackup.PerformAction(s, opts.ID, "restore-instance", nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&compute.ElasticCacheAclCreateOptions{}, "elastic-cache-acl-create", "Create elastisc cache acl", func(s *mcclient.ClientSession, opts *compute.ElasticCacheAclCreateOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCacheAcl.Create(s, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&compute.ElasticCacheIdOptions{}, "elastic-cache-acl-delete", "Delete elastisc cache acl", func(s *mcclient.ClientSession, opts *compute.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCacheAcl.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.BaseListOptions{}, "elastic-cache-acl-list", "List elastisc cache acl", func(s *mcclient.ClientSession, opts *options.BaseListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCacheAcl.List(s, params)
		if err != nil {
			return err
		}

		printList(result, nil)
		return nil
	})

	R(&compute.ElasticCacheAclUpdateOptions{}, "elastic-cache-acl-update", "Update elastisc cache acl", func(s *mcclient.ClientSession, opts *compute.ElasticCacheAclUpdateOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		params.Remove("id")
		result, err := modules.ElasticCacheAcl.Update(s, opts.Id, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.BaseListOptions{}, "elastic-cache-parameter-list", "List elastisc cache parameters", func(s *mcclient.ClientSession, opts *options.BaseListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCacheParameter.List(s, params)
		if err != nil {
			return err
		}

		printList(result, nil)
		return nil
	})

	R(&compute.ElasticCacheParameterUpdateOptions{}, "elastic-cache-parameter-update", "Update elastisc cache parameter", func(s *mcclient.ClientSession, opts *compute.ElasticCacheParameterUpdateOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		params.Remove("id")
		result, err := modules.ElasticCacheParameter.Update(s, opts.Id, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	type ElasticCacheSkusListOptions struct {
		options.BaseListOptions
		Cloudregion   string  `help:"region Id or name"`
		Usable        bool    `help:"Filter usable sku"`
		Zone          string  `help:"zone Id or name"`
		City          *string `help:"city name,eg. BeiJing"`
		LocalCategory *string `help:"local category,eg. single"`
		EngineVersion *string `help:"engine version,eg. 3.0"`
		Cpu           *int    `help:"Cpu core count" json:"cpu_core_count"`
		Mem           *int    `help:"Memory size in MB" json:"memory_size_mb"`
		Name          string  `help:"Name of Sku"`
	}

	R(&ElasticCacheSkusListOptions{}, "elastic-cache-sku-list", "List elastisc cache sku", func(s *mcclient.ClientSession, opts *ElasticCacheSkusListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticcacheSkus.List(s, params)
		if err != nil {
			return err
		}

		printList(result, nil)
		return nil
	})

	R(&options.ResourceMetadataOptions{}, "elastic-cache-add-tag", "Set tag of a server", func(s *mcclient.ClientSession, opts *options.ResourceMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "user-metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ResourceMetadataOptions{}, "elastic-cache-set-tag", "Set tag of a server", func(s *mcclient.ClientSession, opts *options.ResourceMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "set-user-metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&compute.ElasticCacheRemoteUpdateOptions{}, "elastic-cache-remote-update", "Restore elastisc cache backup", func(s *mcclient.ClientSession, opts *compute.ElasticCacheRemoteUpdateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "remote-update", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&compute.ElasticCacheAutoRenewOptions{}, "elastic-cache-auto-renew", "Set elastisc cache auto renew", func(s *mcclient.ClientSession, opts *compute.ElasticCacheAutoRenewOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "set-auto-renew", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&compute.ElasticCacheRenewOptions{}, "elastic-cache-renew", "Renew elastisc cache", func(s *mcclient.ClientSession, opts *compute.ElasticCacheRenewOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "renew", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	type ElasticcacheSecgroupListOptions struct {
		options.BaseListOptions
		Elasticcache string `help:"ID or Name of elastic cache" json:"elasticcache"`
		Secgroup     string `help:"Secgroup ID or name"`
	}

	R(&ElasticcacheSecgroupListOptions{}, "elastic-cache-secgroup-list", "List elastisc cache secgroups", func(s *mcclient.ClientSession, opts *ElasticcacheSecgroupListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCacheSecgroup.List(s, params)
		if err != nil {
			return err
		}

		printList(result, nil)
		return nil
	})
}
