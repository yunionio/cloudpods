package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type StorageAccountListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&StorageAccountListOptions{}, "storage-account-list", "List storage account", func(cli *azure.SRegion, args *StorageAccountListOptions) error {
		if accounts, err := cli.GetStorageAccounts(); err != nil {
			return err
		} else {
			printList(accounts, len(accounts), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type StorageAccountOptions struct {
		ID string `help:"StorageAccount ID"`
	}

	shellutils.R(&StorageAccountOptions{}, "storage-account-delete", "Delete storage account", func(cli *azure.SRegion, args *StorageAccountOptions) error {
		return cli.DeleteStorageAccount(args.ID)
	})

	shellutils.R(&StorageAccountOptions{}, "storage-account-show", "Show storage account detail", func(cli *azure.SRegion, args *StorageAccountOptions) error {
		if account, err := cli.GetStorageAccountDetail(args.ID); err != nil {
			return err
		} else {
			printObject(account)
			return nil
		}
	})

	shellutils.R(&StorageAccountOptions{}, "storage-account-key", "Get storage account key", func(cli *azure.SRegion, args *StorageAccountOptions) error {
		if key, err := cli.GetStorageAccountKey(args.ID); err != nil {
			return err
		} else {
			fmt.Printf("Key: %s", key)
			return nil
		}
	})

	type StorageAccountCreateOptions struct {
		NAME string `help:"StorageAccount NAME"`
	}

	shellutils.R(&StorageAccountCreateOptions{}, "storage-account-create", "Create a storage account", func(cli *azure.SRegion, args *StorageAccountCreateOptions) error {
		if account, err := cli.CreateStorageAccount(args.NAME); err != nil {
			return err
		} else {
			printObject(account)
			return nil
		}
	})

}
