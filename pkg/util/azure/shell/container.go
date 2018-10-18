package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ContainerListOptions struct {
		STORAGE string `helo:"Storage Account ID"`
		Limit   int    `help:"page size"`
		Offset  int    `help:"page offset"`
	}

	shellutils.R(&ContainerListOptions{}, "container-list", "List containers", func(cli *azure.SRegion, args *ContainerListOptions) error {
		storage, err := cli.GetStorageAccountDetail(args.STORAGE)
		if err != nil {
			return err
		}
		containers, err := cli.GetContainers(storage)
		if err != nil {
			return err
		}
		printList(containers, len(containers), args.Offset, args.Limit, []string{})
		return nil
	})

	type ContainerOptions struct {
		STORAGE string `helo:"Storage Name"`
		NAME    string `help:"Container Name"`
	}

	shellutils.R(&ContainerOptions{}, "container-show", "Show container detail", func(cli *azure.SRegion, args *ContainerOptions) error {
		storage, err := cli.GetStorageAccountDetail(args.STORAGE)
		if err != nil {
			return err
		}
		container, err := cli.GetContainerDetail(storage, args.NAME)
		if err != nil {
			return err
		}
		printObject(container)
		return nil
	})

	shellutils.R(&ContainerOptions{}, "container-create", "Create container", func(cli *azure.SRegion, args *ContainerOptions) error {
		storage, err := cli.GetStorageAccountDetail(args.STORAGE)
		if err != nil {
			return err
		}
		container, err := cli.CreateContainer(storage, args.NAME)
		if err != nil {
			return err
		}
		printObject(container)
		return nil
	})

	type ContainerBlobListOptions struct {
		STORAGE string `helo:"Storage Account ID"`
		NAME    string `help:"Container Name"`
		Limit   int    `help:"page size"`
		Offset  int    `help:"page offset"`
	}

	shellutils.R(&ContainerBlobListOptions{}, "container-blob-list", "List container files", func(cli *azure.SRegion, args *ContainerBlobListOptions) error {
		storage, err := cli.GetStorageAccountDetail(args.STORAGE)
		if err != nil {
			return err
		}
		blobs, err := cli.GetContainerBlobs(storage, args.NAME)
		if err != nil {
			return err
		}
		printList(blobs, len(blobs), args.Offset, args.Limit, []string{})
		return nil
	})

	type ContainerBlobOptions struct {
		STORAGE string `helo:"Storage Account ID"`
		NAME    string `help:"Container Name"`
		BLOB    string `help:"Blob Name"`
		Output  string `help:"Donwload path"`
	}

	shellutils.R(&ContainerBlobOptions{}, "container-blob-show", "List container blob detail", func(cli *azure.SRegion, args *ContainerBlobOptions) error {
		storage, err := cli.GetStorageAccountDetail(args.STORAGE)
		if err != nil {
			return err
		}
		blob, err := cli.GetContainerBlobDetail(storage, args.NAME, args.BLOB)
		if err != nil {
			return err
		}
		printObject(blob)
		return nil
	})

	shellutils.R(&ContainerBlobOptions{}, "container-blob-delete", "Delete container blob", func(cli *azure.SRegion, args *ContainerBlobOptions) error {
		storage, err := cli.GetStorageAccountDetail(args.STORAGE)
		if err != nil {
			return err
		}
		return cli.DeleteContainerBlob(storage, args.NAME, args.BLOB)
	})

	shellutils.R(&ContainerBlobOptions{}, "container-blob-upload", "Upload a vhd image to container", func(cli *azure.SRegion, args *ContainerBlobOptions) error {
		storage, err := cli.GetStorageAccountDetail(args.STORAGE)
		if err != nil {
			return err
		}
		uri, err := cli.UploadVHD(storage, args.NAME, args.BLOB)
		if err != nil {
			return err
		}
		fmt.Printf("upload to %s succeed", uri)
		return nil
	})

	// shellutils.R(&ContainerBlobOptions{}, "container-blob-download", "Donload file from container", func(cli *azure.SRegion, args *ContainerBlobOptions) error {
	// 	return cli.DownloadPageBlob(args.STORAGE, args.NAME, args.BLOB, args.Output)
	// })

	type ContainerBlobCreateOptions struct {
		STORAGE  string `helo:"Storage Account ID"`
		NAME     string `help:"Container Name"`
		Snapshot string `help:"Snapshot ID"`
	}

	shellutils.R(&ContainerBlobCreateOptions{}, "container-blob-create", "Create a blob to container from snapshot", func(cli *azure.SRegion, args *ContainerBlobCreateOptions) error {
		storage, err := cli.GetStorageAccountDetail(args.STORAGE)
		if err != nil {
			return err
		}
		blob, err := cli.CreateBlobFromSnapshot(storage, args.NAME, args.Snapshot)
		if err != nil {
			return err
		}
		printObject(blob)
		return nil
	})

}
