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
	"github.com/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type MariadbRegionListOptions struct {
	}
	shellutils.R(&MariadbRegionListOptions{}, "mariadb-region-list", "List mariadb region list", func(cli *qcloud.SRegion, args *MariadbRegionListOptions) error {
		regions, err := cli.DescribeSaleInfo()
		if err != nil {
			return errors.Wrapf(err, "DescribeSaleInfo")
		}
		printList(regions, 0, 0, 0, nil)
		return nil
	})

	type MariadbSpecListOptions struct {
	}

	shellutils.R(&MariadbSpecListOptions{}, "mariadb-spec-list", "List mariadb specs", func(cli *qcloud.SRegion, args *MariadbSpecListOptions) error {
		specs, err := cli.DescribeDBInstanceSpecs()
		if err != nil {
			return errors.Wrapf(err, "DescribeDBInstanceSpecs")
		}
		printList(specs, 0, 0, 0, nil)
		return nil
	})

	type MariadbSkuListOptions struct {
	}

	shellutils.R(&MariadbSkuListOptions{}, "mariadb-sku-list", "List mariadb skus", func(cli *qcloud.SRegion, args *MariadbSkuListOptions) error {
		skus, err := cli.ListMariadbSkus()
		if err != nil {
			return errors.Wrapf(err, "ListMariadbSkus")
		}
		printList(skus, 0, 0, 0, nil)
		return nil
	})

}
