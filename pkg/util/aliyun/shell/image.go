package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Status string   `help:"image status type" choices:"Creating|Available|UnAvailable|CreateFailed"`
		Owner  string   `help:"Owner type" choices:"system|self|others|marketplace"`
		Id     []string `help:"Image ID"`
		Name   string   `help:"image name"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *aliyun.SRegion, args *ImageListOptions) error {
		images, total, e := cli.GetImages(aliyun.ImageStatusType(args.Status), aliyun.ImageOwnerType(args.Owner), args.Id, args.Name, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(images, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type ImageShowOptions struct {
		ID string `help:"image ID"`
	}
	shellutils.R(&ImageShowOptions{}, "image-show", "Show image", func(cli *aliyun.SRegion, args *ImageShowOptions) error {
		img, err := cli.GetImage(args.ID)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `help:"ID or Name to delete"`
	}
	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *aliyun.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})

	type ImageCreateOptions struct {
		SNAPSHOT string `help:"Snapshot id"`
		NAME     string `help:"Image name"`
		Desc     string `help:"Image desc"`
	}
	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *aliyun.SRegion, args *ImageCreateOptions) error {
		imageId, err := cli.CreateImage(args.SNAPSHOT, args.NAME, args.Desc)
		if err != nil {
			return err
		}
		fmt.Println(imageId)
		return nil
	})

	type ImageExportOptions struct {
		ID     string `help:"ID or Name to export"`
		BUCKET string `help:"Bucket name"`
	}

	shellutils.R(&ImageExportOptions{}, "image-export", "Export image", func(cli *aliyun.SRegion, args *ImageExportOptions) error {
		oss, err := cli.GetOssClient()
		if err != nil {
			return err
		}
		exist, err := oss.IsBucketExist(args.BUCKET)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("not exist bucket %s", args.BUCKET)
		}
		bucket, err := oss.Bucket(args.BUCKET)
		if err != nil {
			return err
		}
		task, err := cli.ExportImage(args.ID, bucket)
		if err != nil {
			return err
		}
		printObject(task)
		return nil
	})
}
