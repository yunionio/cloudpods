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

package compute

import (
	"fmt"
	"io"
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	computeoptions "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Buckets)
	cmd.List(&computeoptions.BucketListOptions{})
	cmd.GetProperty(&computeoptions.BucketGetPropertyOptions{})
	cmd.Show(&computeoptions.BucketIdOptions{})
	cmd.Perform("syncstatus", &computeoptions.BucketIdOptions{})
	cmd.Update(&computeoptions.BucketUpdateOptions{})
	cmd.Delete(&computeoptions.BucketIdOptions{})
	cmd.Create(&computeoptions.BucketCreateOptions{})
	cmd.GetWithCustomOptionShow("objects", func(data jsonutils.JSONObject, args shell.IGetOpt) {
		listResult := printutils.ListResult{}
		err := data.Unmarshal(&listResult)
		if err != nil {
			return
		}
		printList(&listResult, []string{})
	}, &computeoptions.BucketListObjectsOptions{})
	cmd.Perform("delete", &computeoptions.BucketDeleteObjectsOptions{})
	cmd.Perform("makedir", &computeoptions.BucketMakeDirOptions{})
	cmd.Perform("temp-url", &computeoptions.BucketPresignObjectsOptions{})
	cmd.Perform("acl", &computeoptions.BucketSetAclOptions{})
	cmd.GetWithCustomOptionShow("acl", func(data jsonutils.JSONObject, args shell.IGetOpt) {
		printObject(data)
	}, &computeoptions.BucketAclOptions{})
	cmd.Perform("sync", &computeoptions.BucketSyncOptions{})
	cmd.Perform("limit", &computeoptions.BucketLimitOptions{})
	cmd.GetWithCustomOptionShow("access-info", func(data jsonutils.JSONObject, args shell.IGetOpt) {
		printObject(data)
	}, &computeoptions.BucketAccessInfoOptions{})
	cmd.Perform("metadata", &computeoptions.BucketSetMetadataOptions{})
	cmd.Perform("set-website", &computeoptions.BucketSetWebsiteOption{})
	cmd.GetWithCustomOptionShow("website", func(data jsonutils.JSONObject, args shell.IGetOpt) {
		printObject(data)
	}, &computeoptions.BucketGetWebsiteConfOption{})
	cmd.Perform("delete-website", &computeoptions.BucketDeleteWebsiteConfOption{})
	cmd.Perform("set-cors", &computeoptions.BucketSetCorsOption{})
	cmd.GetWithCustomOptionShow("cors", func(data jsonutils.JSONObject, args shell.IGetOpt) {
		printObject(data)
	}, &computeoptions.BucketGetCorsOption{})
	cmd.Perform("delete-cors", &computeoptions.BucketDeleteCorsOption{})
	cmd.Perform("set-referer", &computeoptions.BucketSetRefererOption{})
	cmd.GetWithCustomOptionShow("referer", func(data jsonutils.JSONObject, args shell.IGetOpt) {
		printObject(data)
	}, &computeoptions.BucketGetRefererOption{})
	cmd.GetWithCustomOptionShow("cdn-domain", func(data jsonutils.JSONObject, args shell.IGetOpt) {
		printObject(data)
	}, &computeoptions.BucketGetCdnDomainOption{})
	cmd.GetWithCustomOptionShow("policy", func(data jsonutils.JSONObject, args shell.IGetOpt) {
		printObject(data)
	}, &computeoptions.BucketGetPolicyOption{})
	cmd.Perform("set-policy", &computeoptions.BucketSetPolicyOption{})
	cmd.Perform("delete-policy", &computeoptions.BucketDeletePolicyOption{})

	R(&computeoptions.BucketUploadObjectsOptions{}, "bucket-object-upload", "Upload an object into a bucket", func(s *mcclient.ClientSession, args *computeoptions.BucketUploadObjectsOptions) error {
		var body io.Reader
		if len(args.Path) > 0 {
			file, err := os.Open(args.Path)
			if err != nil {
				return err
			}
			defer file.Close()
			body = file

			fileInfo, err := file.Stat()
			if err != nil {
				return err
			}

			args.ContentLength = fileInfo.Size()
		} else {
			body = os.Stdin
		}

		if args.ContentLength < 0 {
			return fmt.Errorf("required content-length")
		}

		meta := args.ObjectHeaderOptions.Options2Header()

		err := modules.Buckets.Upload(s, args.ID, args.KEY, body, args.ContentLength, args.StorageClass, args.Acl, meta)
		if err != nil {
			return err
		}
		return nil
	})

	R(&computeoptions.BucketPerfMonOptions{}, "bucket-perf-mon", "Bucket performance monitor", func(s *mcclient.ClientSession, args *computeoptions.BucketPerfMonOptions) error {
		result, err := modules.Buckets.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		bucketDetails := compute.BucketDetails{}
		err = result.Unmarshal(&bucketDetails)
		if err != nil {
			return err
		}

		bucket, err := modules.GetIBucket(s.GetContext(), s, &bucketDetails)
		if err != nil {
			return err
		}

		payload, err := fileutils.GetSizeBytes(args.Payload, 1024)
		if err != nil {
			return err
		}

		stats, err := modules.ProbeBucketStats(s.GetContext(), bucket, "test", int64(payload))
		if err != nil {
			return err
		}

		fmt.Printf("Upload delay %f ms throughput %f MB/s\n", stats.UploadDelayMs(), stats.UploadThroughputMbps(payload/1024/1024))
		fmt.Printf("Download delay %f ms throughput %f MB/s\n", stats.DownloadDelayMs(), stats.DownloadThroughputMbps(payload/1024/1024))
		fmt.Printf("Delete delay %f ms\n", stats.DeleteDelayMs())

		return nil
	})
}
