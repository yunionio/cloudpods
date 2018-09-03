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
		if err := cli.CheckBlob(args.ResourceGroup, args.StorageAccount, args.BlobName); err != nil {
			return err
		}
		return nil
		// if images, err := cli.GetImages(); err != nil {
		// 	return err
		// } else {
		// 	printList(images, len(images), args.Offset, args.Limit, []string{})
		// 	return nil
		// }
	})
}
