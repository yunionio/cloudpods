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
	"io"
	"os"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ImageOptionalOptions struct {
	Public             bool     `help:"Make image public"`
	Private            bool     `help:"Make image private"`
	Format             string   `help:"Image format" choices:"raw|qcow2|iso|vmdk|docker|vhd"`
	Protected          bool     `help:"Prevent image from being deleted"`
	Unprotected        bool     `help:"Allow image to be deleted"`
	Standard           bool     `help:"Mark image as a standard image"`
	Nonstandard        bool     `help:"Mark image as a non-standard image"`
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
	OsArch             string   `help:"Os hardware architecture" choices:"x86|x86_64"`
	OsLang             string   `help:"OS Language" choices:"zh_CN|en_US"`
	Preference         int64    `help:"Disk preferences"`
	Notes              string   `help:"Notes about the image"`
	Hypervisor         []string `help:"Prefer hypervisor type" choices:"kvm|esxi|baremetal|container"`
	DiskDriver         string   `help:"Perfer disk driver" choices:"virtio|scsi|pvscsi|ide|sata"`
	NetDriver          string   `help:"Preferred network driver" choices:"virtio|e1000|vmxnet3"`
}

func addImageOptionalOptions(s *mcclient.ClientSession, params *jsonutils.JSONDict, args ImageOptionalOptions) error {
	if args.Public && !args.Private {
		params.Add(jsonutils.NewString("true"), "is-public")
	} else if !args.Public && args.Private {
		params.Add(jsonutils.NewString("false"), "is-public")
	}
	if len(args.Format) > 0 {
		params.Add(jsonutils.NewString("bare"), "container-format")
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
		projectId, e := modules.Projects.GetId(s, args.OwnerProject, nil)
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
	return nil
}

func init() {
	type ImageListOptions struct {
		options.BaseListOptions

		IsPublic   string   `help:"filter images public or not(True, False or None)" choices:"true|false"`
		IsStandard string   `help:"filter images standard or non-standard" choices:"true|false"`
		Protected  string   `help:"filter images by protected" choices:"true|false"`
		Format     []string `help:"Disk formats"`
		Name       string   `help:"Name filter"`
	}
	R(&ImageListOptions{}, "image-list", "List images", func(s *mcclient.ClientSession, args *ImageListOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		if len(args.IsPublic) > 0 {
			params.Add(jsonutils.NewString(args.IsPublic), "is_public")
		}
		if len(args.IsStandard) > 0 {
			params.Add(jsonutils.NewString(args.IsStandard), "is_standard")
		}
		if len(args.Protected) > 0 {
			params.Add(jsonutils.NewString(args.Protected), "protected")
		}
		if len(args.Tenant) > 0 {
			tid, e := modules.Projects.GetId(s, args.Tenant, nil)
			if e != nil {
				return e
			}
			params.Add(jsonutils.NewString(tid), "owner")
		}
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Format) > 0 {
			if len(args.Format) == 1 {
				params.Add(jsonutils.NewString(args.Format[0]), "disk_format")
			} else {
				fs := jsonutils.NewArray()
				for _, f := range args.Format {
					fs.Add(jsonutils.NewString(f))
				}
				params.Add(fs, "disk_formats")
			}
		}
		result, err := modules.Images.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Images.GetColumns(s))
		return nil
	})

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
		ID          string `help:"ID or Name of Image"`
		Name        string `help:"New name of the image"`
		Description string `help:"Description of image"`
		ImageOptionalOptions
	}

	R(&ImageUpdateOptions{}, "image-update", "Update images meta infomation", func(s *mcclient.ClientSession, args *ImageUpdateOptions) error {
		/* img, e := modules.Images.Get(s, args.ID, nil)
		if e != nil {
			return e
		}
		idstr, e := img.GetString("id")
		if e != nil {
			return e
		}*/
		params := jsonutils.NewDict()
		/* properties, _ := img.Get("properties")
		if properties != nil {
			params.Add(properties, "properties")
		}*/
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
		img, err := modules.Images.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})

	type ImageMembershipOptions struct {
		IMAGE   string `help:"ID or name of image to share"`
		PROJECT string `help:"ID or name of project to share image with"`
	}
	type ImageMembershipAddOptions struct {
		ImageMembershipOptions
		CanShare bool `help:"Indicating whether the project can share the image with others"`
	}
	R(&ImageMembershipAddOptions{}, "image-add-project", "Add a project to private image's membership list", func(s *mcclient.ClientSession, args *ImageMembershipAddOptions) error {
		return modules.Images.AddMembership(s, args.IMAGE, args.PROJECT, args.CanShare)
	})

	R(&ImageMembershipOptions{}, "image-remove-project", "Remove a project from private image's membership list", func(s *mcclient.ClientSession, args *ImageMembershipOptions) error {
		return modules.Images.RemoveMembership(s, args.IMAGE, args.PROJECT)
	})

	type ImageDetailOptions struct {
		ID string `help:"Image ID or name"`
	}

	type ImageDeleteOptions struct {
		ID                    []string `help:"Image ID or name"`
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

	R(&ImageDetailOptions{}, "image-list-project", "List image members", func(s *mcclient.ClientSession, args *ImageDetailOptions) error {
		imgId, e := modules.Images.GetId(s, args.ID, nil)
		if e != nil {
			return e
		}
		result, e := modules.Images.ListMemberProjects(s, imgId)
		if e != nil {
			return e
		}
		printList(result, modules.Projects.GetColumns(s))
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
		ImageOptionalOptions
	}
	R(&ImageUploadOptions{}, "image-upload", "Upload a local image", func(s *mcclient.ClientSession, args *ImageUploadOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		// if len(args.Format) == 0 {
		// 	return fmt.Errorf("Please specify image format")
		//}
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
		img, err := modules.Images.Upload(s, params, f, size)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})

	type ImageImportOptions struct {
		ImageOptionalOptions
		NAME     string `help:"Image Name"`
		COPYFROM string `help:"Image external location url"`
	}
	R(&ImageImportOptions{}, "image-import", "Import a external image", func(s *mcclient.ClientSession, args *ImageImportOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Format) == 0 {
			return fmt.Errorf("Please specify image format")
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
		if len(args.Output) > 0 {
			f, err := os.Create(args.Output)
			if err != nil {
				return err
			}
			defer f.Close()
			sink = f
		} else {
			sink = os.Stdout
		}
		meta, src, err := modules.Images.Download(s, imgId, args.Format, args.Torrent)
		if err != nil {
			return err
		}
		_, e := io.Copy(sink, src)
		if e != nil {
			return e
		}
		if len(args.Output) > 0 {
			printObject(meta)
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

	R(&ImageOperationOptions{}, "image-public", "Make a image public", func(s *mcclient.ClientSession, args *ImageOperationOptions) error {
		if len(args.ID) == 0 {
			return fmt.Errorf("No image ID provided")
		} else if len(args.ID) == 1 {
			result, err := modules.Images.PerformAction(s, args.ID[0], "public", nil)
			if err != nil {
				return err
			}
			printObject(result)
		} else {
			results := modules.Images.BatchPerformAction(s, args.ID, "public", nil)
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
			listResult := modules.ListResult{Data: arrays}
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
			projid, err := modules.Projects.GetId(s, opts.PROJECT, nil)
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

}
