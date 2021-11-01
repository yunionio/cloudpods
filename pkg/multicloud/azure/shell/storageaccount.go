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

	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type StorageAccountListOptions struct {
	}
	shellutils.R(&StorageAccountListOptions{}, "storage-account-list", "List storage account", func(cli *azure.SRegion, args *StorageAccountListOptions) error {
		accounts, err := cli.ListStorageAccounts()
		if err != nil {
			return err
		}
		printList(accounts, len(accounts), 0, 0, []string{})
		return nil
	})

	shellutils.R(&StorageAccountListOptions{}, "classic-storage-account-list", "List classic storage account", func(cli *azure.SRegion, args *StorageAccountListOptions) error {
		accounts, err := cli.ListClassicStorageAccounts()
		if err != nil {
			return err
		}
		printList(accounts, len(accounts), 0, 0, []string{})
		return nil
	})

	type StorageAccountOptions struct {
		ID string `help:"StorageAccount ID"`
	}

	shellutils.R(&StorageAccountOptions{}, "classic-storage-account-show", "Show storage account detail", func(cli *azure.SRegion, args *StorageAccountOptions) error {
		account, err := cli.GetClassicStorageAccount(args.ID)
		if err != nil {
			return err
		}
		printObject(account)
		return nil
	})

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
		blobs, err := container.ListAllFiles(nil)
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
		url, err := container.UploadFile(args.FILE, nil)
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
		uniqName, err := cli.GetUniqStorageAccountName()
		if err != nil {
			return err
		}
		fmt.Println(uniqName)
		return nil
	})

	type SStorageAccountSkuOptions struct {
	}
	shellutils.R(&SStorageAccountSkuOptions{}, "storage-account-skus", "List skus of storage account", func(cli *azure.SRegion, args *SStorageAccountSkuOptions) error {
		skus, err := cli.GetStorageAccountSkus()
		if err != nil {
			return err
		}
		printList(skus, 0, 0, 0, nil)
		return nil
	})
}
