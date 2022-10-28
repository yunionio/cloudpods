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

	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type BucketListOptions struct {
		options.BaseListOptions
		DistinctField string `help:"query specified distinct field"`
	}
	R(&BucketListOptions{}, "bucket-list", "List all buckets", func(s *mcclient.ClientSession, args *BucketListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		if len(args.DistinctField) > 0 {
			params.Add(jsonutils.NewString(args.DistinctField), "extra_field")
			result, err := modules.Buckets.Get(s, "distinct-field", params)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		}
		result, err := modules.Buckets.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Buckets.GetColumns(s))
		return nil
	})

	type BucketIdOptions struct {
		ID string `help:"ID or name of bucket"`
	}
	R(&BucketIdOptions{}, "bucket-show", "Id details of bucket", func(s *mcclient.ClientSession, args *BucketIdOptions) error {
		result, err := modules.Buckets.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&BucketIdOptions{}, "bucket-syncstatus", "Sync bucket statust", func(s *mcclient.ClientSession, args *BucketIdOptions) error {
		result, err := modules.Buckets.PerformAction(s, args.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketUpdateOptions struct {
		ID   string `help:"ID or name of bucket" json:"-"`
		Name string `help:"new name of bucket" json:"name"`
		Desc string `help:"Description of bucket" json:"description" token:"desc"`
	}
	R(&BucketUpdateOptions{}, "bucket-update", "update bucket", func(s *mcclient.ClientSession, args *BucketUpdateOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Buckets.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketDeleteOptions struct {
		ID string `help:"ID or name of bucket" json:"-"`
	}
	R(&BucketDeleteOptions{}, "bucket-delete", "delete bucket", func(s *mcclient.ClientSession, args *BucketDeleteOptions) error {
		result, err := modules.Buckets.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketCreateOptions struct {
		NAME        string `help:"name of bucket" json:"name"`
		CLOUDREGION string `help:"location of bucket" json:"cloudregion"`
		MANAGER     string `help:"cloud provider" json:"manager"`

		StorageClass string `help:"bucket storage class"`
		Acl          string `help:"bucket ACL"`
	}
	R(&BucketCreateOptions{}, "bucket-create", "Create a bucket", func(s *mcclient.ClientSession, args *BucketCreateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Buckets.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketListObjectsOptions struct {
		ID           string `help:"ID or name of bucket" json:"-"`
		Prefix       string `help:"List objects with prefix"`
		Recursive    bool   `help:"List objects recursively"`
		Limit        int    `help:"maximal items per request"`
		PagingMarker string `help:"paging marker"`
	}
	R(&BucketListObjectsOptions{}, "bucket-object-list", "List objects in a bucket", func(s *mcclient.ClientSession, args *BucketListObjectsOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Buckets.GetSpecific(s, args.ID, "objects", params)
		if err != nil {
			return err
		}

		listResult := modulebase.ListResult{}
		err = result.Unmarshal(&listResult)
		if err != nil {
			return err
		}
		printList(&listResult, []string{})
		return nil
	})

	type BucketDeleteObjectsOptions struct {
		ID   string   `help:"ID or name of bucket" json:"-"`
		KEYS []string `help:"List of objects to delete"`
	}
	R(&BucketDeleteObjectsOptions{}, "bucket-object-delete", "Delete objects in a bucket", func(s *mcclient.ClientSession, args *BucketDeleteObjectsOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Buckets.PerformAction(s, args.ID, "delete", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketMakeDirOptions struct {
		ID  string `help:"ID or name of bucket" json:"-"`
		KEY string `help:"DIR key to create"`
	}
	R(&BucketMakeDirOptions{}, "bucket-mkdir", "Mkdir in a bucket", func(s *mcclient.ClientSession, args *BucketMakeDirOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Buckets.PerformAction(s, args.ID, "makedir", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketUploadObjectsOptions struct {
		ID   string `help:"ID or name of bucket" json:"-"`
		KEY  string `help:"Key of object to upload"`
		Path string `help:"Path to file to upload" required:"true"`

		ContentLength int64  `help:"Content lenght (bytes)" default:"-1"`
		StorageClass  string `help:"storage CLass"`
		Acl           string `help:"object acl." choices:"private|public-read|public-read-write"`

		objectstore.ObjectHeaderOptions
	}
	R(&BucketUploadObjectsOptions{}, "bucket-object-upload", "Upload an object into a bucket", func(s *mcclient.ClientSession, args *BucketUploadObjectsOptions) error {
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

	type BucketPresignObjectsOptions struct {
		ID            string `help:"ID or name of bucket" json:"-"`
		KEY           string `help:"Key of object to upload"`
		Method        string `help:"Request method" choices:"GET|PUT|DELETE"`
		ExpireSeconds int    `help:"expire in seconds" default:"60"`
	}
	R(&BucketPresignObjectsOptions{}, "bucket-object-tempurl", "Get temporal URL for an object in a bucket", func(s *mcclient.ClientSession, args *BucketPresignObjectsOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Buckets.PerformAction(s, args.ID, "temp-url", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketSetAclOptions struct {
		ID  string   `help:"ID or name of bucket" json:"-"`
		ACL string   `help:"ACL to set" choices:"default|private|public-read|public-read-write" json:"acl"`
		Key []string `help:"Optional object key" json:"key"`
	}
	R(&BucketSetAclOptions{}, "bucket-set-acl", "Set ACL of bucket or object", func(s *mcclient.ClientSession, args *BucketSetAclOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Buckets.PerformAction(s, args.ID, "acl", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketAclOptions struct {
		ID  string `help:"ID or name of bucket" json:"-"`
		Key string `help:"Optional object key"`
	}
	R(&BucketAclOptions{}, "bucket-acl", "Get ACL of bucket or object", func(s *mcclient.ClientSession, args *BucketAclOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Buckets.GetSpecific(s, args.ID, "acl", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketSyncOptions struct {
		ID        string `help:"ID or name of bucket" json:"-"`
		StatsOnly bool   `help:"sync statistics only"`
	}
	R(&BucketSyncOptions{}, "bucket-sync", "Sync bucket", func(s *mcclient.ClientSession, args *BucketSyncOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Buckets.PerformAction(s, args.ID, "sync", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketLimitOptions struct {
		ID          string `help:"ID or name of bucket" json:"-"`
		SizeBytes   int64  `help:"size limit in bytes"`
		ObjectCount int64  `help:"object count limit"`
	}
	R(&BucketLimitOptions{}, "bucket-limit", "Set limit of bucket", func(s *mcclient.ClientSession, args *BucketLimitOptions) error {
		limit := jsonutils.Marshal(args)
		params := jsonutils.NewDict()
		params.Set("limit", limit)
		result, err := modules.Buckets.PerformAction(s, args.ID, "limit", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketAccessInfoOptions struct {
		ID string `help:"ID or name of bucket" json:"-"`
	}
	R(&BucketAccessInfoOptions{}, "bucket-access-info", "Show backend access info of a bucket", func(s *mcclient.ClientSession, args *BucketAccessInfoOptions) error {
		result, err := modules.Buckets.GetSpecific(s, args.ID, "access-info", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketSetMetadataOptions struct {
		ID string `help:"ID or name of bucket" json:"-"`

		Key []string `help:"Optional object key" json:"key"`

		objectstore.ObjectHeaderOptions
	}
	R(&BucketSetMetadataOptions{}, "bucket-set-metadata", "Set metadata of object", func(s *mcclient.ClientSession, args *BucketSetMetadataOptions) error {
		input := api.BucketMetadataInput{}
		input.Key = args.Key
		input.Metadata = args.ObjectHeaderOptions.Options2Header()
		err := input.Validate()
		if err != nil {
			return err
		}
		result, err := modules.Buckets.PerformAction(s, args.ID, "metadata", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketSetWebsiteOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
		// 主页
		Index string `help:"main page"`
		// 错误时返回的文档
		ErrorDocument string `help:"error return"`
		// http或https
		Protocol string `help:"force https" choices:"http|https"`
	}
	R(&BucketSetWebsiteOption{}, "bucket-set-website", "Set bucket website", func(s *mcclient.ClientSession, args *BucketSetWebsiteOption) error {
		conf := api.BucketWebsiteConf{
			Index:         args.Index,
			ErrorDocument: args.ErrorDocument,
			Protocol:      args.Protocol,
		}
		result, err := modules.Buckets.PerformAction(s, args.ID, "set-website", jsonutils.Marshal(conf))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketGetWebsiteConfOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
	}
	R(&BucketGetWebsiteConfOption{}, "bucket-get-website", "Get bucket website", func(s *mcclient.ClientSession, args *BucketGetWebsiteConfOption) error {
		result, err := modules.Buckets.GetSpecific(s, args.ID, "website", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketDeleteWebsiteConfOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
	}
	R(&BucketDeleteWebsiteConfOption{}, "bucket-delete-website", "Delete bucket website", func(s *mcclient.ClientSession, args *BucketDeleteWebsiteConfOption) error {
		result, err := modules.Buckets.PerformAction(s, args.ID, "delete-website", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketSetCorsOption struct {
		ID             string   `help:"ID or name of bucket" json:"-"`
		AllowedMethods []string `help:"allowed http method" choices:"PUT|GET|POST|DELETE|HEAD"`
		// 允许的源站，可以设为*
		AllowedOrigins []string
		AllowedHeaders []string
		MaxAgeSeconds  int
		ExposeHeaders  []string
		RuleId         string
	}
	R(&BucketSetCorsOption{}, "bucket-set-cors", "Set bucket cors", func(s *mcclient.ClientSession, args *BucketSetCorsOption) error {

		rule := api.BucketCORSRule{
			AllowedOrigins: args.AllowedOrigins,
			AllowedMethods: args.AllowedMethods,
			AllowedHeaders: args.AllowedHeaders,
			MaxAgeSeconds:  args.MaxAgeSeconds,
			ExposeHeaders:  args.ExposeHeaders,
			Id:             args.RuleId,
		}
		rules := api.BucketCORSRules{Data: []api.BucketCORSRule{rule}}
		result, err := modules.Buckets.PerformAction(s, args.ID, "set-cors", jsonutils.Marshal(rules))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketGetCorsOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
	}
	R(&BucketGetCorsOption{}, "bucket-get-cors", "Get bucket cors", func(s *mcclient.ClientSession, args *BucketGetCorsOption) error {
		result, err := modules.Buckets.GetSpecific(s, args.ID, "cors", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketDeleteCorsOption struct {
		ID string   `help:"ID or name of bucket" json:"-"`
		Id []string `"help:Id of rules to delete"`
	}
	R(&BucketDeleteCorsOption{}, "bucket-delete-cors", "Delete bucket cors", func(s *mcclient.ClientSession, args *BucketDeleteCorsOption) error {
		input := api.BucketCORSRuleDeleteInput{}
		input.Id = args.Id
		result, err := modules.Buckets.PerformAction(s, args.ID, "delete-cors", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketSetRefererOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
		// 域名列表
		DomainList []string
		// 是否允许空referer 访问
		AllowEmptyRefer bool `help:"all empty refer access"`
		Enabled         bool
		RerererType     string `help:"Referer type" choices:"Black-List|White-List"`
	}
	R(&BucketSetRefererOption{}, "bucket-set-referer", "Set bucket referer", func(s *mcclient.ClientSession, args *BucketSetRefererOption) error {
		conf := api.BucketRefererConf{
			Enabled:         args.Enabled,
			AllowEmptyRefer: args.AllowEmptyRefer,
			RefererType:     args.RerererType,
			DomainList:      args.DomainList,
		}
		result, err := modules.Buckets.PerformAction(s, args.ID, "set-referer", jsonutils.Marshal(conf))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketGetRefererOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
	}
	R(&BucketGetRefererOption{}, "bucket-get-referer", "get bucket referer", func(s *mcclient.ClientSession, args *BucketGetRefererOption) error {
		result, err := modules.Buckets.GetSpecific(s, args.ID, "referer", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketGetCdnDomainOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
	}
	R(&BucketGetRefererOption{}, "bucket-get-cdn-domain", "get bucket cdn domain", func(s *mcclient.ClientSession, args *BucketGetRefererOption) error {
		result, err := modules.Buckets.GetSpecific(s, args.ID, "cdn-domain", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketGetPolicyOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
	}
	R(&BucketGetPolicyOption{}, "bucket-get-policy", "get bucket policy", func(s *mcclient.ClientSession, args *BucketGetPolicyOption) error {
		result, err := modules.Buckets.GetSpecific(s, args.ID, "policy", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketSetPolicyOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
		// 格式主账号id:子账号id
		PrincipalId []string `help:"ext account id, accountId:subaccountId"`
		// Read|ReadWrite|FullControl
		CannedAction string `help:"authority action" choice:"Read|FullControl"`
		// Allow|Deny
		Effect string `help:"allow or deny" choice:"Allow|Deny"`
		// 被授权的资源地址
		ResourcePath []string
		// ip 条件
		IpEquals    []string
		IpNotEquals []string
	}
	R(&BucketSetPolicyOption{}, "bucket-set-policy", "set bucket policy", func(s *mcclient.ClientSession, args *BucketSetPolicyOption) error {
		opts := api.BucketPolicyStatementInput{}
		opts.CannedAction = args.CannedAction
		opts.Effect = args.Effect
		opts.IpEquals = args.IpEquals
		opts.IpNotEquals = args.IpNotEquals
		opts.ResourcePath = args.ResourcePath
		opts.PrincipalId = args.PrincipalId

		result, err := modules.Buckets.PerformAction(s, args.ID, "set-policy", jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type BucketDeletePolicyOption struct {
		ID string `help:"ID or name of bucket" json:"-"`
		Id []string
	}
	R(&BucketDeletePolicyOption{}, "bucket-delete-policy", "delete bucket policy", func(s *mcclient.ClientSession, args *BucketDeletePolicyOption) error {
		input := api.BucketPolicyDeleteInput{}
		input.Id = args.Id
		result, err := modules.Buckets.PerformAction(s, args.ID, "delete-policy", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
