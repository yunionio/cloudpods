package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type BlobListOptions struct {
		ResourceGroup  string `help:"Resource Group Name`
		StorageAccount string `helo:"Storage Account"`
		BlobName       string `helo:"Blob name`
		Limit          int    `help:"page size"`
		Offset         int    `help:"page offset"`
	}
	shellutils.R(&BlobListOptions{}, "blob-file-list", "List intances", func(cli *azure.SRegion, args *BlobListOptions) error {
		if err := cli.CheckBlobContainer(args.ResourceGroup, args.StorageAccount, args.BlobName); err != nil {
			return err
		} else if result, err := cli.ListContainerFiles(args.ResourceGroup, args.StorageAccount, args.BlobName); err != nil {
			return err
		} else {
			printList(result, len(result), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type BlobUploadOptions struct {
		ResourceGroup  string `help:"Resource Group Name`
		StorageAccount string `helo:"Storage Account"`
		BlobName       string `helo:"Blob name`
		FilePath       string `helo:"Filet path to upload`
		Limit          int    `help:"page size"`
		Offset         int    `help:"page offset"`
	}

	shellutils.R(&BlobUploadOptions{}, "blob-file-upload", "Upload file to blob", func(cli *azure.SRegion, args *BlobUploadOptions) error {
		if err := cli.CheckBlobContainer(args.ResourceGroup, args.StorageAccount, args.BlobName); err != nil {
			return err
		} else if url, err := cli.UploadContainerFiles(args.ResourceGroup, args.StorageAccount, args.BlobName, args.FilePath); err != nil {
			return err
		} else {
			printObject(url)
			return nil
		}
	})
}
