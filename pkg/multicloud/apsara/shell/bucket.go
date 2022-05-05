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
	"strconv"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/multicloud/apsara"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	objectstore.S3Shell()

	type BucketOptions struct {
		BUCKET string
	}
	shellutils.R(&BucketOptions{}, "bucket-show", "Show bucket", func(cli *apsara.SRegion, args *BucketOptions) error {
		bucket, e := cli.GetBucket(args.BUCKET)
		if e != nil {
			return e
		}
		printObject(bucket)
		return nil
	})

	type BucketSizeOptions struct {
	}
	shellutils.R(&BucketSizeOptions{}, "bucket-size", "Show bucket size", func(cli *apsara.SRegion, args *BucketSizeOptions) error {
		buckets, err := cli.GetBuckets()
		if err != nil {
			return err
		}
		for i := range buckets {
			department, _ := strconv.Atoi(buckets[i].Department)
			size, _ := cli.GetBucketSize(buckets[i].Name, department)
			capa, _ := cli.GetBucketCapacity(buckets[i].Name, department)
			log.Infof("bucket %s size: %dMb capa: %dG", buckets[i].Name, size/1024/1024/1024, capa)
		}
		return nil
	})

}
