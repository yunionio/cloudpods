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
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type BucketListOptions struct {
		options.BaseListOptions
	}
	R(&BucketListOptions{}, "bucket-list", "List all buckets", func(s *mcclient.ClientSession, args *BucketListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Buckets.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Buckets.GetColumns(s))
		return nil
	})

	type BucketShowOptions struct {
		ID string `help:"ID or name of bucket"`
	}
	R(&BucketShowOptions{}, "bucket-show", "Show details of bucket", func(s *mcclient.ClientSession, args *BucketShowOptions) error {
		result, err := modules.Buckets.Get(s, args.ID, nil)
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
		ID        string `help:"ID or name of bucket" json:"-"`
		Prefix    string `help:"List objects with prefix"`
		Recursive bool   `help:"List objects recursively"`
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

		arrays, _ := result.GetArray("objects")
		listResult := modulebase.ListResult{Data: arrays}
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
		ContentType   string `help:"Content type"`
		StorageClass  string `help:"storage CLass"`
		Acl           string `help:"object acl." choices:"private|public-read|public-read-write"`
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

		err := modules.Buckets.Upload(s, args.ID, args.KEY, body, args.ContentLength, args.ContentType, args.StorageClass, args.Acl)
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
}
