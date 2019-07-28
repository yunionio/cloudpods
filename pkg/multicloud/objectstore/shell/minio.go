package shell

import (
	"fmt"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type MinioBucketOption struct {
		BUCKET string `help:"name of bucket"`
	}
	shellutils.R(&MinioBucketOption{}, "bucket-head", "HEAD bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		cli.IBucketExist(args.BUCKET)
		return nil
	})

	shellutils.R(&MinioBucketOption{}, "bucket-acl", "Get ACL of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		acl, err := cli.GetIBucketAcl(args.BUCKET)
		if err != nil {
			return err
		}
		fmt.Println("ACL:", acl)
		return nil
	})

	type MinioBucketCannedAclConfigOption struct {
		BUCKET string `help:"name of bucket"`
		ACL    string `help:"canned ACL" choices:"private|public-read|public-read-write|auth-read"`
	}
	shellutils.R(&MinioBucketCannedAclConfigOption{}, "bucket-canned-acl-config", "Set canned ACL of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketCannedAclConfigOption) error {
		err := cli.SetIBucketAcl(args.BUCKET, cloudprovider.TBucketACLType(args.ACL))
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&MinioBucketOption{}, "bucket-policy", "Get Policy of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		policy, err := cli.GetIBucketPolicy(args.BUCKET)
		if err != nil {
			return err
		}
		fmt.Println("Policy:", policy)
		return nil
	})

	shellutils.R(&MinioBucketOption{}, "bucket-lifecycle", "Get lifecycle of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		lifecycle, err := cli.GetIBucketLiftcycle(args.BUCKET)
		if err != nil {
			return err
		}
		fmt.Println("Lifecycle:", lifecycle)
		return nil
	})

	shellutils.R(&MinioBucketOption{}, "bucket-info", "Get info of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		info, err := cli.GetIBucketInfo(args.BUCKET)
		if err != nil {
			return err
		}
		fmt.Println("Info:", info)
		return nil
	})

	shellutils.R(&MinioBucketOption{}, "bucket-location", "Get location of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		info, err := cli.GetIBucketLocation(args.BUCKET)
		if err != nil {
			return err
		}
		fmt.Println("Info:", info)
		return nil
	})

	shellutils.R(&MinioBucketOption{}, "bucket-website", "Get website info of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		info, err := cli.GetIBucketWebsite(args.BUCKET)
		if err != nil {
			return err
		}
		fmt.Println("Info:", info)
		return nil
	})

	shellutils.R(&MinioBucketOption{}, "bucket-logging", "Get logging configuration of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		info, err := cli.GetIBucketLogging(args.BUCKET)
		if err != nil {
			return err
		}
		fmt.Println("Logging:", info)
		return nil
	})

	type MinioSetLoggingOption struct {
		BUCKET string `help:"id or name of bucket"`
		Target string `help:"target bucket"`
		Prefix string `help:"target prefix"`
		Email  string `help:"email"`
	}
	shellutils.R(&MinioSetLoggingOption{}, "bucket-logging-config", "Set logging configuration of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioSetLoggingOption) error {
		err := cli.SetIBucketLogging(args.BUCKET, args.Target, args.Prefix, args.Email)
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&MinioBucketOption{}, "bucket-referer", "Get info of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		info, err := cli.GetIBucketReferer(args.BUCKET)
		if err != nil {
			return err
		}
		fmt.Println("Info:", info)
		return nil
	})

	shellutils.R(&MinioBucketOption{}, "bucket-cors", "Get info of bucket", func(cli *objectstore.SObjectStoreClient, args *MinioBucketOption) error {
		info, err := cli.GetIBucketCors(args.BUCKET)
		if err != nil {
			return err
		}
		fmt.Println("Info:", info)
		return nil
	})
}
