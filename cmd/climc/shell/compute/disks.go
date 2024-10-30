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

package compute

import (
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cheggaaa/pb/v3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	compute_options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/sparsefile"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Disks)
	cmd.List(&compute_options.DiskListOptions{})
	cmd.Perform("set-class-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("rebuild", &compute_options.DiskRebuildOptions{})
	cmd.Perform("migrate", &compute_options.DiskMigrateOptions{})
	cmd.Perform("reset-template", &compute_options.DiskResetTemplateOptions{})

	type DiskDetailOptions struct {
		ID string `help:"ID or Name of disk"`
	}
	R(&DiskDetailOptions{}, "disk-show", "Show details of disk", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		disk, e := modules.Disks.Get(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})
	R(&DiskDetailOptions{}, "disk-cancel-delete", "Cancel pending delete disks", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		disk, e := modules.Disks.PerformAction(s, args.ID, "cancel-delete", nil)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	type DiskDeleteOptions struct {
		ID                    []string `help:"ID of disks to delete" metavar:"DISK"`
		OverridePendingDelete bool     `help:"Delete disk directly instead of pending delete" short-token:"f"`
		DeleteSnapshots       bool     `help:"Delete disk snapshots before delete disk"`
	}

	R(&DiskDeleteOptions{}, "disk-delete", "Delete a disk", func(s *mcclient.ClientSession, args *DiskDeleteOptions) error {
		params := jsonutils.NewDict()
		if args.OverridePendingDelete {
			params.Add(jsonutils.JSONTrue, "override_pending_delete")
		}
		if args.DeleteSnapshots {
			params.Add(jsonutils.JSONTrue, "delete_snapshots")
		}
		ret := modules.Disks.BatchDeleteWithParam(s, args.ID, params, nil)
		printBatchResults(ret, modules.Disks.GetColumns(s))
		return nil
	})

	type DiskBatchOpsOptions struct {
		ID []string `help:"id list of disks to operate"`
	}
	R(&DiskBatchOpsOptions{}, "disk-purge", "Delete a disk record in database, not actually do deletion", func(s *mcclient.ClientSession, args *DiskBatchOpsOptions) error {
		ret := modules.Disks.BatchPerformAction(s, args.ID, "purge", nil)
		printBatchResults(ret, modules.Disks.GetColumns(s))
		return nil
	})

	R(&DiskDetailOptions{}, "disk-public", "Make a disk public", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		disk, e := modules.Disks.PerformAction(s, args.ID, "public", nil)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	R(&DiskDetailOptions{}, "disk-private", "Make a disk private", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		disk, e := modules.Disks.PerformAction(s, args.ID, "private", nil)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	R(&DiskDetailOptions{}, "disk-metadata", "Get metadata of a disk", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		meta, e := modules.Disks.GetMetadata(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(meta)
		return nil
	})

	R(&DiskDetailOptions{}, "disk-syncstatus", "Sync status for disk", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		ret, e := modules.Disks.PerformAction(s, args.ID, "syncstatus", nil)
		if e != nil {
			return e
		}
		printObject(ret)
		return nil
	})

	type DiskUpdateOptions struct {
		ID           string `help:"ID or name of disk"`
		Name         string `help:"New name of disk"`
		Desc         string `help:"Description" metavar:"DESCRIPTION"`
		AutoDelete   string `help:"enable/disable auto delete of disk" choices:"enable|disable"`
		AutoSnapshot string `help:"enable/disable auto snapshot of disk" choices:"enable|disable"`
		DiskType     string `help:"Disk type" choices:"data|volume|sys"`
		IsSsd        *bool  `help:"mark disk as ssd" negative:"no-is-ssd"`
	}
	R(&DiskUpdateOptions{}, "disk-update", "Update property of a virtual disk", func(s *mcclient.ClientSession, args *DiskUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.AutoDelete) > 0 {
			if args.AutoDelete == "enable" {
				params.Add(jsonutils.JSONTrue, "auto_delete")
			} else {
				params.Add(jsonutils.JSONFalse, "auto_delete")
			}
		}
		if len(args.AutoSnapshot) > 0 {
			if args.AutoSnapshot == "enable" {
				params.Add(jsonutils.JSONTrue, "auto_snapshot")
			} else {
				params.Add(jsonutils.JSONFalse, "auto_snapshot")
			}
		}
		if len(args.DiskType) > 0 {
			params.Add(jsonutils.NewString(args.DiskType), "disk_type")
		}
		if args.IsSsd != nil {
			if *args.IsSsd {
				params.Add(jsonutils.JSONTrue, "is_ssd")
			} else {
				params.Add(jsonutils.JSONFalse, "is_ssd")
			}
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		disk, e := modules.Disks.Update(s, args.ID, params)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	R(&compute_options.DiskCreateOptions{}, "disk-create", "Create a virtual disk", func(s *mcclient.ClientSession, args *compute_options.DiskCreateOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		if args.TaskNotify {
			s.PrepareTask()
		}
		if args.Count > 1 {
			results := modules.Disks.BatchCreate(s, params.JSON(params), args.Count)
			printBatchResults(results, modules.Disks.GetColumns(s))
		} else {
			disk, err := modules.Disks.Create(s, params.JSON(params))
			if err != nil {
				return err
			}
			printObject(disk)
		}
		if args.TaskNotify {
			s.WaitTaskNotify()
		}
		return nil
	})

	type DiskResizeOptions struct {
		DISK string `help:"ID or name of disk"`
		SIZE string `help:"Size of disk"`
	}
	R(&DiskResizeOptions{}, "disk-resize", "Resize a disk", func(s *mcclient.ClientSession, args *DiskResizeOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.SIZE), "size")
		disk, err := modules.Disks.PerformAction(s, args.DISK, "resize", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})
	type DiskResetOptions struct {
		DISK      string `help:"ID or name of disk"`
		SNAPSHOT  string `help:"snapshots ID of disk"`
		AutoStart bool   `help:"Autostart guest"`
	}
	R(&DiskResetOptions{}, "disk-reset", "Resize a disk", func(s *mcclient.ClientSession, args *DiskResetOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.SNAPSHOT), "snapshot_id")
		if args.AutoStart {
			params.Add(jsonutils.JSONTrue, "auto_start")
		}
		disk, err := modules.Disks.PerformAction(s, args.DISK, "disk-reset", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskSaveOptions struct {
		ID     string `help:"ID or name of the disk" json:"-"`
		NAME   string `help:"Image name"`
		OSTYPE string `help:"Os type" choices:"Linux|Windows|VMware" json:"-"`
		Public *bool  `help:"Make the image public available" json:"is_public"`
		Format string `help:"image format" choices:"vmdk|qcow2"`
		Notes  string `help:"Notes about the image"`
	}
	R(&DiskSaveOptions{}, "disk-save", "Disk save image", func(s *mcclient.ClientSession, args *DiskSaveOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		params.Add(jsonutils.NewString(args.OSTYPE), "properties", "os_type")
		disk, err := modules.Disks.PerformAction(s, args.ID, "save", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskUpdateStatusOptions struct {
		ID     string `help:"ID or name of disk"`
		STATUS string `help:"Disk status" choices:"ready"`
	}
	R(&DiskUpdateStatusOptions{}, "disk-update-status", "Set disk status", func(s *mcclient.ClientSession, args *DiskUpdateStatusOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.STATUS), "status")
		disk, err := modules.Disks.PerformAction(s, args.ID, "status", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskChangeOwnerOptions struct {
		ID      string `help:"Disk to change owner" json:"-"`
		PROJECT string `help:"Project ID or change" json:"tenant"`
	}
	R(&DiskChangeOwnerOptions{}, "disk-change-owner", "Change owner porject of a disk", func(s *mcclient.ClientSession, opts *DiskChangeOwnerOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		srv, err := modules.Disks.PerformAction(s, opts.ID, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	R(&DiskDetailOptions{}, "disk-change-owner-candidate-domains", "Get change owner candidate domain list", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		result, err := modules.Disks.GetSpecific(s, args.ID, "change-owner-candidate-domains", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DiskDownloadOptions struct {
		ID       string `help:"ID or name of disk" json:"-"`
		Compress bool
		Sparse   bool

		Timeout int `help:"Timeout hours for download" default:"5"`
		Debug   bool

		FILE string
	}
	R(&DiskDownloadOptions{}, "disk-download", "Download disk from host", func(s *mcclient.ClientSession, args *DiskDownloadOptions) error {
		disk, err := modules.Disks.GetById(s, args.ID, nil)
		if err != nil {
			return err
		}
		storageId, _ := disk.GetString("storage_id")
		storage, err := modules.Storages.GetById(s, storageId, nil)
		if err != nil {
			return err
		}

		hostsInfo := []struct {
			Id string
		}{}
		storage.Unmarshal(&hostsInfo, "hosts")
		header := http.Header{}
		header.Set("X-Auth-Token", s.GetToken().GetTokenString())
		if args.Compress {
			header.Set("X-Compress-Content", "zlib")
		}
		if args.Sparse {
			header.Set("X-Sparse-Content", "true")
		}

		client := httputils.GetTimeoutClient(time.Hour * time.Duration(args.Timeout))

		for _, host := range hostsInfo {
			host, err := modules.Hosts.GetById(s, host.Id, nil)
			if err != nil {
				return err
			}
			managerUri, _ := host.GetString("manager_uri")
			if len(managerUri) == 0 {
				continue
			}
			url := fmt.Sprintf("%s/download/disks/%s/%s", managerUri, storageId, args.ID)
			resp, err := httputils.Request(client, context.Background(), httputils.GET, url, header, nil, args.Debug)
			if err != nil {
				log.Errorf("request %s error: %v", url, err)
				continue
			}
			defer resp.Body.Close()

			totalSize, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
			sparseHeader, _ := strconv.ParseInt(resp.Header.Get("X-Sparse-Header"), 10, 64)

			fi, err := os.Create(args.FILE)
			if err != nil {
				return errors.Wrapf(err, "os.Create(%s)", args.FILE)
			}
			defer fi.Close()

			var reader = resp.Body

			if args.Compress {
				zlibRC, err := zlib.NewReader(resp.Body)
				if err != nil {
					return errors.Wrapf(err, "zlib.NewReader")
				}
				defer zlibRC.Close()
				reader = zlibRC
			}

			var writer io.Writer = fi

			if sparseHeader > 0 {
				writer = sparsefile.NewSparseFileWriter(fi, sparseHeader, totalSize)
				fileSize, _ := strconv.ParseInt(resp.Header.Get("X-File-Size"), 10, 64)
				if fileSize > 0 {
					err = fi.Truncate(fileSize)
					if err != nil {
						return errors.Wrapf(err, "failed truncate file")
					}
				}
			}

			bar := pb.Full.Start64(totalSize)
			barReader := bar.NewProxyReader(reader)

			_, err = io.Copy(writer, barReader)
			return err
		}
		return fmt.Errorf("no available download url")
	})

	type SparseHoleOptions struct {
		FILE string
	}
	R(&SparseHoleOptions{}, "sparse-file-hole", "Show sparse file holes", func(s *mcclient.ClientSession, args *SparseHoleOptions) error {
		fi, err := os.Open(args.FILE)
		if err != nil {
			return err
		}
		defer fi.Close()
		sp, err := sparsefile.NewSparseFileReader(fi)
		if err != nil {
			return err
		}
		holes := sp.GetHoles()
		printObject(jsonutils.Marshal(holes))
		return nil
	})

}
