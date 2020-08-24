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

package notifyv2

import (
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ConfigListOptions struct {
		options.BaseListOptions
	}
	R(&ConfigListOptions{}, "notify-config-list", "List notify config", func(s *mcclient.ClientSession, args *ConfigListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.NotifyConfig.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.NotifyConfig.GetColumns(s))
		return nil
	})
	type ConfigCreateOptions struct {
		TYPE    string   `help:"Type contact config"`
		Configs []string `help:"Config content, format: 'key:value'"`
	}
	R(&ConfigCreateOptions{}, "notify-config-create", "Create notify config", func(s *mcclient.ClientSession, args *ConfigCreateOptions) error {
		configs := jsonutils.NewDict()
		for _, kv := range args.Configs {
			index := strings.IndexByte(kv, ':')
			configs.Set(kv[:index], jsonutils.NewString(kv[index+1:]))
		}
		params := jsonutils.NewDict()
		params.Set("type", jsonutils.NewString(args.TYPE))
		params.Set("content", configs)
		ret, err := modules.NotifyConfig.Create(s, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&ConfigCreateOptions{}, "notify-config-update", "Update notify config", func(s *mcclient.ClientSession, args *ConfigCreateOptions) error {
		configs := jsonutils.NewDict()
		for _, kv := range args.Configs {
			index := strings.IndexByte(kv, ':')
			configs.Set(kv[:index], jsonutils.NewString(kv[index+1:]))
		}
		params := jsonutils.NewDict()
		params.Set("content", configs)

		id, err := configIdFromType(s, args.TYPE)
		if err != nil {
			return err
		}
		ret, err := modules.NotifyConfig.Update(s, id, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	type ConfigOptions struct {
		TYPE string `help:"Type contact config"`
	}
	R(&ConfigOptions{}, "notify-config-delete", "Delete notify config", func(s *mcclient.ClientSession, args *ConfigOptions) error {
		id, err := configIdFromType(s, args.TYPE)
		if err != nil {
			return err
		}
		ret, err := modules.NotifyConfig.Delete(s, id, nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&ConfigOptions{}, "notify-config-show", "Show notify config", func(s *mcclient.ClientSession, args *ConfigOptions) error {
		listParams := jsonutils.NewDict()
		listParams.Set("type", jsonutils.NewString(args.TYPE))
		list, err := modules.NotifyConfig.List(s, listParams)
		if err != nil {
			return err
		}
		data := list.Data[0]
		printObject(data)
		return nil
	})
}

func configIdFromType(s *mcclient.ClientSession, t string) (string, error) {
	listParams := jsonutils.NewDict()
	listParams.Set("type", jsonutils.NewString(t))
	list, err := modules.NotifyConfig.List(s, listParams)
	if err != nil {
		return "", err
	}
	id, err := list.Data[0].GetString("id")
	if err != nil {
		return "", err
	}
	return id, nil
}
