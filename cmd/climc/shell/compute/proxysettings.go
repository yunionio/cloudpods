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

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.ProxySettingCreateOptions{}, "proxysetting-create", "Create proxysetting", func(s *mcclient.ClientSession, opts *options.ProxySettingCreateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		proxysetting, err := modules.ProxySettings.Create(s, params)
		if err != nil {
			return err
		}
		printObject(proxysetting)
		return nil
	})
	R(&options.ProxySettingGetOptions{}, "proxysetting-show", "Show proxysetting", func(s *mcclient.ClientSession, opts *options.ProxySettingGetOptions) error {
		proxysetting, err := modules.ProxySettings.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(proxysetting)
		return nil
	})
	R(&options.ProxySettingListOptions{}, "proxysetting-list", "List proxysettings", func(s *mcclient.ClientSession, opts *options.ProxySettingListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.ProxySettings.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ProxySettings.GetColumns(s))
		return nil
	})
	R(&options.ProxySettingUpdateOptions{}, "proxysetting-update", "Update proxysetting", func(s *mcclient.ClientSession, opts *options.ProxySettingUpdateOptions) error {
		params, err := options.StructToParams(opts)
		proxysetting, err := modules.ProxySettings.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(proxysetting)
		return nil
	})
	R(&options.ProxySettingDeleteOptions{}, "proxysetting-delete", "Delete proxysetting", func(s *mcclient.ClientSession, opts *options.ProxySettingDeleteOptions) error {
		proxysetting, err := modules.ProxySettings.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(proxysetting)
		return nil
	})
	R(&options.ProxySettingTestOptions{}, "proxysetting-test", "Test proxysetting", func(s *mcclient.ClientSession, opts *options.ProxySettingTestOptions) error {
		proxysetting, err := modules.ProxySettings.PerformAction(s, opts.ID, "test", nil)
		if err != nil {
			return err
		}
		printObject(proxysetting)
		return nil
	})
	R(&options.ProxySettingPublicOptions{}, "proxysetting-public", "Make proxysetting public", func(s *mcclient.ClientSession, opts *options.ProxySettingPublicOptions) error {
		params := jsonutils.Marshal(opts)
		result, err := modules.ProxySettings.PerformAction(s, opts.ID, "public", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&options.ProxySettingPrivateOptions{}, "proxysetting-private", "Make proxysetting private", func(s *mcclient.ClientSession, opts *options.ProxySettingPrivateOptions) error {
		params := jsonutils.Marshal(opts)
		result, err := modules.ProxySettings.PerformAction(s, opts.ID, "private", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
