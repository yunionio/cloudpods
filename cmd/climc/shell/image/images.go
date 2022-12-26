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

package image

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cheggaaa/pb/v3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/glance"
)

type ImageOptionalOptions struct {
	Format             string   `help:"Image format" choices:"raw|qcow2|iso|vmdk|docker|vhd"`
	Protected          bool     `help:"Prevent image from being deleted"`
	Unprotected        bool     `help:"Allow image to be deleted"`
	Standard           bool     `help:"Mark image as a standard image"`
	Nonstandard        bool     `help:"Mark image as a non-standard image"`
	Public             bool     `help:"make image public"`
	MinDisk            int64    `help:"Disk size after expanded, in MB" metavar:"MIN_DISK_SIZE_MB"`
	MinRam             int64    `help:"Minimal memory size required" metavar:"MIN_RAM_MB"`
	VirtualSize        int64    `help:"Disk size after expanded, in MB"`
	Size               int64    `help:"Disk size, in MB"`
	Location           string   `help:"Image location"`
	Status             string   `help:"Image status" choices:"killed|active|queued"`
	OwnerProject       string   `help:"Owner project Id or Name"`
	OwnerProjectDomain string   `help:"Owner project Domain"`
	OsType             string   `help:"Type of OS" choices:"Windows|Linux|Freebsd|Android|macOS|VMWare"`
	OsDist             string   `help:"Distribution name of OS" metavar:"OS_DISTRIBUTION"`
	OsVersion          string   `help:"Version of OS"`
	OsCodename         string   `help:"Codename of OS"`
	OsArch             string   `help:"Os hardware architecture" choices:"x86|x86_64|aarch32|aarch64"`
	OsLang             string   `help:"OS Language" choices:"zh_CN|en_US"`
	Preference         int64    `help:"Disk preferences"`
	Notes              string   `help:"Notes about the image"`
	Hypervisor         []string `help:"Prefer hypervisor type" choices:"kvm|esxi|baremetal|container|openstack|ctyun"`
	DiskDriver         string   `help:"Perfer disk driver" choices:"virtio|scsi|pvscsi|ide|sata"`
	NetDriver          string   `help:"Preferred network driver" choices:"virtio|e1000|vmxnet3"`
	DisableUsbKbd      bool     `help:"Disable usb keyboard on this image(for hypervisor kvm)"`
	BootMode           string   `help:"UEFI support" choices:"UEFI|BIOS"`
	VdiProtocol        string   `help:"VDI protocol" choices:"vnc|spice"`
}

func addImageOptionalOptions(s *mcclient.ClientSession, params *jsonutils.JSONDict, args ImageOptionalOptions) error {
	if len(args.Format) > 0 {
		params.Add(jsonutils.NewString(args.Format), "disk-format")
	}
	if args.Protected && !args.Unprotected {
		params.Add(jsonutils.NewString("true"), "protected")
	} else if !args.Protected && args.Unprotected {
		params.Add(jsonutils.NewString("false"), "protected")
	}
	if args.Standard && !args.Nonstandard {
		params.Add(jsonutils.JSONTrue, "is_standard")
	} else if !args.Standard && args.Nonstandard {
		params.Add(jsonutils.JSONFalse, "is_standard")
	}
	if args.Public {
		params.Add(jsonutils.JSONTrue, "is_public")
	}
	if args.MinDisk > 0 {
		params.Add(jsonutils.NewString(fmt.Sprintf("%d", args.MinDisk)), "min_disk")
	}
	if args.MinRam > 0 {
		params.Add(jsonutils.NewString(fmt.Sprintf("%d", args.MinRam)), "min_ram")
	}
	if args.Size > 0 {
		params.Add(jsonutils.NewString(fmt.Sprintf("%d", args.Size*1024*1024)), "size")
	}
	if args.VirtualSize > 0 {
		params.Add(jsonutils.NewString(fmt.Sprintf("%d", args.VirtualSize)), "min_disk")
	}
	if len(args.Location) > 0 {
		params.Add(jsonutils.NewString(args.Location), "location")
	}
	if len(args.Status) > 0 {
		params.Add(jsonutils.NewString(args.Status), "status")
	}
	if len(args.OwnerProject) > 0 {
		projectId, e := identity.Projects.GetId(s, args.OwnerProject, nil)
		if e != nil {
			return e
		}
		params.Add(jsonutils.NewString(projectId), "owner")
	}
	if len(args.OsType) > 0 {
		params.Add(jsonutils.NewString(args.OsType), "properties", "os_type")
	}
	if len(args.OsDist) > 0 {
		params.Add(jsonutils.NewString(args.OsDist), "properties", "os_distribution")
	}
	if len(args.OsVersion) > 0 {
		params.Add(jsonutils.NewString(args.OsVersion), "properties", "os_version")
	}
	if len(args.OsCodename) > 0 {
		params.Add(jsonutils.NewString(args.OsCodename), "properties", "os_codename")
	}
	if len(args.OsArch) > 0 {
		params.Add(jsonutils.NewString(args.OsArch), "properties", "os_arch")
		params.Add(jsonutils.NewString(args.OsArch), "os_arch")
	}
	if len(args.OsLang) > 0 {
		params.Add(jsonutils.NewString(args.OsLang), "properties", "os_language")
	}
	if args.Preference > 0 {
		params.Add(jsonutils.NewString(fmt.Sprintf("%d", args.Preference)), "properties", "preference")
	}
	if len(args.Notes) > 0 {
		params.Add(jsonutils.NewString(args.Notes), "properties", "notes")
	}
	if len(args.DiskDriver) > 0 {
		params.Add(jsonutils.NewString(args.DiskDriver), "properties", "disk_driver")
	}
	if len(args.NetDriver) > 0 {
		params.Add(jsonutils.NewString(args.NetDriver), "properties", "net_driver")
	}
	if len(args.Hypervisor) > 0 {
		params.Add(jsonutils.NewString(strings.Join(args.Hypervisor, ",")), "properties", "hypervisor")
	}
	if args.DisableUsbKbd {
		params.Add(jsonutils.NewString("true"), "properties", "disable_usb_kbd")
	}
	if args.BootMode == "UEFI" {
		params.Add(jsonutils.JSONTrue, "properties", "uefi_support")
	} else if args.BootMode == "BIOS" {
		params.Add(jsonutils.JSONFalse, "properties", "uefi_support")
	}
	if len(args.VdiProtocol) > 0 {
		params.Add(jsonutils.NewString(args.VdiProtocol), "properties", "vdi_protocol")
	}
	return nil
}

func init() {

	cmd := shell.NewResourceCmd(&modules.Images)
	cmd.List(&glance.ImageListOptions{})
	cmd.GetProperty(&glance.ImageStatusStatisticsOptions{})
	cmd.Perform("user-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("set-user-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("class-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("set-class-metadata", &options.ResourceMetadataOptions{})

	type ImageOperationOptions struct {
		ID []string `help:"Image id or name" metavar:"IMAGE"`
	}

	type ImageShowOptions struct {
		ID      []string `help:"Image id or name" metavar:"IMAGE"`
		Format  string   `help:"Image format"`
		Torrent bool     `help:"show torrent information"`
	}
	R(&ImageShowOptions{}, "image-show", "Show details of a image", func(s *mcclient.ClientSession, args *ImageShowOptions) error {
		params := jsonutils.NewDict()
		if len(args.Format) > 0 {
			params.Add(jsonutils.NewString(args.Format), "format")
			if args.Torrent {
				params.Add(jsonutils.JSONTrue, "torrent")
			}
		}
		if len(args.ID) == 0 {
			return fmt.Errorf("No image ID provided")
		} else if len(args.ID) == 1 {
			result, e := modules.Images.Get(s, args.ID[0], params)
			if e != nil {
				return e
			}
			printObject(result)
		} else {
			sr := modules.Images.BatchGet(s, args.ID, params)
			printBatchResults(sr, modules.Images.GetColumns(s))
		}
		return nil
	})

	type ImageUpdateOptions struct {
		ID          []string `help:"ID or Name of Image"`
		Name        string   `help:"New name of the image"`
		Description string   `help:"Description of image"`
		ImageOptionalOptions
	}

	R(&ImageUpdateOptions{}, "image-update", "Update images meta infomation", func(s *mcclient.ClientSession, args *ImageUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Description) > 0 {
			params.Add(jsonutils.NewString(args.Description), "description")
		}
		err := addImageOptionalOptions(s, params, args.ImageOptionalOptions)
		if err != nil {
			return err
		}
		result := modules.Images.BatchUpdate(s, args.ID, params)
		printBatchResults(result, modules.Images.GetColumns(s))
		return nil
	})

	type ImageDetailOptions struct {
		ID string `help:"Image ID or name"`
	}

	type ImageDeleteOptions struct {
		ID                    []string `help:"Image ID or name" json:"-"`
		OverridePendingDelete *bool    `help:"Delete image directly instead of pending delete" short-token:"f"`
	}
	R(&ImageDeleteOptions{}, "image-delete", "Delete a image", func(s *mcclient.ClientSession, args *ImageDeleteOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		ret := modules.Images.BatchDeleteWithParam(s, args.ID, params, nil)
		printBatchResults(ret, modules.Images.GetColumns(s))
		return nil
	})

	type ImageCancelDeleteOptions struct {
		ID string `help:"Image id or name" metavar:"IMAGE"`
	}
	R(&ImageCancelDeleteOptions{}, "image-cancel-delete", "Cancel pending delete images", func(s *mcclient.ClientSession, args *ImageCancelDeleteOptions) error {
		if image, e := modules.Images.PerformAction(s, args.ID, "cancel-delete", nil); e != nil {
			return e
		} else {
			printObject(image)
		}
		return nil
	})

	type ImageUploadOptions struct {
		NAME string `help:"Image Name"`
		FILE string `help:"The local image filename to Upload"`

		EncryptKey string `help:"encrypt key id"`

		ImageOptionalOptions
	}
	R(&ImageUploadOptions{}, "image-upload", "Upload a local image", func(s *mcclient.ClientSession, args *ImageUploadOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.EncryptKey) > 0 {
			params.Add(jsonutils.NewString(args.EncryptKey), "encrypt_key_id")
		}
		err := addImageOptionalOptions(s, params, args.ImageOptionalOptions)
		if err != nil {
			return err
		}
		f, err := os.Open(args.FILE)
		if err != nil {
			return err
		}
		defer f.Close()
		finfo, err := f.Stat()
		if err != nil {
			return err
		}
		size := finfo.Size()
		bar := pb.Full.Start64(size)
		barReader := bar.NewProxyReader(f)
		img, err := modules.Images.Upload(s, params, barReader, size)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})

	type ImageImportOptions struct {
		ImageOptionalOptions
		NAME       string `help:"Image Name"`
		COPYFROM   string `help:"Image external location url"`
		EncryptKey string `help:"encrypt key id"`
	}
	R(&ImageImportOptions{}, "image-import", "Import a external image", func(s *mcclient.ClientSession, args *ImageImportOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Format) == 0 {
			return fmt.Errorf("Please specify image format")
		}
		if len(args.EncryptKey) > 0 {
			params.Add(jsonutils.NewString(args.EncryptKey), "encrypt_key_id")
		}
		err := addImageOptionalOptions(s, params, args.ImageOptionalOptions)
		if err != nil {
			return err
		}

		params.Add(jsonutils.NewString(args.COPYFROM), "copy_from")
		img, err := modules.Images.Create(s, params)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})

	type ImageDownloadOptions struct {
		ID      string `help:"ID or Name of image"`
		Output  string `help:"Destination file, if omitted, output to stdout"`
		Format  string `help:"Image format"`
		Torrent bool   `help:"show torrent information"`
	}
	R(&ImageDownloadOptions{}, "image-download", "Download image data to a file or stdout", func(s *mcclient.ClientSession, args *ImageDownloadOptions) error {
		imgId, err := modules.Images.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		var sink io.Writer
		showProgress := false
		if len(args.Output) > 0 {
			f, err := os.Create(args.Output)
			if err != nil {
				return err
			}
			defer f.Close()
			sink = f
			showProgress = true
		} else {
			sink = os.Stdout
		}
		meta, src, size, err := modules.Images.Download(s, imgId, args.Format, args.Torrent)
		if err != nil {
			return err
		}
		if !showProgress {
			_, err = io.Copy(sink, src)
			if err != nil {
				return err
			}
		} else {
			bar := pb.Full.Start64(size)
			barReader := bar.NewProxyReader(src)
			_, err = io.Copy(sink, barReader)
			if err != nil {
				return err
			}
		}
		if len(args.Output) > 0 {
			printObject(meta)
			fmt.Println("Image size: ", size)
		}
		return nil
	})

	R(&ImageOperationOptions{}, "image-private", "Make a image private", func(s *mcclient.ClientSession, args *ImageOperationOptions) error {
		if len(args.ID) == 0 {
			return fmt.Errorf("No image ID provided")
		} else if len(args.ID) == 1 {
			result, err := modules.Images.PerformAction(s, args.ID[0], "private", nil)
			if err != nil {
				return err
			}
			printObject(result)
		} else {
			results := modules.Images.BatchPerformAction(s, args.ID, "private", nil)
			printBatchResults(results, modules.Images.GetColumns(s))
		}
		return nil
	})

	type ImagePublicOptions struct {
		ID             []string `help:"ID or name of image" json:"-"`
		Scope          string   `help:"sharing scope" choices:"system|domain|project"`
		SharedProjects []string `help:"Share to projects"`
		SharedDomains  []string `help:"Share to domains"`
	}
	R(&ImagePublicOptions{}, "image-public", "Make a image public", func(s *mcclient.ClientSession, args *ImagePublicOptions) error {
		params := jsonutils.Marshal(args)
		if len(args.ID) == 0 {
			return fmt.Errorf("No image ID provided")
		} else if len(args.ID) == 1 {
			result, err := modules.Images.PerformAction(s, args.ID[0], "public", params)
			if err != nil {
				return err
			}
			printObject(result)
		} else {
			results := modules.Images.BatchPerformAction(s, args.ID, "public", params)
			printBatchResults(results, modules.Images.GetColumns(s))
		}
		return nil
	})

	R(&ImageOperationOptions{}, "image-subformats", "Show all format status of a image", func(s *mcclient.ClientSession, args *ImageOperationOptions) error {
		for i := range args.ID {
			result, err := modules.Images.GetSpecific(s, args.ID[i], "subformats", nil)
			if err != nil {
				fmt.Println("Fail to fetch subformats for", args.ID[i])
				continue
			}
			arrays, _ := result.(*jsonutils.JSONArray).GetArray()
			listResult := printutils.ListResult{Data: arrays}
			printList(&listResult, []string{})
		}
		return nil
	})

	R(&ImageOperationOptions{}, "image-mark-standard", "Mark image standard", func(s *mcclient.ClientSession, args *ImageOperationOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.JSONTrue, "is_standard")
		results := modules.Images.BatchPerformAction(s, args.ID, "mark-standard", params)
		printBatchResults(results, modules.Images.GetColumns(s))
		return nil
	})

	R(&ImageOperationOptions{}, "image-mark-unstandard", "Mark image not standard", func(s *mcclient.ClientSession, args *ImageOperationOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.JSONFalse, "is_standard")
		results := modules.Images.BatchPerformAction(s, args.ID, "mark-standard", params)
		printBatchResults(results, modules.Images.GetColumns(s))
		return nil
	})

	type ImageChangeOwnerOptions struct {
		ID      string `help:"Image to change owner"`
		PROJECT string `help:"Project ID or change"`
		RawId   bool   `help:"User raw ID, instead of name"`
	}
	R(&ImageChangeOwnerOptions{}, "image-change-owner", "Change owner project of an image", func(s *mcclient.ClientSession, opts *ImageChangeOwnerOptions) error {
		params := jsonutils.NewDict()
		if opts.RawId {
			projid, err := identity.Projects.GetId(s, opts.PROJECT, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(projid), "tenant")
			params.Add(jsonutils.JSONTrue, "raw_id")
		} else {
			params.Add(jsonutils.NewString(opts.PROJECT), "tenant")
		}
		srv, err := modules.Images.PerformAction(s, opts.ID, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ImageProbeOptions struct {
		ID string `help:"ID or name of image to probe"`
	}
	R(&ImageProbeOptions{}, "image-probe", "Start image probe task", func(s *mcclient.ClientSession, opts *ImageProbeOptions) error {
		img, err := modules.Images.PerformAction(s, opts.ID, "probe", nil)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})
}
