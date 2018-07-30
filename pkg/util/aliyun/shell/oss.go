package shell

import (
	"yunion.io/yunioncloud/pkg/util/aliyun"
)

func init() {
	type OssListOptions struct {
	}
	R(&OssListOptions{}, "oss-list", "List OSS buckets", func(cli *aliyun.SRegion, args *OssListOptions) error {
		oss, err := cli.GetOssClient()
		if err != nil {
			return err
		}
		result, err := oss.ListBuckets()
		if err != nil {
			return err
		}
		printList(result.Buckets, len(result.Buckets), 0, 50, nil)
		return nil
	})

	type OssListBucketOptions struct {
		BUCKET string `help:"bucket name"`
	}

	R(&OssListBucketOptions{}, "oss-list-bucket", "List content of a OSS bucket", func(cli *aliyun.SRegion, args *OssListBucketOptions) error {
		oss, err := cli.GetOssClient()
		if err != nil {
			return err
		}
		bucket, err := oss.Bucket(args.BUCKET)
		if err != nil {
			return err
		}
		result, err := bucket.ListObjects()
		if err != nil {
			return err
		}
		printList(result.Objects, len(result.Objects), 0, len(result.Objects), nil)
		return nil
	})

	R(&OssListBucketOptions{}, "oss-create-bucket", "Create a OSS bucket", func(cli *aliyun.SRegion, args *OssListBucketOptions) error {
		oss, err := cli.GetOssClient()
		if err != nil {
			return err
		}
		err = oss.CreateBucket(args.BUCKET)
		if err != nil {
			return err
		}
		return nil
	})

	type OssUploadOptions struct {
		BUCKET string `help:"bucket name"`
		KEY    string `help:"Object key"`
		FILE   string `help:"Local file path"`
	}
	R(&OssUploadOptions{}, "oss-upload", "Upload a file to a OSS bucket", func(cli *aliyun.SRegion, args *OssUploadOptions) error {
		oss, err := cli.GetOssClient()
		if err != nil {
			return err
		}
		bucket, err := oss.Bucket(args.BUCKET)
		if err != nil {
			return err
		}
		err = bucket.UploadFile(args.KEY, args.FILE, 1024*1024)
		return err
	})

	type OssDeleteOptions struct {
		BUCKET string `help:"bucket name"`
		KEY    string `help:"Object key"`
	}

	R(&OssDeleteOptions{}, "oss-delete", "Delete a file from a OSS bucket", func(cli *aliyun.SRegion, args *OssDeleteOptions) error {
		oss, err := cli.GetOssClient()
		if err != nil {
			return err
		}
		bucket, err := oss.Bucket(args.BUCKET)
		if err != nil {
			return err
		}
		err = bucket.DeleteObject(args.KEY)
		return err
	})
}
