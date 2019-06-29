package shell

import (
	"fmt"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type BucketListOptions struct {
	}
	shellutils.R(&BucketListOptions{}, "bucket-list", "List all bucket", func(cli *objectstore.SObjectStoreClient, args *BucketListOptions) error {
		buckets, err := cli.GetIBuckets()
		if err != nil {
			return err
		}
		printList(buckets, nil)
		return nil
	})

	type BucketCreateOptions struct {
		NAME string `help:"name of bucket to create"`
	}
	shellutils.R(&BucketCreateOptions{}, "bucket-create", "Create bucket", func(cli *objectstore.SObjectStoreClient, args *BucketCreateOptions) error {
		err := cli.CreateIBucket(args.NAME, "", "")
		if err != nil {
			return err
		}
		return nil
	})

	type BucketDeleteOptions struct {
		NAME string `help:"name of bucket to delete"`
	}
	shellutils.R(&BucketDeleteOptions{}, "bucket-delete", "Delete bucket", func(cli *objectstore.SObjectStoreClient, args *BucketDeleteOptions) error {
		err := cli.DeleteIBucket(args.NAME)
		if err != nil {
			return err
		}
		return nil
	})

	type BucketPolicyOptions struct {
		NAME string `help:"name of bucket to get policy"`
	}
	shellutils.R(&BucketPolicyOptions{}, "bucket-policy", "Get bucket policy", func(cli *objectstore.SObjectStoreClient, args *BucketPolicyOptions) error {
		policy, err := cli.GetIBucketPolicy(args.NAME)
		if err != nil {
			return err
		}
		fmt.Println(policy)
		return nil
	})

	shellutils.R(&BucketPolicyOptions{}, "bucket-lifecycle", "Get bucket lifecycle", func(cli *objectstore.SObjectStoreClient, args *BucketPolicyOptions) error {
		lifecycle, err := cli.GetIBucketLiftcycle(args.NAME)
		if err != nil {
			return err
		}
		fmt.Println(lifecycle)
		return nil
	})
}
