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
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/service/s3"

	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/streamutils"
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

	type S3ListObjectOptions struct {
		BUCKET string
		Prefix string
		Marker string
		Limit  int64 `help:"list limit" default:"20"`
	}
	shellutils.R(&S3ListObjectOptions{}, "s3-list-objects", "List objects in a bucket", func(cli *aws.SRegion, args *S3ListObjectOptions) error {
		s3cli, err := cli.GetS3Client()
		if err != nil {
			return err
		}
		input := &s3.ListObjectsInput{}
		input = input.SetBucket(args.BUCKET)
		if args.Limit > 0 {
			input = input.SetMaxKeys(args.Limit)
		}
		if len(args.Prefix) > 0 {
			input = input.SetPrefix(args.Prefix)
		}
		if len(args.Marker) > 0 {
			input = input.SetMarker(args.Marker)
		}

		output, err := s3cli.ListObjects(input)
		if err != nil {
			return err
		}
		printList(output.Contents, 0, 0, 0, nil)
		if output.IsTruncated != nil && *output.IsTruncated {
			fmt.Println("More ...")
		}
		return nil
	})

	type S3DownloadObjectOptions struct {
		BUCKET string `help:"Bucket name"`
		OBJECT string `help:"Object key"`
		Output string `help:"Location of output file"`
	}
	shellutils.R(&S3DownloadObjectOptions{}, "s3-download", "Download an object in a bucket", func(cli *aws.SRegion, args *S3DownloadObjectOptions) error {
		s3cli, err := cli.GetS3Client()
		if err != nil {
			return err
		}
		input := &s3.GetObjectInput{}
		input = input.SetBucket(args.BUCKET).SetKey(args.OBJECT)
		output, err := s3cli.GetObject(input)
		if err != nil {
			return err
		}
		var fio *os.File
		if len(args.Output) > 0 {
			fio, err = os.Create(args.Output)
			if err != nil {
				return err
			}
			defer fio.Close()
		} else {
			fio = os.Stdout
		}
		prop, err := streamutils.StreamPipe(output.Body, fio, true)
		if err != nil {
			return err
		}
		if len(args.Output) > 0 {
			fmt.Printf("File: %s Size: %d Chksum: %s\n", args.Output, prop.Size, prop.CheckSum)
		}
		return nil
	})
}
