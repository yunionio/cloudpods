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

package objectstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func S3Shell() {
	type BucketListOptions struct {
	}
	shellutils.R(&BucketListOptions{}, "bucket-list", "List all bucket", func(cli cloudprovider.ICloudRegion, args *BucketListOptions) error {
		buckets, err := cli.GetIBuckets()
		if err != nil {
			return err
		}
		printutils.PrintGetterList(buckets, nil)
		return nil
	})

	type BucketCreateOptions struct {
		NAME         string `help:"name of bucket to create"`
		Acl          string `help:"ACL string" choices:"private|public-read|public-read-write"`
		StorageClass string `help:"StorageClass" choices:"STANDARD|IA|ARCHIVE"`
	}
	shellutils.R(&BucketCreateOptions{}, "bucket-create", "Create bucket", func(cli cloudprovider.ICloudRegion, args *BucketCreateOptions) error {
		err := cli.CreateIBucket(args.NAME, args.StorageClass, args.Acl)
		if err != nil {
			return err
		}
		return nil
	})

	type BucketAclOptions struct {
		BUCKET string `help:"name of bucket"`
		ACL    string `help:"ACL string" choices:"private|public-read|public-read-write"`
	}
	shellutils.R(&BucketAclOptions{}, "bucket-set-acl", "Create bucket", func(cli cloudprovider.ICloudRegion, args *BucketAclOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		err = bucket.SetAcl(cloudprovider.TBucketACLType(args.ACL))
		if err != nil {
			return err
		}
		return nil
	})

	type BucketDeleteOptions struct {
		NAME string `help:"name of bucket to delete"`
	}
	shellutils.R(&BucketDeleteOptions{}, "bucket-delete", "Delete bucket", func(cli cloudprovider.ICloudRegion, args *BucketDeleteOptions) error {
		err := cli.DeleteIBucket(args.NAME)
		if err != nil {
			return err
		}
		return nil
	})

	type BucketExistOptions struct {
		NAME string `help:"name of bucket to delete"`
	}
	shellutils.R(&BucketExistOptions{}, "bucket-exist", "Test existence of a bucket", func(cli cloudprovider.ICloudRegion, args *BucketExistOptions) error {
		exist, err := cli.IBucketExist(args.NAME)
		if err != nil {
			return err
		}
		fmt.Printf("Exist: %v\n", exist)
		return nil
	})

	type BucketObjectsOptions struct {
		BUCKET    string `help:"name of bucket to list objects"`
		Prefix    string `help:"prefix"`
		Marker    string `help:"marker"`
		Demiliter string `help:"delimiter"`
		Max       int    `help:"Max count"`
	}
	shellutils.R(&BucketObjectsOptions{}, "bucket-object", "List objects in a bucket", func(cli cloudprovider.ICloudRegion, args *BucketObjectsOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		result, err := bucket.ListObjects(args.Prefix, args.Marker, args.Demiliter, args.Max)
		if err != nil {
			return err
		}
		if result.IsTruncated {
			fmt.Printf("NextMarker: %s IsTruncated: %v\n", result.NextMarker, result.IsTruncated)
		}
		fmt.Println("Common prefixes:")
		printutils.PrintGetterList(result.CommonPrefixes, []string{"key", "size_bytes"})
		fmt.Println("Objects:")
		printutils.PrintGetterList(result.Objects, []string{"key", "size_bytes"})
		return nil
	})

	type BucketListObjectsOptions struct {
		BUCKET string `help:"name of bucket to list objects"`
		Prefix string `help:"prefix"`
	}
	shellutils.R(&BucketListObjectsOptions{}, "bucket-list-object", "List objects in a bucket", func(cli cloudprovider.ICloudRegion, args *BucketListObjectsOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		objects, err := bucket.GetIObjects(args.Prefix, true)
		if err != nil {
			return err
		}
		printutils.PrintGetterList(objects, []string{"key", "size_bytes"})
		return nil
	})

	shellutils.R(&BucketListObjectsOptions{}, "bucket-dir-object", "List objects in a bucket like directory", func(cli cloudprovider.ICloudRegion, args *BucketListObjectsOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		objects, err := bucket.GetIObjects(args.Prefix, false)
		if err != nil {
			return err
		}
		printutils.PrintGetterList(objects, []string{"key", "size_bytes"})
		return nil
	})

	type BucketMakrdirOptions struct {
		BUCKET string `help:"name of bucket to put object"`
		DIR    string `help:"dir to make"`
	}
	shellutils.R(&BucketMakrdirOptions{}, "bucket-mkdir", "Mkdir in a bucket", func(cli cloudprovider.ICloudRegion, args *BucketMakrdirOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		err = cloudprovider.Makedir(context.Background(), bucket, args.DIR)
		if err != nil {
			return err
		}
		fmt.Printf("Mkdir success\n")
		return nil
	})

	type BucketPutObjectOptions struct {
		BUCKET string `help:"name of bucket to put object"`
		KEY    string `help:"key of object"`
		Path   string `help:"Path of file to upload"`

		BlockSize int64 `help:"blocksz in MB" default:"100"`

		Acl          string `help:"acl" choices:"private|public-read|public-read-write"`
		ContentType  string `help:"content-type"`
		StorageClass string `help:"storage class"`
	}
	shellutils.R(&BucketPutObjectOptions{}, "put-object", "Put object into a bucket", func(cli cloudprovider.ICloudRegion, args *BucketPutObjectOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		var input io.ReadSeeker
		var fSize int64
		if len(args.Path) > 0 {
			finfo, err := os.Stat(args.Path)
			if err != nil {
				return errors.Wrap(err, "os.Stat")
			}
			fSize = finfo.Size()
			file, err := os.Open(args.Path)
			if err != nil {
				return errors.Wrap(err, "os.Open")
			}
			defer file.Close()

			input = file
		} else {
			input = os.Stdout
		}
		err = cloudprovider.UploadObject(context.Background(), bucket, args.KEY, args.BlockSize*1000*1000, input, fSize, args.ContentType, cloudprovider.TBucketACLType(args.Acl), args.StorageClass, true)
		if err != nil {
			return err
		}
		fmt.Printf("Upload success\n")
		return nil
	})

	type BucketDeleteObjectOptions struct {
		BUCKET string `help:"name of bucket to put object"`
		KEY    string `help:"key of object"`
	}
	shellutils.R(&BucketDeleteObjectOptions{}, "delete-object", "Delete object from a bucket", func(cli cloudprovider.ICloudRegion, args *BucketDeleteObjectOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		err = bucket.DeleteObject(context.Background(), args.KEY)
		if err != nil {
			return err
		}
		fmt.Printf("Delete success\n")
		return nil
	})

	type BucketTempUrlOption struct {
		BUCKET   string `help:"name of bucket to put object"`
		METHOD   string `help:"http method" choices:"GET|PUT|DELETE"`
		KEY      string `help:"key of object"`
		Duration int    `help:"duration in seconds" default:"60"`
	}
	shellutils.R(&BucketTempUrlOption{}, "temp-url", "generate temp url", func(cli cloudprovider.ICloudRegion, args *BucketTempUrlOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		urlStr, err := bucket.GetTempUrl(args.METHOD, args.KEY, time.Duration(args.Duration)*time.Second)
		if err != nil {
			return err
		}
		fmt.Println(urlStr)
		return nil
	})

	type BucketAclOption struct {
		BUCKET string `help:"name of bucket to put object"`
		KEY    string `help:"key of object"`
	}
	shellutils.R(&BucketAclOption{}, "object-acl", "Get object acl", func(cli cloudprovider.ICloudRegion, args *BucketAclOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		object, err := cloudprovider.GetIObject(bucket, args.KEY)
		if err != nil {
			return err
		}
		fmt.Println(object.GetAcl())
		return nil
	})

	type BucketSetAclOption struct {
		BUCKET string `help:"name of bucket to put object"`
		KEY    string `help:"key of object"`
		ACL    string `help:"Target acl" choices:"default|private|public-read|public-read-write"`
	}
	shellutils.R(&BucketSetAclOption{}, "object-set-acl", "Get object acl", func(cli cloudprovider.ICloudRegion, args *BucketSetAclOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		object, err := cloudprovider.GetIObject(bucket, args.KEY)
		if err != nil {
			return err
		}
		err = object.SetAcl(cloudprovider.TBucketACLType(args.ACL))
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})
}
