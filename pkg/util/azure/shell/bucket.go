package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type BucketListOptions struct {
	}
	shellutils.R(&BucketListOptions{}, "bucket-list", "List buckets", func(cli *azure.SRegion, args *BucketListOptions) error {
		buckets, err := cli.GetIBuckets()
		if err != nil {
			return err
		}
		printutils.PrintGetterList(buckets, nil)
		return nil
	})

	type BucketCreateOptions struct {
		BUCKET       string `help:"bucket name"`
		STORAGECLASS string `help:"storage class"`
	}
	shellutils.R(&BucketCreateOptions{}, "bucket-create", "Create bucket", func(cli *azure.SRegion, args *BucketCreateOptions) error {
		err := cli.CreateIBucket(args.BUCKET, args.STORAGECLASS, "")
		if err != nil {
			return err
		}
		return nil
	})

	type BucketDeleteOptions struct {
		BUCKET string `help:"bucket name"`
	}
	shellutils.R(&BucketDeleteOptions{}, "bucket-delete", "Delete a bucket", func(cli *azure.SRegion, args *BucketDeleteOptions) error {
		err := cli.DeleteIBucket(args.BUCKET)
		if err != nil {
			return err
		}
		return nil
	})

	type BucketShowOptions struct {
		BUCKET string `help:"bucket name"`
	}
	shellutils.R(&BucketShowOptions{}, "bucket-show", "Show a bucket", func(cli *azure.SRegion, args *BucketShowOptions) error {
		bucket, err := cli.GetIBucketByName(args.BUCKET)
		if err != nil {
			return err
		}
		printObject(bucket)
		return nil
	})
}
