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

	huawei "yunion.io/x/onecloud/pkg/multicloud/huaweistack"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	objectstore.S3Shell()

	type BucketNameOptions struct {
		NAME string
	}
	shellutils.R(&BucketNameOptions{}, "bucket-head", "Head bucket", func(cli *huawei.SRegion, args *BucketNameOptions) error {
		result, err := cli.HeadBucket(args.NAME)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	shellutils.R(&BucketNameOptions{}, "bucket-project", "Show bucket project", func(cli *huawei.SRegion, args *BucketNameOptions) error {
		bucket, err := cli.GetIBucketByName(args.NAME)
		if err != nil {
			return err
		}
		fmt.Println("project: ", bucket.GetProjectId())
		return nil
	})

}
