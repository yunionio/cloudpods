package shell

import (
	"fmt"
	"io"
	"os"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ImageOptionalOptions struct {
	Public             bool     `help:"Make image public"`
	Private            bool     `help:"Make image private"`
	Format             string   `help:"Image format" choices:"raw|qcow2|iso|vmdk|docker|vhd"`
	Protected          bool     `help:"Prevent image from being deleted"`
	Unprotected        bool     `help:"Allow image to be deleted"`
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
		Tenant        string   `help:"Tenant id of image owner"`
		IsPublic      string   `help:"filter images public or not(True, False or None)" choices:"true|false|none"`
		Admin         bool     `help:"Show images of all tenants, ADMIN only"`
		PendingDelete bool     `help:"Show pending deleted images"`
		Limit         int64    `help:"Max items show, 0 means no limit"`
		Offset        int64    `help:"Pagination offset, default 0"`
		Format        []string `help:"Disk formats"`
		Search        string   `help:"Search text"`
		Marker        string   `help:"The last Image ID of the previous page"`
		Name          string   `help:"Name filter"`
		Details       bool     `help:"Show details"`
	}
	R(&ImageListOptions{}, "image-list", "List images", func(s *mcclient.ClientSession, args *ImageListOptions) error {
		params := jsonutils.NewDict()
		if len(args.IsPublic) > 0 {
			params.Add(jsonutils.NewString(args.IsPublic), "is_public")
		}
		if args.PendingDelete {
			params.Add(jsonutils.JSONTrue, "pending_delete")
		}
		if args.Admin {
			params.Add(jsonutils.JSONTrue, "admin")
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
		if len(args.Marker) > 0 {
			params.Add(jsonutils.NewString(args.Marker), "marker")
		}
		if args.Limit > 0 {
			params.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			params.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		if len(args.Search) > 0 {
			params.Add(jsonutils.NewString(args.Search), "search")
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
		if args.Details {
			params.Add(jsonutils.JSONTrue, "details")
		} else {
			params.Add(jsonutils.JSONFalse, "details")
		}
		result, err := modules.Images.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Images.GetColumns(s))
		return nil
	})

	type ImageShowOptions struct {
		ID []string `help:"Image id or name" metavar:"IMAGE"`
	}
	R(&ImageShowOptions{}, "image-show", "Show details of a image", func(s *mcclient.ClientSession, args *ImageShowOptions) error {
		if len(args.ID) == 0 {
			return fmt.Errorf("No image ID provided")
		} else if len(args.ID) == 1 {
			result, e := modules.Images.Get(s, args.ID[0], nil)
			if e != nil {
				return e
			}
			printObject(result)
		} else {
			sr := modules.Images.BatchGet(s, args.ID, nil)
			printBatchResults(sr, modules.Images.GetColumns(s))
		}
		return nil
	})

	type ImageUpdateOptions struct {
		ID   string `help:"ID or Name of Image"`
		Name string `help:"New name of the image"`
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

	R(&ImageDetailOptions{}, "image-delete", "Delete a image", func(s *mcclient.ClientSession, args *ImageDetailOptions) error {
		imgID, err := modules.Images.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		if result, err := modules.Images.Delete(s, imgID, nil); err != nil {
			return err
		} else {
			printObject(result)
		}
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
		if len(args.Format) == 0 {
			return fmt.Errorf("Please specify image format")
		}
		if len(args.OsType) == 0 {
			return fmt.Errorf("Please specify OS type")
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
		ID     string `help:"ID or Name of image"`
		Output string `help:"Destination file, if omitted, output to stdout"`
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
		meta, src, err := modules.Images.Download(s, imgId)
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

}
