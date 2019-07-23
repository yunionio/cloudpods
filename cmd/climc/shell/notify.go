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
)

func init() {

	type ConfigCreate2Options struct {
		CONTACTTYPE string   `help:"contact type (email|sms_aliyun or others)"`
		CONFIGS     []string `help:"config (k, v)"`
	}
	R(&ConfigCreate2Options{}, "notify-config-update", "config update, example: notify_config-update email mail.smtp.hostname hostname mail.smtp.hostport 123.", func(s *mcclient.ClientSession, args *ConfigCreate2Options) error {
		tmp := jsonutils.NewDict()
		for i := 0; i+1 < len(args.CONFIGS); i += 2 {
			tmp.Add(jsonutils.NewString(args.CONFIGS[i+1]), args.CONFIGS[i])
		}
		body := jsonutils.NewDict()
		body.Add(tmp, args.CONTACTTYPE)
		modules.Configs.Create(s, body)
		return nil
	})

	type ConfigGet2Options struct {
		TYPE string `help:"contact type (email|sms_aliyun or others)"`
	}
	R(&ConfigGet2Options{}, "notify-config-show", "config show", func(s *mcclient.ClientSession, args *ConfigGet2Options) error {
		result, err := modules.Configs.Get(s, args.TYPE, jsonutils.JSONNull)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&ConfigGet2Options{}, "notify-config-delete", "config delete", func(s *mcclient.ClientSession, args *ConfigGet2Options) error {
		result, err := modules.Configs.Delete(s, args.TYPE, jsonutils.JSONNull)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
