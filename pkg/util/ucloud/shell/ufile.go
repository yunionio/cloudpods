package shell

import (
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type UFileBucketListOptions struct {
	}
	shellutils.R(&UFileBucketListOptions{}, "bucket-list", "List buckets", func(cli *ucloud.SRegion, args *UFileBucketListOptions) error {
		buckets, err := cli.GetIBuckets()
		if err != nil {
			return err
		}
		printutils.PrintGetterList(buckets, nil)
		return nil
	})

	type UFileBucketCreateOptions struct {
		BUCKET string `help:"Name of bucket"`
		Acl    string `help:"Acl" choices:"private|public"`
	}
	shellutils.R(&UFileBucketCreateOptions{}, "bucket-create", "create a bucket", func(cli *ucloud.SRegion, args *UFileBucketCreateOptions) error {
		err := cli.CreateIBucket(args.BUCKET, "", args.Acl)
		if err != nil {
			return err
		}
		return nil
	})

	type UFileBucketDeleteOptions struct {
		BUCKET string `help:"Name of bucket"`
	}
	shellutils.R(&UFileBucketDeleteOptions{}, "bucket-delete", "delete a bucket", func(cli *ucloud.SRegion, args *UFileBucketDeleteOptions) error {
		err := cli.DeleteIBucket(args.BUCKET)
		if err != nil {
			return err
		}
		return nil
	})
}
