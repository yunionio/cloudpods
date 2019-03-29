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
	 * 查询邮件配置信息
	 */
	type EmailConfigShowOptions struct {
		TYPE string `help:"type "`
	}
	R(&EmailConfigShowOptions{}, "email-config-show", "Show email-config details",
		func(s *mcclient.ClientSession, args *EmailConfigShowOptions) error {
			result, err := modules.EmailConfigs.Get(s, args.TYPE, nil)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})

	/**
	 * 增加邮件配置信息
	 */
	type EmailConfigCreateOptions struct {
		USERNAME  string `help:"Username for email sender"`
		PASSWORD  string `help:"Password for email sender"`
		HOSTNAME  string `help:"Email server name"`
		SSLGLOBAL string `help:"use ssl_global"`
		HOSTPORT  int64  `help:"Email server port"`
	}

	R(&EmailConfigCreateOptions{}, "email-config-create", "Create a Email Config",
		func(s *mcclient.ClientSession, args *EmailConfigCreateOptions) error {
			params := jsonutils.NewDict()
			params.Add(jsonutils.NewString(args.USERNAME), "username")
			params.Add(jsonutils.NewString(args.PASSWORD), "password")
			params.Add(jsonutils.NewString(args.HOSTNAME), "hostname")
			params.Add(jsonutils.NewString(args.SSLGLOBAL), "ssl_global")
			params.Add(jsonutils.NewInt(args.HOSTPORT), "hostport")

			result, err := modules.EmailConfigs.Create(s, params)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})

	/**
	 * 修改
	 */
	type EmailConfigUpdateOptions struct {
		TYPE      string `help:"type of email "`
		USERNAME  string `help:"Username for email sender"`
		PASSWORD  string `help:"Password for email sender"`
		HOSTNAME  string `help:"Email server name"`
		SSLGLOBAL string `help:"use ssl_global"`
		HOSTPORT  int64  `help:"Email server port"`
	}
	R(&EmailConfigUpdateOptions{}, "email-config-update", "Update a email-config", func(s *mcclient.ClientSession, args *EmailConfigUpdateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.USERNAME), "username")
		params.Add(jsonutils.NewString(args.PASSWORD), "password")
		params.Add(jsonutils.NewString(args.HOSTNAME), "hostname")
		params.Add(jsonutils.NewString(args.SSLGLOBAL), "ssl_global")
		params.Add(jsonutils.NewInt(args.HOSTPORT), "hostport")

		result, err := modules.EmailConfigs.Put(s, args.TYPE, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 删除
	 */
	type EmailConfigDeleteOptions struct {
		TYPE string `help:"type of email "`
	}
	R(&EmailConfigDeleteOptions{}, "email-config-delete", "Delete a email config", func(s *mcclient.ClientSession, args *EmailConfigDeleteOptions) error {
		result, e := modules.EmailConfigs.Delete(s, args.TYPE, nil)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

}
