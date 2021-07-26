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
	huawei "yunion.io/x/onecloud/pkg/multicloud/huaweistack"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ObsBucketListOptions struct {
	}
	shellutils.R(&ObsBucketListOptions{}, "obs-list", "List all buckets", func(cli *huawei.SRegion, args *ObsBucketListOptions) error {
		buckets, err := cli.GetIBuckets()
		if err != nil {
			return err
		}
		printList(buckets, 0, 0, 0, nil)
		return nil
	})
	shellutils.R(&ObsBucketListOptions{}, "bucket-list", "List all buckets", func(cli *huawei.SRegion, args *ObsBucketListOptions) error {
		buckets, err := cli.GetIBuckets()
		if err != nil {
			return err
		}
		printutils.PrintGetterList(buckets, nil)
		return nil
	})

	type ObsBucketShowOptions struct {
		BUCKET string `help:"bucket name to show"`
	}
	shellutils.R(&ObsBucketShowOptions{}, "obs-show", "Show bucket detail", func(cli *huawei.SRegion, args *ObsBucketShowOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		printObject(bucket)
		return nil
	})

	type ObsBucketCreateOptions struct {
		BUCKET       string `help:"bucket name to show"`
		StorageClass string `help:"storage class"`
		Acl          string `help:"acl"`
	}
	shellutils.R(&ObsBucketCreateOptions{}, "obs-create", "Create new OBS bucket", func(cli *huawei.SRegion, args *ObsBucketCreateOptions) error {
		err := cli.CreateIBucket(args.BUCKET, args.StorageClass, args.Acl)
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&ObsBucketShowOptions{}, "obs-delete", "Delete OBS bucket", func(cli *huawei.SRegion, args *ObsBucketShowOptions) error {
		err := cli.DeleteIBucket(args.BUCKET)
		if err != nil {
			return err
		}
		return nil
	})
}
