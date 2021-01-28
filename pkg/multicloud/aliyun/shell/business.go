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
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type AccountBalanceOptions struct {
	}
	shellutils.R(&AccountBalanceOptions{}, "balance", "Get account balance", func(cli *aliyun.SRegion, args *AccountBalanceOptions) error {
		result1, err := cli.GetClient().QueryAccountBalance()
		if err != nil {
			return err
		}
		printObject(result1)

		result2, err := cli.GetClient().QueryCashCoupons()
		if err != nil {
			return err
		}
		if len(result2) > 0 {
			printList(result2, len(result2), 0, 0, nil)
		}

		result3, err := cli.GetClient().QueryPrepaidCards()
		if err != nil {
			return err
		}
		if len(result3) > 0 {
			printList(result3, len(result3), 0, 0, nil)
		}
		return nil
	})

	type AliyunSubscribeBillOptions struct {
		BUCKET string `help:"bucket name to store billing records"`
	}
	shellutils.R(&AliyunSubscribeBillOptions{}, "subscribe-bill", "Subscribe bill to OSS storage", func(cli *aliyun.SRegion, args *AliyunSubscribeBillOptions) error {
		err := cli.GetClient().SubscribeBillToOSS(args.BUCKET)
		if err != nil {
			return err
		}
		return nil
	})

}
