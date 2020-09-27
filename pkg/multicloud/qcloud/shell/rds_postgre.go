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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type PostgreSQLProductListOptions struct {
	}
	shellutils.R(&PostgreSQLProductListOptions{}, "postgre-product-list", "List postgre sql products", func(cli *qcloud.SRegion, args *PostgreSQLProductListOptions) error {
		products, err := cli.DescribeProductConfig()
		if err != nil {
			return errors.Wrapf(err, "DescribeProductConfig")
		}
		printList(products, 0, 0, 0, nil)
		return nil
	})

	type SPostgreSQLSkuListOptions struct {
	}

	shellutils.R(&SPostgreSQLSkuListOptions{}, "postgre-sku-list", "List postgre sql skus", func(cli *qcloud.SRegion, args *SPostgreSQLSkuListOptions) error {
		skus, err := cli.ListPostgreSQLSkus()
		if err != nil {
			return errors.Wrapf(err, "ListPostgreSQLSkus")
		}
		printList(skus, 0, 0, 0, nil)
		return nil
	})

}
