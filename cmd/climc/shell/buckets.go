package shell

import (
	"yunion.io/x/jsonutils"

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
}
