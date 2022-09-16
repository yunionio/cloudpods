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
	"io"
	"net/http"
	"os"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ObjectPutOptions struct {
		BUCKET      string
		FILE        string
		ContentType string
		StorageType string `choices:"STANDARD|NEARLINE|COLDLINE|ARCHIVE"`
		Acl         string `choices:"private|public-read|public-read-write|authenticated-read"`
	}

	shellutils.R(&ObjectPutOptions{}, "object-upload", "Upload object to buckets", func(cli *google.SRegion, args *ObjectPutOptions) error {
		file, err := os.Open(args.FILE)
		if err != nil {
			return errors.Wrap(err, "so.Open")
		}
		stat, err := file.Stat()
		if err != nil {
			return errors.Wrap(err, "file.Stat")
		}
		header := http.Header{}
		if len(args.ContentType) > 0 {
			header.Set("Content-Type", args.ContentType)
		}
		if stat.Size() <= cloudprovider.MAX_PUT_OBJECT_SIZEBYTES {
			return cli.PutObject(args.BUCKET, args.FILE, file, stat.Size(), cloudprovider.TBucketACLType(args.Acl), header)
		}
		uploadId, err := cli.NewMultipartUpload(args.BUCKET, args.FILE, cloudprovider.TBucketACLType(args.Acl), args.StorageType, http.Header{})
		if err != nil {
			return errors.Wrap(err, "NewMultipartUpload")
		}
		fmt.Println("uploadId: ", uploadId)
		offset := int64(0)
		partSize := cloudprovider.MAX_PUT_OBJECT_SIZEBYTES
		count := stat.Size() / int64(partSize)
		if stat.Size()%int64(partSize) > 0 {
			count += 1
		}
		for i := 0; i < int(count); i++ {
			err := cli.UploadPart(args.BUCKET, uploadId, 0, int64(offset), io.LimitReader(file, int64(partSize)), int64(partSize), stat.Size())
			if err != nil {
				return errors.Wrap(err, "UploadPart")
			}
			offset += partSize
			if int64(offset+partSize) > stat.Size() {
				partSize = stat.Size() % int64(partSize)
			}
		}
		return nil
	})

	type ObjectListOptions struct {
		BUCKET        string
		Prefix        string
		Delimiter     string
		NextPageToken string
		MaxResult     int
	}

	shellutils.R(&ObjectListOptions{}, "object-list", "List object from bucket", func(cli *google.SRegion, args *ObjectListOptions) error {
		objs, err := cli.GetObjects(args.BUCKET, args.Prefix, args.NextPageToken, args.Delimiter, args.MaxResult)
		if err != nil {
			return err
		}
		printList(objs.Items, 0, 0, 0, nil)
		return nil
	})

	type ObjectUrlOptions struct {
		BUCKET string
		OBJECT string
		METHOD string
		Hour   int
	}

	shellutils.R(&ObjectUrlOptions{}, "object-url", "Object temp url", func(cli *google.SRegion, args *ObjectUrlOptions) error {
		url, err := cli.SingedUrl(args.BUCKET, args.OBJECT, args.METHOD, time.Duration(args.Hour)*time.Hour)
		if err != nil {
			return err
		}
		fmt.Println(url)
		return nil
	})

	type ObjectOptions struct {
		BUCKET string
		OBJECT string
	}

	shellutils.R(&ObjectOptions{}, "object-acl-list", "List Object acl", func(cli *google.SRegion, args *ObjectOptions) error {
		acls, err := cli.GetObjectAcl(args.BUCKET, args.OBJECT)
		if err != nil {
			return err
		}
		printList(acls, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&ObjectOptions{}, "object-delete", "Delete Object", func(cli *google.SRegion, args *ObjectOptions) error {
		return cli.DeleteObject(args.BUCKET, args.OBJECT)
	})

	type ObjectDownloadOptions struct {
		BUCKET string
		OBJECT string
		Start  int64
		End    int64
	}

	shellutils.R(&ObjectDownloadOptions{}, "object-download", "Download Object", func(cli *google.SRegion, args *ObjectDownloadOptions) error {
		data, err := cli.DownloadObjectRange(args.BUCKET, args.OBJECT, args.Start, args.End)
		if err != nil {
			return errors.Wrap(err, "DownloadObjectRange")
		}
		content, err := io.ReadAll(data)
		if err != nil {
			return errors.Wrap(err, "io.ReadAll")
		}
		return os.WriteFile(args.OBJECT, content, 0644)
	})

	type ObjectUploadCheckOptions struct {
		BUCKET   string
		UPLOADID string
	}

	shellutils.R(&ObjectUploadCheckOptions{}, "object-upload-check", "Upload object to buckets", func(cli *google.SRegion, args *ObjectUploadCheckOptions) error {
		return cli.CheckUploadRange(args.BUCKET, args.UPLOADID)
	})

}
