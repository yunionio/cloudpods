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
	"context"
	"os"

	coslib "github.com/nelsonken/cos-go-sdk-v5/cos"

	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type CosListOptions struct {
	}
	shellutils.R(&CosListOptions{}, "cos-list", "List COS buckets", func(cli *qcloud.SRegion, args *CosListOptions) error {
		buckets, err := cli.GetIBuckets()
		if err != nil {
			return err
		}
		printList(buckets, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&CosListOptions{}, "bucket-list", "List COS buckets", func(cli *qcloud.SRegion, args *CosListOptions) error {
		buckets, err := cli.GetIBuckets()
		if err != nil {
			return err
		}
		printutils.PrintGetterList(buckets, nil)
		return nil
	})

	type CosCreateBucketOptions struct {
		BUCKET string `help:"name of bucket to create"`
		Acl    string `help:"Acl"`
	}
	shellutils.R(&CosCreateBucketOptions{}, "cos-create-bucket", "Create a COS bucket", func(cli *qcloud.SRegion, args *CosCreateBucketOptions) error {
		err := cli.CreateIBucket(args.BUCKET, "", args.Acl)
		if err != nil {
			return err
		}
		return nil
	})

	type CosDeleteBucketOptions struct {
		BUCKET string `help:"name of bucket to delete"`
	}
	shellutils.R(&CosDeleteBucketOptions{}, "cos-delete-bucket", "Delete a COS bucket", func(cli *qcloud.SRegion, args *CosDeleteBucketOptions) error {
		err := cli.DeleteIBucket(args.BUCKET)
		if err != nil {
			return err
		}
		return nil
	})

	type CosListBucketOptions struct {
		BUCKET string `help:"bucket name"`
	}

	shellutils.R(&CosListBucketOptions{}, "cos-bucket-list", "List content of a OSS bucket", func(cli *qcloud.SRegion, args *CosListBucketOptions) error {
		cos, err := cli.GetCosClient()
		if err != nil {
			return err
		}
		result, err := cos.ListBucketContents(context.Background(), args.BUCKET, &coslib.QueryCondition{})
		if err != nil {
			return err
		}
		printList(result.Contents, len(result.Contents), 0, len(result.Contents), nil)
		return nil
	})

	shellutils.R(&CosListBucketOptions{}, "cos-bucket-create", "Create a OSS bucket", func(cli *qcloud.SRegion, args *CosListBucketOptions) error {
		cos, err := cli.GetCosClient()
		if err != nil {
			return err
		}
		return cos.CreateBucket(context.Background(), args.BUCKET, &coslib.AccessControl{})
	})

	type CosUploadOptions struct {
		BUCKET string `help:"bucket name"`
		KEY    string `help:"Object key"`
		FILE   string `help:"Local file path"`
		Acl    string `help:"Object ACL" choices:"private|public-read|public-read-write"`
	}
	shellutils.R(&CosUploadOptions{}, "cos-upload", "Upload a file to a Cos bucket", func(cli *qcloud.SRegion, args *CosUploadOptions) error {
		cos, err := cli.GetCosClient()
		if err != nil {
			return err
		}
		return cos.Bucket(args.BUCKET).UploadObjectBySlice(context.Background(), args.KEY, args.FILE, 3, nil)
	})

	type CosDownloadOptions struct {
		BUCKET string `help:"bucket name"`
		NAME   string `help:"File name"`
	}
	shellutils.R(&CosDownloadOptions{}, "cos-download", "Download a file", func(cli *qcloud.SRegion, args *CosDownloadOptions) error {
		cos, err := cli.GetCosClient()
		if err != nil {
			return err
		}
		//file
		return cos.Bucket(args.BUCKET).DownloadObject(context.Background(), args.NAME, os.Stdout)
		//return cos.Bucket(args.BUCKET).UploadObjectBySlice(context.Background(), args.KEY, args.FILE, 3, nil)
	})

	type CosObjectAclOptions struct {
		BUCKET string `help:"bucket name"`
		KEY    string `help:"object key"`
		ACL    string `help:"ACL" choices:"private|public-read|public-read-write"`
	}
	shellutils.R(&CosObjectAclOptions{}, "cos-set-acl", "Set acl for a object", func(cli *qcloud.SRegion, args *CosObjectAclOptions) error {
		cos, err := cli.GetCosClient()
		if err != nil {
			return err
		}
		return cos.SetBucketACL(context.Background(), args.KEY, &coslib.AccessControl{ACL: args.ACL})
	})

	type CosDeleteOptions struct {
		BUCKET string `help:"bucket name"`
		KEY    string `help:"Object key"`
	}

	shellutils.R(&CosDeleteOptions{}, "cos-delete", "Delete a file from a Cos bucket", func(cli *qcloud.SRegion, args *CosDeleteOptions) error {
		cos, err := cli.GetCosClient()
		if err != nil {
			return err
		}
		return cos.Bucket(args.BUCKET).DeleteObject(context.Background(), args.KEY)
	})
}
