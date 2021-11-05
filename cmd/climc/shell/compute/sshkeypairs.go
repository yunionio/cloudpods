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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

func init() {
	type SshkeypairQueryOptions struct {
		Project string `help:"get keypair for specific project"`
		Admin   bool   `help:"get admin keypair, sysadmin ONLY option"`
	}
	R(&SshkeypairQueryOptions{}, "sshkeypair-show", "Get ssh keypairs", func(s *mcclient.ClientSession, args *SshkeypairQueryOptions) error {
		query := jsonutils.NewDict()
		if args.Admin {
			query.Add(jsonutils.JSONTrue, "admin")
		}
		var keys jsonutils.JSONObject
		if len(args.Project) == 0 {
			listResult, err := modules.Sshkeypairs.List(s, query)
			if err != nil {
				return err
			}
			keys = listResult.Data[0]
		} else {
			result, err := modules.Sshkeypairs.GetById(s, args.Project, query)
			if err != nil {
				return err
			}
			keys = result
		}
		privKey, _ := keys.GetString("private_key")
		pubKey, _ := keys.GetString("public_key")

		fmt.Print(privKey)
		fmt.Print(pubKey)

		return nil
	})
}
