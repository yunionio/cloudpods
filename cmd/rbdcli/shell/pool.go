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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/rbdutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type PoolListOptions struct {
	}
	shellutils.R(&PoolListOptions{}, "pool-list", "List pools", func(cli *rbdutils.SPool, args *PoolListOptions) error {
		cluster := cli.GetCluster()
		pools, err := cluster.ListPools()
		if err != nil {
			return errors.Wrapf(err, "ListPools")
		}
		printObject(jsonutils.Marshal(map[string][]string{"pools": pools}))
		return nil
	})

	type PoolDeleteOptions struct {
		POOL string
	}

	shellutils.R(&PoolDeleteOptions{}, "pool-delete", "Delete Cluster pool", func(cli *rbdutils.SPool, args *PoolDeleteOptions) error {
		return cli.GetCluster().DeletePool(args.POOL)
	})

	type ImageListOptions struct {
	}

	shellutils.R(&ImageListOptions{}, "image-list", "List Pool images", func(cli *rbdutils.SPool, args *ImageListOptions) error {
		images, err := cli.ListImages()
		if err != nil {
			return err
		}
		printObject(jsonutils.Marshal(map[string][]string{"images": images}))
		return nil
	})

}
