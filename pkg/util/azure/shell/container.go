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
		if containers, err := cli.GetContainers(args.STORAGE); err != nil {
			return err
		} else {
			printList(containers, len(containers), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type ContainerOptions struct {
		STORAGE string `helo:"Storage Name"`
		NAME    string `help:"Container Name"`
	}

	shellutils.R(&ContainerOptions{}, "container-show", "Show container detail", func(cli *azure.SRegion, args *ContainerOptions) error {
		if container, err := cli.GetContainerDetail(args.STORAGE, args.NAME); err != nil {
			return err
		} else {
			printObject(container)
			return nil
		}
	})

	shellutils.R(&ContainerOptions{}, "container-create", "Create container", func(cli *azure.SRegion, args *ContainerOptions) error {
		if container, err := cli.CreateContainer(args.STORAGE, args.NAME); err != nil {
			return err
		} else {
			printObject(container)
			return nil
		}
	})

	type ContainerBlobListOptions struct {
		STORAGE string `helo:"Storage Account ID"`
		NAME    string `help:"Container Name"`
		Limit   int    `help:"page size"`
		Offset  int    `help:"page offset"`
	}

	shellutils.R(&ContainerBlobListOptions{}, "container-blob-list", "List container files", func(cli *azure.SRegion, args *ContainerBlobListOptions) error {
		if blobs, err := cli.GetContainerBlobs(args.STORAGE, args.NAME); err != nil {
			return err
		} else {
			printList(blobs, len(blobs), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type ContainerBlobOptions struct {
		STORAGE string `helo:"Storage Account ID"`
		NAME    string `help:"Container Name"`
		BLOB    string `help:"Blob Name"`
		Output  string `help:"Donwload path"`
	}

	shellutils.R(&ContainerBlobOptions{}, "container-blob-show", "List container blob detail", func(cli *azure.SRegion, args *ContainerBlobOptions) error {
		if blob, err := cli.GetContainerBlobDetail(args.STORAGE, args.NAME, args.BLOB); err != nil {
			return err
		} else {
			printObject(blob)
			return nil
		}
	})

	shellutils.R(&ContainerBlobOptions{}, "container-blob-delete", "Delete container blob", func(cli *azure.SRegion, args *ContainerBlobOptions) error {
		return cli.DeleteContainerBlob(args.STORAGE, args.NAME, args.BLOB)
	})

	shellutils.R(&ContainerBlobOptions{}, "container-blob-upload", "Upload a vhd image to container", func(cli *azure.SRegion, args *ContainerBlobOptions) error {
		if uri, err := cli.UploadVHD(args.STORAGE, args.NAME, args.BLOB); err != nil {
			return err
		} else {
			fmt.Printf("upload to %s succeed", uri)
			return nil
		}
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
		if blob, err := cli.CreateBlobFromSnapshot(args.STORAGE, args.NAME, args.Snapshot); err != nil {
			return err
		} else {
			printObject(blob)
			return nil
		}
	})

}
