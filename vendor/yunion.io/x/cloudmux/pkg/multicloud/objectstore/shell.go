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
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
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

	type BucketCORSListOptions struct {
		ID string
	}
	shellutils.R(&BucketCORSListOptions{}, "bucket-cors-list", "List all cors bucket", func(cli cloudprovider.ICloudRegion, args *BucketCORSListOptions) error {
		bucket, err := cli.GetIBucketById(args.ID)
		if err != nil {
			return errors.Wrap(err, "GetIBucketById")
		}
		cors, err := bucket.GetCORSRules()
		if err != nil {
			return errors.Wrap(err, "GetCORSRules")
		}
		printObject(cors)
		return nil
	})

	shellutils.R(&BucketCORSListOptions{}, "bucket-cors-delete", "delete all cors bucket", func(cli cloudprovider.ICloudRegion, args *BucketCORSListOptions) error {
		bucket, err := cli.GetIBucketById("yunion12")
		if err != nil {
			return errors.Wrap(err, "GetIBucketById")
		}
		err = bucket.DeleteCORS()
		if err != nil {
			return errors.Wrap(err, "DeleteCORS")
		}
		return nil
	})

	shellutils.R(&BucketCORSListOptions{}, "bucket-policy-list", "List all cors bucket", func(cli cloudprovider.ICloudRegion, args *BucketCORSListOptions) error {
		bucket, err := cli.GetIBucketById(args.ID)
		if err != nil {
			return errors.Wrap(err, "GetIBucketById")
		}
		policies, err := bucket.GetPolicy()
		if err != nil {
			return errors.Wrap(err, "GetPolicy")
		}
		printObject(policies)
		return nil
	})

	type BucketCORSDeleteOptions struct {
		ID    string
		INDEX string
	}
	shellutils.R(&BucketCORSDeleteOptions{}, "bucket-policy-delete", "delete all policy bucket", func(cli cloudprovider.ICloudRegion, args *BucketCORSDeleteOptions) error {
		bucket, err := cli.GetIBucketById(args.ID)
		if err != nil {
			return errors.Wrap(err, "GetIBucketById")
		}
		_, err = bucket.DeletePolicy([]string{args.INDEX})
		if err != nil {
			return errors.Wrap(err, "DeletePolicy")
		}
		return nil
	})

	type SetBucketPolicyInput struct {
		ID           string
		PrincipalId  string
		CannedAction string
		ResourcePath string
		Effect       string
	}
	shellutils.R(&SetBucketPolicyInput{}, "bucket-policy-create", "create policy bucket", func(cli cloudprovider.ICloudRegion, args *SetBucketPolicyInput) error {
		bucket, err := cli.GetIBucketById(args.ID)
		if err != nil {
			return errors.Wrap(err, "GetIBucketById")
		}
		err = bucket.SetPolicy(cloudprovider.SBucketPolicyStatementInput{
			PrincipalId:  []string{args.PrincipalId},
			Effect:       args.Effect,
			CannedAction: args.CannedAction,
			ResourcePath: []string{
				args.ResourcePath},
		})
		if err != nil {
			return errors.Wrap(err, "SetPolicy")
		}
		return nil
	})

	shellutils.R(&BucketCORSListOptions{}, "bucket-policy-list", "Show bucket detail", func(cli cloudprovider.ICloudRegion, args *BucketCORSListOptions) error {
		bucket, err := cli.GetIBucketById(args.ID)
		if err != nil {
			return err
		}
		res, err := bucket.GetPolicy()
		if err != nil {
			return errors.Wrap(err, "GetPolicy")
		}
		printObject(res)
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
	objectListFunc := func(cli cloudprovider.ICloudRegion, args *BucketObjectsOptions) error {
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
	}
	shellutils.R(&BucketObjectsOptions{}, "bucket-object", "List objects in a bucket (deprecated)", objectListFunc)
	shellutils.R(&BucketObjectsOptions{}, "object-list", "List objects in a bucket", objectListFunc)

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

		Parallel int `help:"upload object parts in parallel"`

		ObjectHeaderOptions
	}
	objectPutFunc := func(cli cloudprovider.ICloudRegion, args *BucketPutObjectOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}

		originMeta := args.ObjectHeaderOptions.Options2Header()

		if len(args.Path) > 0 {
			uploadFile := func(key, path string) error {
				meta := http.Header{}
				for k, v := range originMeta {
					meta[k] = v
				}

				finfo, err := os.Stat(path)
				if err != nil {
					return errors.Wrap(err, "os.Stat")
				}
				fSize := finfo.Size()
				file, err := os.Open(path)
				if err != nil {
					return errors.Wrap(err, "os.Open")
				}
				defer file.Close()

				if contTypes, ok := meta[cloudprovider.META_HEADER_CONTENT_TYPE]; !ok || len(contTypes) == 0 {
					var ext string
					lastSlashPos := strings.LastIndex(path, "/")
					lastExtPos := strings.LastIndex(path, ".")
					if lastExtPos >= 0 && lastExtPos > lastSlashPos {
						ext = path[lastExtPos:]
					}
					if len(ext) > 0 {
						contType := mime.TypeByExtension(ext)
						meta.Set(cloudprovider.META_HEADER_CONTENT_TYPE, contType)
					}
				}

				err = cloudprovider.UploadObjectParallel(context.Background(), bucket, key, args.BlockSize*1000*1000, file, fSize, cloudprovider.TBucketACLType(args.Acl), args.StorageClass, meta, true, args.Parallel)
				if err != nil {
					return err
				}

				return nil
			}
			if fileutils.IsFile(args.Path) {
				err := uploadFile(args.KEY, args.Path)
				if err != nil {
					return errors.Wrap(err, "uploadFile")
				}
			} else if fileutils.IsDir(args.Path) {
				return filepath.Walk(args.Path, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.Mode().IsRegular() {
						rel, _ := filepath.Rel(args.Path, path)
						src := path
						dst := filepath.Join(args.KEY, rel)
						fmt.Println("upload", src, "to", dst)
						uploadErr := uploadFile(dst, src)
						if uploadErr != nil {
							return uploadErr
						}
					}
					return nil
				})
			}
		} else {
			err = cloudprovider.UploadObject(context.Background(), bucket, args.KEY, args.BlockSize*1000*1000, os.Stdin, 0, cloudprovider.TBucketACLType(args.Acl), args.StorageClass, originMeta, true)
			if err != nil {
				return err
			}
		}

		fmt.Printf("Upload success\n")
		return nil
	}
	shellutils.R(&BucketPutObjectOptions{}, "put-object", "Put object into a bucket (deprecated)", objectPutFunc)
	shellutils.R(&BucketPutObjectOptions{}, "object-upload", "Upload object into a bucket", objectPutFunc)

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
		BUCKET      string `help:"name of bucket to put object"`
		RefererType string `help:"referer type" choices:"Black-List|White-List" default:"Black-List"`
		DomainList  []string
		// 是否允许空refer 访问
		AllowEmptyRefer bool `help:"all empty refer access"`
		Disable         bool
	}
	shellutils.R(&BucketSetRefererOption{}, "bucket-set-referer", "Set bucket referer", func(cli cloudprovider.ICloudRegion, args *BucketSetRefererOption) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		conf := cloudprovider.SBucketRefererConf{
			DomainList:      args.DomainList,
			RefererType:     args.RefererType,
			AllowEmptyRefer: args.AllowEmptyRefer,
			Enabled:         true,
		}
		if args.Disable {
			conf.Enabled = false
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
	shellutils.R(&BucketGetMetadata{}, "bucket-tag-list", "List bucket tag", func(cli cloudprovider.ICloudRegion, args *BucketGetMetadata) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		meta, err := bucket.GetTags()
		if err != nil {
			return err
		}
		printObject(meta)
		return nil
	})

	type BucketSetMetadate struct {
		BUCKET  string   `help:"name of bucket to put object"`
		Tags    []string `help:"Tags info, eg: hypervisor=aliyun、os_type=Linux、os_version"`
		Replace bool
	}
	shellutils.R(&BucketSetMetadate{}, "bucket-set-tag", "set bucket tag", func(cli cloudprovider.ICloudRegion, args *BucketSetMetadate) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		tags := map[string]string{}
		for _, tag := range args.Tags {
			pair := strings.Split(tag, "=")
			if len(pair) == 2 {
				tags[pair[0]] = pair[1]
			}
		}
		_, err = cloudprovider.SetBucketTags(context.Background(), bucket, "", tags)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type BucketGetUploads struct {
		BUCKET string `help:"name of bucket to put object"`
	}
	shellutils.R(&BucketGetUploads{}, "bucket-get-uploads", "get bucket uploads", func(cli cloudprovider.ICloudRegion, args *BucketGetUploads) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		uplaods, err := bucket.ListMultipartUploads()
		if err != nil {
			return err
		}
		printList(uplaods, len(uplaods), 0, len(uplaods), nil)
		return nil
	})

	type BucketObjectBatchDownloadOptions struct {
		BUCKET string `help:"name of bucket"`
		PREFIX string `help:"Prefix of object"`
		Output string `help:"target output directory, default to current directory"`
	}
	shellutils.R(&BucketObjectBatchDownloadOptions{}, "batch-download", "Download objects recursively", func(cli cloudprovider.ICloudRegion, args *BucketObjectBatchDownloadOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		marker := ""
		const maxCount = 1000
		for {
			results, err := bucket.ListObjects(args.PREFIX, marker, "", maxCount)
			if err != nil {
				return errors.Wrapf(err, "ListObjects prefix %s marker %s", args.PREFIX, marker)
			}
			for _, obj := range results.Objects {
				fmt.Println(obj.GetKey())
			}
			if !results.IsTruncated {
				break
			} else {
				marker = results.NextMarker
			}
		}
		return nil
	})

	type BucketObjectDownloadOptions struct {
		BUCKET string `help:"name of bucket"`
		KEY    string `help:"Key of object"`
		Output string `help:"target output, default to stdout"`
		Start  int64  `help:"partial download start"`
		End    int64  `help:"partial download end"`

		BlockSize int64 `help:"blocksz in MB" default:"100"`

		Parallel int `help:"upload object parts in parallel"`
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
		var target io.WriterAt
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

		sz, err := cloudprovider.DownloadObjectParallel(context.Background(), bucket, args.KEY, rangeOpt, target, 0, args.BlockSize*1000*1000, true, args.Parallel)
		if err != nil {
			return err
		}
		if len(args.Output) > 0 {
			fmt.Println("Success:", sz, "written")
		}
		return nil
	})

	type BucketObjectTempUrlOptions struct {
		BUCKET string `help:"name of bucket"`
		KEY    string `help:"Key of object"`
		Method string `default:"GET" choices:"GET|PUT|POST"`
		Hour   int64  `default:"1"`
	}

	shellutils.R(&BucketObjectTempUrlOptions{}, "object-temp-url", "Show object temp url", func(cli cloudprovider.ICloudRegion, args *BucketObjectTempUrlOptions) error {
		bucket, err := cli.GetIBucketById(args.BUCKET)
		if err != nil {
			return err
		}
		url, err := bucket.GetTempUrl(args.Method, args.KEY, time.Duration(args.Hour)*time.Hour)
		if err != nil {
			return err
		}
		fmt.Println(url)
		return nil
	})

	type BucketObjectCopyOptions struct {
		SRC    string `help:"name of source bucket"`
		SRCKEY string `help:"Key of source object"`
		DST    string `help:"name of destination bucket"`
		DSTKEY string `help:"key of destination object"`

		BlockSize int64 `help:"block size in MB"`
		Native    bool  `help:"Use native copy"`

		Parallel int `help:"copy object parts in parallel"`

		ObjectHeaderOptions
	}
	objectCopyFunc := func(cli cloudprovider.ICloudRegion, args *BucketObjectCopyOptions) error {
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
			err = cloudprovider.CopyObjectParallel(ctx, args.BlockSize*1000*1000, dstBucket, args.DSTKEY, srcBucket, args.SRCKEY, meta, true, args.Parallel)
			if err != nil {
				return err
			}
		}
		fmt.Println("Success!")
		return nil
	}
	shellutils.R(&BucketObjectCopyOptions{}, "object-copy", "Copy object", objectCopyFunc)

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
