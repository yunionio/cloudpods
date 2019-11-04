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
	R(&options.BaseListOptions{}, "elastic-cache-list", "List elastisc cache instance", func(s *mcclient.ClientSession, opts *options.BaseListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCache.List(s, params)
		if err != nil {
			return err
		}

		printList(result, nil)
		return nil
	})

	R(&options.ElasticCacheCreateOptions{}, "elastic-cache-create", "Create elastisc cache instance", func(s *mcclient.ClientSession, opts *options.ElasticCacheCreateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}

		result, err := modules.ElasticCache.Create(s, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-restart", "Restart elastisc cache instance", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "restart", nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-flush-instance", "Flush elastisc cache instance", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "flush-instance", nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-delete", "Delete elastisc cache instance", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCache.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	type ElasticCacheChangeSpecOptions struct {
		options.ElasticCacheIdOptions
		Sku string `help:"elastic cache sku id"`
	}

	R(&ElasticCacheChangeSpecOptions{}, "elastic-cache-change-spec", "Change elastisc cache instance specification", func(s *mcclient.ClientSession, opts *ElasticCacheChangeSpecOptions) error {
		params := jsonutils.NewDict()
		params.Set("sku", jsonutils.NewString(opts.Sku))
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "change-spec", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	type ElasticCacheMainteananceTimeOptions struct {
		options.ElasticCacheIdOptions
		START_TIME string `help:"elastic cache sku maintenance start time,format: HH:mm"`
		END_TIME   string `help:"elastic cache sku maintenance end time, format: HH:mm"`
	}

	R(&ElasticCacheMainteananceTimeOptions{}, "elastic-cache-set-maintenance-time", "set elastisc cache instance maintenance time", func(s *mcclient.ClientSession, opts *ElasticCacheMainteananceTimeOptions) error {
		params := jsonutils.NewDict()
		start := fmt.Sprintf("%sZ", opts.START_TIME)
		end := fmt.Sprintf("%sZ", opts.END_TIME)
		params.Set("maintain_start_time", jsonutils.NewString(start))
		params.Set("maintain_end_time", jsonutils.NewString(end))
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "set-maintain-time", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-allocate-public-connect", "Allocate elastisc cache instance public access connection", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "allocate-public-connection", nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-enable-auth", "Enable elastisc cache instance auth", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
		params := jsonutils.NewDict()
		params.Set("auth_mode", jsonutils.NewString("on"))
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "update-auth-mode", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-disable-auth", "Disable elastisc cache instance auth", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
		params := jsonutils.NewDict()
		params.Set("auth_mode", jsonutils.NewString("off"))
		result, err := modules.ElasticCache.PerformAction(s, opts.ID, "update-auth-mode", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	type ElasticCacheAccountResetPasswordOptions struct {
		options.ElasticCacheIdOptions
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

	R(&options.ElasticCacheAccountCreateOptions{}, "elastic-cache-account-create", "Create elastisc cache account", func(s *mcclient.ClientSession, opts *options.ElasticCacheAccountCreateOptions) error {
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

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-account-delete", "Delete elastisc cache account", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCacheAccount.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.ElasticCacheBackupCreateOptions{}, "elastic-cache-backup-create", "Create elastisc cache backup", func(s *mcclient.ClientSession, opts *options.ElasticCacheBackupCreateOptions) error {
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

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-backup-delete", "Delete elastisc cache backup", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCacheBackup.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-backup-restore", "Restore elastisc cache backup", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
		result, err := modules.ElasticCacheBackup.PerformAction(s, opts.ID, "restore-instance", nil)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	R(&options.ElasticCacheAclCreateOptions{}, "elastic-cache-acl-create", "Create elastisc cache acl", func(s *mcclient.ClientSession, opts *options.ElasticCacheAclCreateOptions) error {
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

	R(&options.ElasticCacheIdOptions{}, "elastic-cache-acl-delete", "Delete elastisc cache acl", func(s *mcclient.ClientSession, opts *options.ElasticCacheIdOptions) error {
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

	R(&options.ElasticCacheAclUpdateOptions{}, "elastic-cache-acl-update", "Update elastisc cache acl", func(s *mcclient.ClientSession, opts *options.ElasticCacheAclUpdateOptions) error {
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

	R(&options.ElasticCacheParameterUpdateOptions{}, "elastic-cache-parameter-update", "Update elastisc cache parameter", func(s *mcclient.ClientSession, opts *options.ElasticCacheParameterUpdateOptions) error {
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
}
