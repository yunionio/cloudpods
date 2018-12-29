package shell

import (
	"fmt"

	osslib "github.com/aliyun/aliyun-oss-go-sdk/oss"

	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type progressListener struct {
}

func (this *progressListener) ProgressChanged(event *osslib.ProgressEvent) {
	switch event.EventType {
	case osslib.TransferStartedEvent:
		fmt.Printf("\n")
	case osslib.TransferDataEvent:
		fmt.Printf("Progess: %f%%\r", (float64(event.ConsumedBytes) * 100.0 / float64(event.TotalBytes)))
	case osslib.TransferCompletedEvent:
		fmt.Printf("Transfer complete!\n")
	case osslib.TransferFailedEvent:
		fmt.Printf("Transfer failed!\n")
	default:
		fmt.Printf("Unknonw event type %d\n", event.EventType)
	}
}

func str2AclType(aclStr string) osslib.ACLType {
	switch aclStr {
	case "public-rw":
		return osslib.ACLPublicReadWrite
	case "public-read":
		return osslib.ACLPublicRead
	default:
		return osslib.ACLPrivate
	}
}

func init() {
	type OssListOptions struct {
	}
	shellutils.R(&OssListOptions{}, "oss-list", "List OSS buckets", func(cli *aliyun.SRegion, args *OssListOptions) error {
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

	shellutils.R(&OssListBucketOptions{}, "oss-list-bucket", "List content of a OSS bucket", func(cli *aliyun.SRegion, args *OssListBucketOptions) error {
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

	shellutils.R(&OssListBucketOptions{}, "oss-create-bucket", "Create a OSS bucket", func(cli *aliyun.SRegion, args *OssListBucketOptions) error {
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
		BUCKET   string `help:"bucket name"`
		KEY      string `help:"Object key"`
		FILE     string `help:"Local file path"`
		Progress bool   `help:"show progress"`
		Acl      string `help:"Object ACL" choices:"private|public-read|public-rw"`
	}
	shellutils.R(&OssUploadOptions{}, "oss-upload", "Upload a file to a OSS bucket", func(cli *aliyun.SRegion, args *OssUploadOptions) error {
		oss, err := cli.GetOssClient()
		if err != nil {
			return err
		}
		bucket, err := oss.Bucket(args.BUCKET)
		if err != nil {
			return err
		}

		options := make([]osslib.Option, 0)
		if args.Progress {
			listener := progressListener{}
			options = append(options, osslib.Progress(&listener))
		}
		if len(args.Acl) > 0 {
			options = append(options, osslib.ACL(str2AclType(args.Acl)))
		}
		err = bucket.UploadFile(args.KEY, args.FILE, 4*1024*1024, options...)
		return err
	})

	type OssObjectAclOptions struct {
		BUCKET string `help:"bucket name"`
		KEY    string `help:"object key"`
		ACL    string `help:"ACL" choices:"private|public-read|public-rw"`
	}
	shellutils.R(&OssObjectAclOptions{}, "oss-set-acl", "Set acl for a object", func(cli *aliyun.SRegion, args *OssObjectAclOptions) error {
		oss, err := cli.GetOssClient()
		if err != nil {
			return err
		}
		bucket, err := oss.Bucket(args.BUCKET)
		if err != nil {
			return err
		}
		err = bucket.SetObjectACL(args.KEY, str2AclType(args.ACL))
		return err
	})

	type OssDeleteOptions struct {
		BUCKET string `help:"bucket name"`
		KEY    string `help:"Object key"`
	}

	shellutils.R(&OssDeleteOptions{}, "oss-delete", "Delete a file from a OSS bucket", func(cli *aliyun.SRegion, args *OssDeleteOptions) error {
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
