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
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type BucketListOptions struct {
		MaxResults int
		PageToken  string
	}

	shellutils.R(&BucketListOptions{}, "bucket-list", "List buckets", func(cli *google.SRegion, args *BucketListOptions) error {
		buckets, err := cli.GetBuckets(args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(buckets, 0, 0, 0, nil)
		return nil
	})

	type BucketCreateOptions struct {
		NAME         string
		StorageClass string `choices:"STANDARD|NEARLINE|COLDLINE"`
	}

	shellutils.R(&BucketCreateOptions{}, "bucket-create", "Create buckets", func(cli *google.SRegion, args *BucketCreateOptions) error {
		bucket, err := cli.CreateBucket(args.NAME, args.StorageClass)
		if err != nil {
			return err
		}
		printObject(bucket)
		return nil
	})

	type BucketNameOptions struct {
		NAME string
	}

	shellutils.R(&BucketNameOptions{}, "bucket-show", "Show bucket", func(cli *google.SRegion, args *BucketNameOptions) error {
		bucket, err := cli.GetBucket(args.NAME)
		if err != nil {
			return err
		}
		printObject(bucket)
		return nil
	})

	shellutils.R(&BucketNameOptions{}, "bucket-delete", "Delete bucket", func(cli *google.SRegion, args *BucketNameOptions) error {
		return cli.DeleteBucket(args.NAME)
	})

}
