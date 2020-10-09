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
	"context"
	"fmt"
	"os"

	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func getDatastore(cli *esxi.SESXiClient, dcId string, dsId string) (*esxi.SDatastore, error) {
	dc, err := cli.FindDatacenterByMoId(dcId)
	if err != nil {
		return nil, err
	}
	ds, err := dc.GetIStorageByMoId(dsId)
	if err != nil {
		return nil, err
	}
	return ds.(*esxi.SDatastore), nil
}

func init() {
	type DatastoreListOptions struct {
		DATACENTER string `help:"List datastores in datacenter"`
	}
	shellutils.R(&DatastoreListOptions{}, "ds-list", "List datastores in datacenter", func(cli *esxi.SESXiClient, args *DatastoreListOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorages()
		if err != nil {
			return err
		}
		printList(ds, nil)
		return nil
	})

	type DatastoreShowOptions struct {
		DATACENTER string `help:"Datacenter"`
		DSID       string `help:"Datastore ID"`
	}
	shellutils.R(&DatastoreShowOptions{}, "ds-show", "Show details of a datastore", func(cli *esxi.SESXiClient, args *DatastoreShowOptions) error {
		ds, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		printObject(ds)
		return nil
	})

	shellutils.R(&DatastoreShowOptions{}, "ds-cache-show", "Show details of a datastore image cache", func(cli *esxi.SESXiClient, args *DatastoreShowOptions) error {
		ds, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		cache := ds.GetIStoragecache()
		printObject(cache)
		return nil
	})

	shellutils.R(&DatastoreShowOptions{}, "ds-faketemplate-list", "Show image list of a datastore image cache", func(cli *esxi.SESXiClient, args *DatastoreShowOptions) error {
		ds, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		icache := ds.GetIStoragecache()
		cache := icache.(*esxi.SDatastoreImageCache)
		vms, err := cache.GetFakeTempateVM("^((?i)(VMWARE-))")
		if err != nil {
			return err
		}
		printList(vms, []string{})
		return nil
	})
	shellutils.R(&DatastoreShowOptions{}, "ds-cache-list", "Show image list of a datastore image cache", func(cli *esxi.SESXiClient, args *DatastoreShowOptions) error {
		ds, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		cache := ds.GetIStoragecache()
		images, err := cache.GetIImages()
		if err != nil {
			return err
		}
		printList(images, []string{})
		return nil
	})

	type DatastoreListDirOptions struct {
		DATACENTER string `help:"Datacenter"`
		DSID       string `help:"Datastore ID"`
		DIR        string `help:"directory"`
	}
	shellutils.R(&DatastoreListDirOptions{}, "ds-list-dir", "List directory of a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dsObj, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		fileList, err := dsObj.ListDir(ctx, args.DIR)
		if err != nil {
			return err
		}
		printutils.PrintInterfaceList(fileList, 0, 0, 0, []string{"Name", "Date", "Size"})
		return nil
	})

	shellutils.R(&DatastoreListDirOptions{}, "ds-check-file", "Check file status in a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dsObj, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		file, err := dsObj.CheckFile(ctx, args.DIR)
		if err != nil {
			return err
		}
		printutils.PrintInterfaceObject(file)
		return nil
	})

	shellutils.R(&DatastoreListDirOptions{}, "ds-delete-file", "Delete file in a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dsObj, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		err = dsObj.Delete(ctx, args.DIR)
		if err != nil {
			return err
		}
		fmt.Println("success")
		return nil
	})

	shellutils.R(&DatastoreListDirOptions{}, "ds-check-vmdk", "Check vmdk file status in a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dsObj, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		err = dsObj.CheckVmdk(ctx, args.DIR)
		if err != nil {
			return err
		}
		fmt.Println("valid")
		return nil
	})

	shellutils.R(&DatastoreListDirOptions{}, "ds-delete-vmdk", "Delete vmdk file from a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dsObj, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		err = dsObj.DeleteVmdk(ctx, args.DIR)
		if err != nil {
			return err
		}
		fmt.Println("success")
		return nil
	})

	shellutils.R(&DatastoreListDirOptions{}, "ds-mkdir", "Delete vmdk directory from a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dsObj, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		err = dsObj.MakeDir(args.DIR)
		if err != nil {
			return err
		}
		fmt.Println("Make dir success")
		return nil
	})

	shellutils.R(&DatastoreListDirOptions{}, "ds-rmdir", "Remove vmdk directory from a datastore", func(cli *esxi.SESXiClient, args *DatastoreListDirOptions) error {
		dsObj, err := getDatastore(cli, args.DATACENTER, args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		err = dsObj.RemoveDir(ctx, args.DIR)
		if err != nil {
			return err
		}
		fmt.Println("Remove dir success")
		return nil
	})

	type DatastoreDownloadOptions struct {
		DATACENTER string `help:"Datacenter"`
		DSID       string `help:"Datastore ID"`
		DIR        string `help:"directory"`
		LOCAL      string `help:"local file"`
	}
	shellutils.R(&DatastoreDownloadOptions{}, "ds-download", "Download file from a datastore", func(cli *esxi.SESXiClient, args *DatastoreDownloadOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorageByMoId(args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		dsObj := ds.(*esxi.SDatastore)

		file, err := os.Create(args.LOCAL)
		if err != nil {
			return err
		}
		defer file.Close()

		err = dsObj.Download(ctx, args.DIR, file)
		if err != nil {
			return err
		}

		return nil
	})

	shellutils.R(&DatastoreDownloadOptions{}, "ds-upload", "Upload local file to datastore", func(cli *esxi.SESXiClient, args *DatastoreDownloadOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorageByMoId(args.DSID)
		if err != nil {
			return err
		}
		ctx := context.Background()
		dsObj := ds.(*esxi.SDatastore)

		file, err := os.Open(args.LOCAL)
		if err != nil {
			return err
		}
		defer file.Close()

		err = dsObj.Upload(ctx, args.DIR, file)
		if err != nil {
			return err
		}

		return nil
	})

}
