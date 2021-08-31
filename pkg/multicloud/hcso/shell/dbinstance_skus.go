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
	huawei "yunion.io/x/onecloud/pkg/multicloud/hcso"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceDatastoreOptions struct {
		ENGINE string `help:"DBInstance Engine" choices:"MySQL|PostgreSQL|SQLServer"`
	}

	shellutils.R(&DBInstanceDatastoreOptions{}, "dbinstance-datastore-list", "List dbinstance datastores", func(cli *huawei.SRegion, args *DBInstanceDatastoreOptions) error {
		stores, err := cli.GetDBInstanceDatastores(args.ENGINE)
		if err != nil {
			return err
		}
		printList(stores, 0, 0, 0, nil)
		return nil
	})

	type DBInstanceSkuListOptions struct {
	}

	shellutils.R(&DBInstanceSkuListOptions{}, "dbinstance-sku-list", "List dbinstance datastores", func(cli *huawei.SRegion, args *DBInstanceSkuListOptions) error {
		skus, err := cli.GetDBInstanceSkus()
		if err != nil {
			return err
		}
		printList(skus, 0, 0, 0, nil)
		return nil
	})

}
