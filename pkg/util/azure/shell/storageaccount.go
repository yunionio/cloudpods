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
		account, err := cli.GetStorageAccountDetail(args.ID)
		if err != nil {
			return err
		}
		printObject(account)
		return nil
	})

	shellutils.R(&StorageAccountOptions{}, "storage-account-key", "Get storage account key", func(cli *azure.SRegion, args *StorageAccountOptions) error {
		if key, err := cli.GetStorageAccountKey(args.ID); err != nil {
			return err
		} else {
			fmt.Printf("Key: %s", key)
			return nil
		}
	})

	shellutils.R(&StorageAccountOptions{}, "storage-container-list", "Get list of containers of a storage account", func(cli *azure.SRegion, args *StorageAccountOptions) error {
		account, err := cli.GetStorageAccountDetail(args.ID)
		if err != nil {
			return err
		}
		containers, err := account.GetContainers()
		if err != nil {
			return err
		}
		printList(containers, len(containers), 0, 0, nil)
		return nil
	})

	type StorageAccountCreateContainerOptions struct {
		ACCOUNT   string `help:"storage account ID"`
		CONTAINER string `help:"name of container to create"`
	}
	shellutils.R(&StorageAccountCreateContainerOptions{}, "storage-container-create", "Create a container in a storage account", func(cli *azure.SRegion, args *StorageAccountCreateContainerOptions) error {
		account, err := cli.GetStorageAccountDetail(args.ACCOUNT)
		if err != nil {
			return err
		}
		container, err := account.CreateContainer(args.CONTAINER)
		if err != nil {
			return err
		}
		printObject(container)
		return nil
	})

	shellutils.R(&StorageAccountCreateContainerOptions{}, "storage-container-list-objects", "Create a container in a storage account", func(cli *azure.SRegion, args *StorageAccountCreateContainerOptions) error {
		account, err := cli.GetStorageAccountDetail(args.ACCOUNT)
		if err != nil {
			return err
		}
		container, err := account.GetContainer(args.CONTAINER)
		if err != nil {
			return err
		}
		blobs, err := container.ListFiles()
		if err != nil {
			return err
		}
		printList(blobs, len(blobs), 0, 0, nil)
		return nil
	})

	type StorageAccountUploadOptions struct {
		ACCOUNT   string `help:"storage account ID"`
		CONTAINER string `help:"name of container to create"`
		FILE      string `help:"local file to upload"`
	}
	shellutils.R(&StorageAccountUploadOptions{}, "storage-container-upload", "Upload a container in a storage account", func(cli *azure.SRegion, args *StorageAccountUploadOptions) error {
		account, err := cli.GetStorageAccountDetail(args.ACCOUNT)
		if err != nil {
			return err
		}
		container, err := account.GetContainer(args.CONTAINER)
		if err != nil {
			return err
		}
		url, err := container.UploadFile(args.FILE)
		if err != nil {
			return err
		}
		fmt.Println(url)
		return nil
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

	type StorageAccountCheckeOptions struct {
	}

	shellutils.R(&StorageAccountCheckeOptions{}, "storage-uniq-name", "Get a uniqel storage account name", func(cli *azure.SRegion, args *StorageAccountCheckeOptions) error {
		uniqName := cli.GetUniqStorageAccountName()
		fmt.Println(uniqName)
		return nil
	})

}
