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

	/**
	 * 查询短信配置信息
	 */
	type SmsConfigShowOptions struct {
		TYPE string `help:"type "`
	}
	R(&SmsConfigShowOptions{}, "sms-config-show", "Show sms config details",
		func(s *mcclient.ClientSession, args *SmsConfigShowOptions) error {
			result, err := modules.SmsConfigs.Get(s, args.TYPE, nil)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})

	/**
	 * 增加短信配置信息
	 */
	type SmsConfigCreateOptions struct {
		TYPE             string `help:"sms vendor"`
		ACCESSKEYID      string `help:"ACCESSKEYID for sms vendor"`
		ACCESSKEYSECRET  string `help:"ACCESSKEYSECRET for sms vendor"`
		SIGNATURE        string `help:"SIGNATURE for sms vendor"`
		SmsTemplateOne   string `help:"Sms TemplateOne"`
		SmsTemplateTwo   string `help:"Sms TemplateTwo"`
		SmsTemplateThree string `help:"Sms TemplateThree"`
		SmsCheckCode     string `help:"Sms Check Code "`
	}

	R(&SmsConfigCreateOptions{}, "sms-config-create", "Create a sms Config",
		func(s *mcclient.ClientSession, args *SmsConfigCreateOptions) error {
			params := jsonutils.NewDict()
			params.Add(jsonutils.NewString(args.TYPE), "type")
			params.Add(jsonutils.NewString(args.ACCESSKEYID), "access_key_id")
			params.Add(jsonutils.NewString(args.ACCESSKEYSECRET), "access_key_secret")
			params.Add(jsonutils.NewString(args.SIGNATURE), "signature")
			params.Add(jsonutils.NewString(args.SmsTemplateOne), "sms_template_one")
			params.Add(jsonutils.NewString(args.SmsTemplateTwo), "sms_template_two")
			params.Add(jsonutils.NewString(args.SmsTemplateThree), "sms_template_three")
			params.Add(jsonutils.NewString(args.SmsCheckCode), "sms_check_code")

			result, err := modules.SmsConfigs.Create(s, params)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})

	/**
	 * 修改
	 */
	type SmsConfigUpdateOptions struct {
		TYPE             string `help:"sms vendor"`
		ACCESSKEYID      string `help:"ACCESSKEYID for sms vendor"`
		ACCESSKEYSECRET  string `help:"ACCESSKEYSECRET for sms vendor"`
		SIGNATURE        string `help:"SIGNATURE for sms vendor"`
		SmsTemplateOne   string `help:"Sms TemplateOne"`
		SmsTemplateTwo   string `help:"Sms TemplateTwo"`
		SmsTemplateThree string `help:"Sms TemplateThree"`
		SmsCheckCode     string `help:"Sms Check Code "`
	}
	R(&SmsConfigUpdateOptions{}, "sms-config-update", "Update a sms-config", func(s *mcclient.ClientSession, args *SmsConfigUpdateOptions) error {
		params := jsonutils.NewDict()

		params.Add(jsonutils.NewString(args.TYPE), "type")
		params.Add(jsonutils.NewString(args.ACCESSKEYID), "access_key_id")
		params.Add(jsonutils.NewString(args.ACCESSKEYSECRET), "access_key_secret")
		params.Add(jsonutils.NewString(args.SIGNATURE), "signature")
		params.Add(jsonutils.NewString(args.SmsTemplateOne), "sms_template_one")
		params.Add(jsonutils.NewString(args.SmsTemplateTwo), "sms_template_two")
		params.Add(jsonutils.NewString(args.SmsTemplateThree), "sms_template_three")
		params.Add(jsonutils.NewString(args.SmsCheckCode), "sms_check_code")

		result, err := modules.SmsConfigs.Put(s, args.TYPE, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 删除
	 */
	type SmsConfigDeleteOptions struct {
		TYPE string `help:"sms vendor"`
	}
	R(&SmsConfigDeleteOptions{}, "sms-config-delete", "Delete a sms config", func(s *mcclient.ClientSession, args *SmsConfigDeleteOptions) error {
		result, e := modules.SmsConfigs.Delete(s, args.TYPE, nil)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

}
