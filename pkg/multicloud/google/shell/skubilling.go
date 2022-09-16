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
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SkuBillingListOptions struct {
		PageSize  int
		PageToken string
	}
	shellutils.R(&SkuBillingListOptions{}, "sku-billing-list", "List sku billing", func(cli *google.SRegion, args *SkuBillingListOptions) error {
		billings, err := cli.ListSkuBilling(args.PageSize, args.PageToken)
		if err != nil {
			return err
		}
		printList(billings, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&SkuBillingListOptions{}, "compute-sku-billing-list", "List sku billing", func(cli *google.SRegion, args *SkuBillingListOptions) error {
		billings, err := cli.ListSkuBilling(args.PageSize, args.PageToken)
		if err != nil {
			return err
		}
		info := cli.GetSkuRateInfo(billings)
		fmt.Println(jsonutils.Marshal(info).PrettyString())
		return nil
	})

	type SkuEstimate struct {
		RATE_FAILE string
		SKU        string
		REGION     string
		CPU        int
		MEMORY_MB  int
	}

	shellutils.R(&SkuEstimate{}, "sku-estimate", "Estimate sku price", func(cli *google.SRegion, args *SkuEstimate) error {
		data, err := os.ReadFile(args.RATE_FAILE)
		if err != nil {
			return errors.Wrap(err, "os.ReadFile")
		}
		rate := google.SRateInfo{}
		j, err := jsonutils.Parse(data)
		if err != nil {
			return errors.Wrap(err, "jsonutils.Parse")
		}
		err = jsonutils.Update(&rate, j)
		if err != nil {
			return errors.Wrap(err, "jsonutils.Update")
		}
		result, err := rate.GetSkuPrice(args.REGION, args.SKU, args.CPU, args.MEMORY_MB)
		if err != nil {
			return errors.Wrap(err, "GetSkuPrice")
		}
		fmt.Printf("result: %s\n", jsonutils.Marshal(result).PrettyString())
		return nil
	})

}
