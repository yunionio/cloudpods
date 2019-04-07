package shell

import (
	"github.com/aws/aws-sdk-go/service/s3"
	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type S3BucketListOptions struct {
	}
	shellutils.R(&S3BucketListOptions{}, "s3-list", "List all buckets", func(cli *aws.SRegion, args *S3BucketListOptions) error {
		s3cli, err := cli.GetS3Client()
		if err != nil {
			return err
		}
		output, err := s3cli.ListBuckets(&s3.ListBucketsInput{})
		if err != nil {
			return err
		}
		printList(output.Buckets, 0, 0, 0, nil)
		return nil
	})
}
