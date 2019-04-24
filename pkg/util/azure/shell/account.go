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
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type AccountListOptions struct {
	}
	shellutils.R(&AccountListOptions{}, "account-list", "List sub account", func(cli *azure.SRegion, args *AccountListOptions) error {
		if accounts, err := cli.GetClient().GetSubAccounts(); err != nil {
			return err
		} else {
			printObject(accounts)
			return nil
		}
	})
}
