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

package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/streamutils"
)

type ObjectHeaderOptions struct {
	CacheControl       string `help:"Cache-Control"`
	ContentType        string `help:"Content-Type"`
	ContentEncoding    string `help:"Content-Encoding"`
	ContentLanguage    string `help:"Content-Language"`
	ContentDisposition string `help:"Content-Disposition"`
	ContentMD5         string `help:"Content-MD5"`

	Meta []string `help:"header, common seperatored key and value, e.g. max-age:100"`
}

func (args ObjectHeaderOptions) Options2Header() http.Header {
	meta := http.Header{}
	for _, kv := range args.Meta {
		parts := strings.Split(kv, ":")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if len(key) > 0 && len(value) > 0 {
				meta.Add(key, value)
			}
		}
	}
	if len(args.CacheControl) > 0 {
		meta.Set(cloudprovider.META_HEADER_CACHE_CONTROL, args.CacheControl)
	}
	if len(args.ContentType) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_TYPE, args.ContentType)
	}
	if len(args.ContentEncoding) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_ENCODING, args.ContentEncoding)
	}
	if len(args.ContentMD5) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_MD5, args.ContentMD5)
	}
	if len(args.ContentLanguage) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_LANGUAGE, args.ContentLanguage)
	}
	if len(args.ContentDisposition) > 0 {
		meta.Set(cloudprovider.META_HEADER_CONTENT_DISPOSITION, args.ContentDisposition)
	}
	return meta
}

func printList(data interface{}, total, offset, limit int, columns []string) {
	printutils.PrintInterfaceList(data, total, offset, limit, columns)
}

func printObject(obj interface{}) {
	printutils.PrintInterfaceObject(obj)
}

func S3Shell() {
	type BucketListOptions struct {
	}
	shellutils.R(&BucketListOptions{}, "bucket-list", "List all bucket", func(cli cloudprovider.ICloudRegion, args *BucketListOptions) error {
		buckets, err := cli.GetIBuckets()
		if err != nil {
			return err
		}
		printutils.PrintGetterList(buckets, nil)
		return nil
	})

	type BucketCreateOptions struct {
		NAME         string `help:"name of bucket to create"`
		Acl          string `help:"ACL string" choices:"private|public-read|public-read-write"`
		StorageClass string `help:"StorageClass"`
	}
	shellutils.R(&BucketCreateOptions{}, "bucket-create", "Create bucket", func(cli cloudprovider.ICloudRegion, args *BucketCreateOptions) error {
		err := cli.CreateIBucket(args.NAME, args.StorageClass, args.Acl)
		if err != nil {
			return err
		}
		return nil
	})

	type BucketAclOptions struct {
		BUCKET string `help:"name of bucket"`
		ACL    string `help:"ACL string" choices:"private|public-read|public-read-write"`
	}
	shellutils.R(&BucketAclOptions{}, "bucket-set-acl", "Create bucket", func(cli cloudprovider.ICloudRegion, args *BucketAclOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		err = bucket.SetAcl(cloudprovider.TBucketACLType(args.ACL))
		if err != nil {
			return err
		}
		return nil
	})

	type BucketDeleteOptions struct {
		NAME string `help:"name of bucket to delete"`
	}
	shellutils.R(&BucketDeleteOptions{}, "bucket-delete", "Delete bucket", func(cli cloudprovider.ICloudRegion, args *BucketDeleteOptions) error {
		err := cli.DeleteIBucket(args.NAME)
		if err != nil {
			return err
		}
		return nil
	})

	type BucketLimitOptions struct {
		NAME    string `help:"name of bucket to set limit"`
		SizeGB  int    `help:"limit of volumes in GB"`
		Objects int    `help:"limit of object count"`
		Off     bool   `help:"Turn off limit"`
	}
	shellutils.R(&BucketLimitOptions{}, "bucket-set-limit", "Set bucket limit", func(cli cloudprovider.ICloudRegion, args *BucketLimitOptions) error {
		bucket, err := cli.GetIBucketByName(args.NAME)
		if err != nil {
			return err
		}
		if args.Off {
			err = bucket.SetLimit(cloudprovider.SBucketStats{})
		} else {
			fmt.Println("set limit")
			err = bucket.SetLimit(cloudprovider.SBucketStats{SizeBytes: int64(args.SizeGB * 1000 * 1000 * 1000), ObjectCount: args.Objects})
		}
		if err != nil {
			return err
		}
		return nil
	})

	type BucketExistOptions struct {
		NAME string `help:"name of bucket to delete"`
	}
	shellutils.R(&BucketExistOptions{}, "bucket-exist", "Test existence of a bucket", func(cli cloudprovider.ICloudRegion, args *BucketExistOptions) error {
		exist, err := cli.IBucketExist(args.NAME)
		if err != nil {
			return err
		}
		fmt.Printf("Exist: %v\n", exist)
		return nil
	})

	type BucketObjectsOptions struct {
		BUCKET    string `help:"name of bucket to list objects"`
		Prefix    string `help:"prefix"`
		Marker    string `help:"marker"`
		Demiliter string `help:"delimiter"`
		Max       int    `help:"Max count"`
	}
	shellutils.R(&BucketObjectsOptions{}, "bucket-object", "List objects in a bucket", func(cli cloudprovider.ICloudRegion, args *BucketObjectsOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		result, err := bucket.ListObjects(args.Prefix, args.Marker, args.Demiliter, args.Max)
		if err != nil {
			return err
		}
		if result.IsTruncated {
			fmt.Printf("NextMarker: %s IsTruncated: %v\n", result.NextMarker, result.IsTruncated)
		}
		fmt.Println("Common prefixes:")
		printutils.PrintGetterList(result.CommonPrefixes, []string{"key", "size_bytes"})
		fmt.Println("Objects:")
		printutils.PrintGetterList(result.Objects, []string{"key", "size_bytes"})
		return nil
	})

	type BucketListObjectsOptions struct {
		BUCKET string `help:"name of bucket to list objects"`
		Prefix string `help:"prefix"`
		Limit  int    `help:"limit per page request" default:"20"`
		Marker string `help:"offset marker"`
	}
	shellutils.R(&BucketListObjectsOptions{}, "bucket-list-object", "List objects in a bucket", func(cli cloudprovider.ICloudRegion, args *BucketListObjectsOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		objects, marker, err := cloudprovider.GetPagedObjects(bucket, args.Prefix, true, args.Marker, args.Limit)
		if err != nil {
			return err
		}
		printutils.PrintGetterList(objects, []string{"key", "size_bytes"})
		if len(marker) > 0 {
			fmt.Println("Next marker:", marker)
		}
		return nil
	})

	shellutils.R(&BucketListObjectsOptions{}, "bucket-dir-object", "List objects in a bucket like directory", func(cli cloudprovider.ICloudRegion, args *BucketListObjectsOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		objects, marker, err := cloudprovider.GetPagedObjects(bucket, args.Prefix, false, args.Marker, args.Limit)
		if err != nil {
			return err
		}
		printutils.PrintGetterList(objects, []string{"key", "size_bytes"})
		if len(marker) > 0 {
			fmt.Println("Next marker:", marker)
		}
		return nil
	})

	type BucketMakrdirOptions struct {
		BUCKET string `help:"name of bucket to put object"`
		DIR    string `help:"dir to make"`
	}
	shellutils.R(&BucketMakrdirOptions{}, "bucket-mkdir", "Mkdir in a bucket", func(cli cloudprovider.ICloudRegion, args *BucketMakrdirOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		err = cloudprovider.Makedir(context.Background(), bucket, args.DIR)
		if err != nil {
			return err
		}
		fmt.Printf("Mkdir success\n")
		return nil
	})

	type BucketPutObjectOptions struct {
		BUCKET string `help:"name of bucket to put object"`
		KEY    string `help:"key of object"`
		Path   string `help:"Path of file to upload"`

		BlockSize int64 `help:"blocksz in MB" default:"100"`

		Acl string `help:"acl" choices:"private|public-read|public-read-write"`

		StorageClass string `help:"storage class"`

		ObjectHeaderOptions
	}
	shellutils.R(&BucketPutObjectOptions{}, "put-object", "Put object into a bucket", func(cli cloudprovider.ICloudRegion, args *BucketPutObjectOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		var input io.ReadSeeker
		var fSize int64
		if len(args.Path) > 0 {
			finfo, err := os.Stat(args.Path)
			if err != nil {
				return errors.Wrap(err, "os.Stat")
			}
			fSize = finfo.Size()
			file, err := os.Open(args.Path)
			if err != nil {
				return errors.Wrap(err, "os.Open")
			}
			defer file.Close()

			input = file
		} else {
			input = os.Stdout
		}
		meta := args.ObjectHeaderOptions.Options2Header()
		err = cloudprovider.UploadObject(context.Background(), bucket, args.KEY, args.BlockSize*1000*1000, input, fSize, cloudprovider.TBucketACLType(args.Acl), args.StorageClass, meta, true)
		if err != nil {
			return err
		}
		fmt.Printf("Upload success\n")
		return nil
	})

	type BucketDeleteObjectOptions struct {
		BUCKET string `help:"name of bucket to put object"`
		KEY    string `help:"key of object"`
	}
	shellutils.R(&BucketDeleteObjectOptions{}, "delete-object", "Delete object from a bucket", func(cli cloudprovider.ICloudRegion, args *BucketDeleteObjectOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		err = bucket.DeleteObject(context.Background(), args.KEY)
		if err != nil {
			return err
		}
		fmt.Printf("Delete success\n")
		return nil
	})

	shellutils.R(&BucketDeleteObjectOptions{}, "delete-prefix", "Delete object from a bucket", func(cli cloudprovider.ICloudRegion, args *BucketDeleteObjectOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		err = cloudprovider.DeletePrefix(context.Background(), bucket, args.KEY)
		if err != nil {
			return err
		}
		fmt.Printf("Delete success\n")
		return nil
	})

	type BucketTempUrlOption struct {
		BUCKET   string `help:"name of bucket to put object"`
		METHOD   string `help:"http method" choices:"GET|PUT|DELETE"`
		KEY      string `help:"key of object"`
		Duration int    `help:"duration in seconds" default:"60"`
	}
	shellutils.R(&BucketTempUrlOption{}, "temp-url", "generate temp url", func(cli cloudprovider.ICloudRegion, args *BucketTempUrlOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		urlStr, err := bucket.GetTempUrl(args.METHOD, args.KEY, time.Duration(args.Duration)*time.Second)
		if err != nil {
			return err
		}
		fmt.Println(urlStr)
		return nil
	})

	type BucketAclOption struct {
		BUCKET string `help:"name of bucket to put object"`
		KEY    string `help:"key of object"`
	}
	shellutils.R(&BucketAclOption{}, "object-acl", "Get object acl", func(cli cloudprovider.ICloudRegion, args *BucketAclOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		object, err := cloudprovider.GetIObject(bucket, args.KEY)
		if err != nil {
			return err
		}
		fmt.Println(object.GetAcl())
		return nil
	})

	type BucketSetAclOption struct {
		BUCKET string `help:"name of bucket to put object"`
		KEY    string `help:"key of object"`
		ACL    string `help:"Target acl" choices:"default|private|public-read|public-read-write"`
	}
	shellutils.R(&BucketSetAclOption{}, "object-set-acl", "Get object acl", func(cli cloudprovider.ICloudRegion, args *BucketSetAclOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		object, err := cloudprovider.GetIObject(bucket, args.KEY)
		if err != nil {
			return err
		}
		err = object.SetAcl(cloudprovider.TBucketACLType(args.ACL))
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type BucketSetWebsiteOption struct {
		BUCKET string `help:"name of bucket to put object"`
		// 主页
		Index string `help:"main page"`
		// 错误时返回的文档
		ErrorDocument string `help:"error return"`
		// http或https
		Protocol string `help:"force https" choices:"http|https"`
	}
	shellutils.R(&BucketSetWebsiteOption{}, "bucket-set-website", "Set bucket website", func(cli cloudprovider.ICloudRegion, args *BucketSetWebsiteOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		conf := cloudprovider.SBucketWebsiteConf{
			Index:         args.Index,
			ErrorDocument: args.ErrorDocument,
			Protocol:      args.Protocol,
		}
		err = bucket.SetWebsite(conf)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type BucketGetWebsiteConfOption struct {
		BUCKET string `help:"name of bucket to put object"`
	}
	shellutils.R(&BucketGetWebsiteConfOption{}, "bucket-get-website", "Get bucket website", func(cli cloudprovider.ICloudRegion, args *BucketGetWebsiteConfOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		conf, err := bucket.GetWebsiteConf()
		if err != nil {
			return err
		}
		printObject(conf)
		return nil
	})

	type BucketDeleteWebsiteConfOption struct {
		BUCKET string `help:"name of bucket to put object"`
	}
	shellutils.R(&BucketDeleteWebsiteConfOption{}, "bucket-delete-website", "Delete bucket website", func(cli cloudprovider.ICloudRegion, args *BucketDeleteWebsiteConfOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		err = bucket.DeleteWebSiteConf()
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type BucketSetCorsOption struct {
		BUCKET         string `help:"name of bucket to put object"`
		AllowedMethods []string
		// 允许的源站，可以设为*
		AllowedOrigins []string
		AllowedHeaders []string
		MaxAgeSeconds  int
		ExposeHeaders  []string
		Id             string
	}
	shellutils.R(&BucketSetCorsOption{}, "bucket-set-cors", "Set bucket cors", func(cli cloudprovider.ICloudRegion, args *BucketSetCorsOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		rule := cloudprovider.SBucketCORSRule{
			AllowedOrigins: args.AllowedOrigins,
			AllowedMethods: args.AllowedMethods,
			AllowedHeaders: args.AllowedHeaders,
			MaxAgeSeconds:  args.MaxAgeSeconds,
			ExposeHeaders:  args.ExposeHeaders,
			Id:             args.Id,
		}
		err = cloudprovider.SetBucketCORS(bucket, []cloudprovider.SBucketCORSRule{rule})
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type BucketGetCorsOption struct {
		BUCKET string `help:"name of bucket to put object"`
	}
	shellutils.R(&BucketGetCorsOption{}, "bucket-get-cors", "Get bucket cors", func(cli cloudprovider.ICloudRegion, args *BucketGetCorsOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		rules, err := bucket.GetCORSRules()
		if err != nil {
			return err
		}
		printList(rules, len(rules), 0, len(rules), nil)
		return nil
	})

	type BucketDeleteCorsOption struct {
		BUCKET string   `help:"name of bucket to put object"`
		Ids    []string `help:"rule ids to delete"`
	}
	shellutils.R(&BucketDeleteCorsOption{}, "bucket-delete-cors", "Delete bucket cors", func(cli cloudprovider.ICloudRegion, args *BucketDeleteCorsOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		result, err := cloudprovider.DeleteBucketCORS(bucket, args.Ids)
		if err != nil {
			return err
		}
		printList(result, len(result), 0, len(result), nil)
		fmt.Println("Success!")
		return nil
	})

	type BucketSetRefererOption struct {
		BUCKET     string `help:"name of bucket to put object"`
		DomainList []string
		// 是否允许空refer 访问
		AllowEmptyRefer bool `help:"all empty refer access"`
	}
	shellutils.R(&BucketSetRefererOption{}, "bucket-set-referer", "Set bucket referer", func(cli cloudprovider.ICloudRegion, args *BucketSetRefererOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		conf := cloudprovider.SBucketRefererConf{
			WhiteList:       args.DomainList,
			AllowEmptyRefer: args.AllowEmptyRefer,
		}
		err = bucket.SetReferer(conf)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil

	})

	type BucketGetPolicyOption struct {
		BUCKET string `help:"name of bucket to put object"`
	}
	shellutils.R(&BucketGetPolicyOption{}, "bucket-get-policy", "get bucket policy", func(cli cloudprovider.ICloudRegion, args *BucketGetPolicyOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		policy, err := bucket.GetPolicy()
		if err != nil {
			return err
		}
		printList(policy, len(policy), 0, len(policy), nil)
		return nil
	})

	type BucketSetPolicyOption struct {
		BUCKET string `help:"name of bucket to put object"`
		// 格式主账号id:子账号id
		PrincipalId []string
		// Read|ReadWrite|FullControl
		CannedAction string
		// Allow|Deny
		Effect string
		// 被授权的资源地址
		ResourcePath []string
		// ip 条件
		IpEquals    []string
		IpNotEquals []string
	}
	shellutils.R(&BucketSetPolicyOption{}, "bucket-set-policy", "set bucket policy", func(cli cloudprovider.ICloudRegion, args *BucketSetPolicyOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		opts := cloudprovider.SBucketPolicyStatementInput{}
		opts.CannedAction = args.CannedAction
		opts.Effect = args.Effect
		opts.IpEquals = args.IpEquals
		opts.IpNotEquals = args.IpNotEquals
		opts.ResourcePath = args.ResourcePath
		opts.PrincipalId = args.PrincipalId

		err = bucket.SetPolicy(opts)
		if err != nil {
			return err
		}
		return nil
	})

	type BucketDeletePolicyOption struct {
		BUCKET string `help:"name of bucket to put object"`
		Id     []string
	}
	shellutils.R(&BucketDeletePolicyOption{}, "bucket-delete-policy", "delete bucket policy", func(cli cloudprovider.ICloudRegion, args *BucketDeletePolicyOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		result, err := bucket.DeletePolicy(args.Id)
		if err != nil {
			return err
		}
		printList(result, len(result), 0, len(result), nil)
		return nil
	})

	type BucketGetRefererOption struct {
		BUCKET string `help:"name of bucket to put object"`
	}
	shellutils.R(&BucketGetRefererOption{}, "bucket-get-referer", "get bucket referer", func(cli cloudprovider.ICloudRegion, args *BucketGetRefererOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		conf, err := bucket.GetReferer()
		if err != nil {
			return err
		}
		printObject(conf)
		return nil
	})

	type BucketGetCdnDomainOption struct {
		BUCKET string `help:"name of bucket to put object"`
	}
	shellutils.R(&BucketGetCdnDomainOption{}, "bucket-get-cdn-domains", "get bucket cdn domains", func(cli cloudprovider.ICloudRegion, args *BucketGetCdnDomainOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		domains, err := bucket.GetCdnDomains()
		if err != nil {
			return err
		}
		printList(domains, len(domains), 0, len(domains), nil)
		return nil
	})

	type BucketGetMetadata struct {
		BUCKET string `help:"name of bucket to put object"`
	}
	shellutils.R(&BucketGetMetadata{}, "bucket-get-metadata", "get bucket metadata", func(cli cloudprovider.ICloudRegion, args *BucketGetMetadata) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		meta := bucket.GetMetadata()
		printObject(meta)
		return nil
	})

	type BucketSetMetadate struct {
		BUCKET  string   `help:"name of bucket to put object"`
		Tags    []string `help:"Tags info, eg: hypervisor=aliyun、os_type=Linux、os_version"`
		Replace bool
	}
	shellutils.R(&BucketSetMetadate{}, "bucket-set-metadata", "set bucket metadata", func(cli cloudprovider.ICloudRegion, args *BucketSetMetadate) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		log.Infof("tags:%s", args.Tags)
		tags := map[string]string{}
		for _, tag := range args.Tags {
			pair := strings.Split(tag, "=")
			if len(pair) == 2 {
				tags[pair[0]] = pair[1]
			}
		}
		err = cloudprovider.SetBucketMetadata(bucket, tags, args.Replace)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type BucketObjectDownloadOptions struct {
		BUCKET string `help:"name of bucket"`
		KEY    string `help:"Key of object"`
		Output string `help:"target output, default to stdout"`
		Start  int64  `help:"partial download start"`
		End    int64  `help:"partial download end"`
	}
	shellutils.R(&BucketObjectDownloadOptions{}, "object-download", "Download", func(cli cloudprovider.ICloudRegion, args *BucketObjectDownloadOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		obj, err := cloudprovider.GetIObject(bucket, args.KEY)
		if err != nil {
			return err
		}

		var rangeOpt *cloudprovider.SGetObjectRange
		if args.Start != 0 || args.End != 0 {
			if args.End <= 0 {
				args.End = obj.GetSizeBytes() - 1
			}
			rangeOpt = &cloudprovider.SGetObjectRange{Start: args.Start, End: args.End}
		}
		output, err := bucket.GetObject(context.Background(), args.KEY, rangeOpt)
		if err != nil {
			return err
		}
		defer output.Close()
		var target io.Writer
		if len(args.Output) == 0 {
			target = os.Stdout
		} else {
			fp, err := os.Create(args.Output)
			if err != nil {
				return err
			}
			defer fp.Close()
			target = fp
		}
		prop, err := streamutils.StreamPipe(output, target, false, nil)
		if err != nil {
			return err
		}
		if len(args.Output) > 0 {
			fmt.Println("Success:", prop.Size, "written")
		}
		return nil
	})

	type BucketObjectCopyOptions struct {
		SRC       string `help:"name of source bucket"`
		SRCKEY    string `help:"Key of source object"`
		DST       string `help:"name of destination bucket"`
		DSTKEY    string `help:"key of destination object"`
		Debug     bool   `help:"show debug info"`
		BlockSize int64  `help:"block size in MB"`
		Native    bool   `help:"Use native copy"`

		ObjectHeaderOptions
	}
	shellutils.R(&BucketObjectCopyOptions{}, "object-copy", "Copy object", func(cli cloudprovider.ICloudRegion, args *BucketObjectCopyOptions) error {
		ctx := context.Background()
		dstBucket, err := cli.GetIBucketByName(args.DST)
		if err != nil {
			return err
		}
		srcBucket, err := cli.GetIBucketByName(args.SRC)
		if err != nil {
			return err
		}
		srcObj, err := cloudprovider.GetIObject(srcBucket, args.SRCKEY)
		if err != nil {
			return err
		}
		meta := args.ObjectHeaderOptions.Options2Header()
		if args.Native {
			err = dstBucket.CopyObject(ctx, args.DSTKEY, args.SRC, args.SRCKEY, srcObj.GetAcl(), srcObj.GetStorageClass(), meta)
			if err != nil {
				return err
			}
		} else {
			err = cloudprovider.CopyObject(ctx, args.BlockSize*1000*1000, dstBucket, args.DSTKEY, srcBucket, args.SRCKEY, meta, args.Debug)
			if err != nil {
				return err
			}
		}
		fmt.Println("Success!")
		return nil
	})

	type ObjectMetaOptions struct {
		BUCKET string `help:"bucket name"`
		KEY    string `help:"object key"`
	}
	shellutils.R(&ObjectMetaOptions{}, "object-meta", "Show object meta header", func(cli cloudprovider.ICloudRegion, args *ObjectMetaOptions) error {
		bucket, err := cli.GetIBucketByName(args.BUCKET)
		if err != nil {
			return err
		}
		obj, err := cloudprovider.GetIObject(bucket, args.KEY)
		if err != nil {
			return err
		}
		meta := obj.GetMeta()
		for k, v := range meta {
			fmt.Println(k, ": ", v[0])
		}
		return nil
	})

	type ObjectSetMetaOptions struct {
		BUCKET string `help:"bucket name"`
		KEY    string `help:"object key"`

		ObjectHeaderOptions
	}
	shellutils.R(&ObjectSetMetaOptions{}, "object-set-meta", "Set object meta header", func(cli cloudprovider.ICloudRegion, args *ObjectSetMetaOptions) error {
		bucket, err := cli.GetIBucketByName(args.BUCKET)
		if err != nil {
			return err
		}
		obj, err := cloudprovider.GetIObject(bucket, args.KEY)
		if err != nil {
			return err
		}
		meta := args.ObjectHeaderOptions.Options2Header()
		err = obj.SetMeta(context.Background(), meta)
		if err != nil {
			return err
		}
		return nil
	})
}
