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

package meter

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	type CloudSkuRateListOptions struct {
		PARAMKEYS string `help:"param_keys like key1$key2$key3, key=provider(lowercase)::region::name"`
	}

	R(&CloudSkuRateListOptions{}, "cloud-sku-rate-list", "list cloud-sku-rates", func(s *mcclient.ClientSession, args *CloudSkuRateListOptions) error {

		params := jsonutils.NewDict()

		params.Add(jsonutils.NewString(args.PARAMKEYS), "param_keys")

		result, err := modules.CloudSkuRates.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.CloudSkuRates.GetColumns(s))
		return nil
	})
}
