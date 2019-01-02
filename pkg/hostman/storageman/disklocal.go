package storageman

import (
	"context"
	"fmt"
	"os"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/fuseutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/pkg/utils"
)

var _ALTER_SUFFIX_ = ".alter"

type SLocalDisk struct {
	SBaseDisk
	isAlter bool
}

func NewLocalDisk(storage IStorage, id string) *SLocalDisk {
	var ret = new(SLocalDisk)
	ret.SBaseDisk = *NewBaseDisk(storage, id)
	return ret
}

func (d *SLocalDisk) getPath() string {
	return path.Join(d.Storage.GetPath(), d.Id)
}

func (d *SLocalDisk) getAlterPath() string {
	return path.Join(d.Storage.GetPath(), d.Id, _ALTER_SUFFIX_)
}

func (d *SLocalDisk) GetPath() string {
	if d.isAlter {
		return d.getAlterPath()
	} else {
		return d.getPath()
	}
}

func (d *SLocalDisk) Probe() error {
	if _, err := os.Stat(d.getPath()); !os.IsNotExist(err) {
		d.isAlter = false
		return nil
	} else if _, err := os.Stat(d.getAlterPath()); !os.IsNotExist(err) {
		d.isAlter = true
		return nil
	}
	return fmt.Errorf("Disk not found")
}

func (d *SLocalDisk) Delete() error {
	dpath := d.GetPath()
	log.Infof("Delete guest disk %s", dpath)
	if err := d.Storage.DeleteDiskfile(dpath); err != nil {
		return err
	}
	// TODO: PostCreateFromImageFuse umount fuse fs
	// d.UmountImageFuse()

	/* ????????????????
	   files = os.listdir(self.storage.path)
	   for f in files:
	       if f.startswith(self.id):
	           if not re.match(r'[a-z0-9\-]*\.\d{14}', f):
	               path = os.path.join(self.storage.path, f)
	               print 'delete backing-file:', path
	               self.storage.delete_diskfile(path)
	*/

	d.Storage.RemoveDisk(d)
	return nil
}

func (d *SLocalDisk) CreateFromImageFuse(ctx context.Context, url string) error {
	log.Infof("Create from image fuse %s", url)

	var (
		localPath   = d.Storage.GetFuseTmpPath()
		mntPath     = path.Join(d.Storage.GetFuseMountPath())
		contentPath = path.Join(mntPath, "content")
		newImg, err = qemuimg.NewQemuImage(d.getPath())
	)

	if err != nil {
		log.Errorln(err)
		return err
	}

	if newImg.IsValid() && newImg.IsChained() && newImg.BackFilePath != contentPath {
		if err := newImg.Delete(); err != nil {
			log.Errorln(err)
			return err
		}
	}
	if !newImg.IsValid() || newImg.IsChained() {
		if err := fuseutils.MountFusefs(options.HostOptions.FetcherfsPath, url, localPath,
			auth.GetTokenString(), mntPath, fuseutils.DEFAULT_BLOCKSIZE); err != nil {
			log.Errorln(err)
			return err
		}
	}
	if !newImg.IsValid() {
		if err := newImg.CreateQcow2(0, false, contentPath); err != nil {
			log.Errorln(err)
			return err
		}
	}

	return nil
}

func (d *SLocalDisk) CreateFromTemplate(ctx context.Context, imageId, format string, size int64) (jsonutils.JSONObject, error) {
	var imageCacheManager = storageManager.LocalStorageImagecacheManager
	imageCache := imageCacheManager.AcquireImage(ctx, imageId, d.GetZone(), "")
	if imageCache != nil {
		defer imageCacheManager.ReleaseImage(imageId)
		cacheImagePath := imageCache.GetPath()

		if fileutils2.Exists(d.GetPath()) {
			err := os.Remove(d.GetPath())
			if err != nil {
				log.Errorln(err)
				return nil, fmt.Errorf("Fail to Create disk %s", d.Id)
			}
		}

		newImg, err := qemuimg.NewQemuImage(d.GetPath())
		if err != nil {
			log.Errorln(err)
			return nil, err
		}
		if err := newImg.CreateQcow2(int(size), false, cacheImagePath); err != nil {
			log.Errorln(err)
			return nil, fmt.Errorf("Fail to create disk %s", d.Id)
		}
		return d.GetDiskDesc(), nil

	} else {
		return nil, fmt.Errorf("Fail to fetch image %s", imageId)
	}
}

func (d *SLocalDisk) CreateRaw(ctx context.Context, sizeMB int, diskFormat, fsFormat string,
	encryption bool, diskId string, back string) (jsonutils.JSONObject, error) {
	if fileutils2.Exists(d.GetPath()) {
		os.Remove(d.GetPath())
	}

	img, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	switch diskFormat {
	case "qcow2":
		err = img.CreateQcow2(sizeMB, false, back)
	case "vmdk":
		err = img.CreateVmdk(sizeMB, false)
	default:
		err = img.CreateRaw(sizeMB)
	}

	if err != nil {
		log.Errorln(err)
		fmt.Errorf("create_raw: Fail to create disk")
	}

	if options.HostOptions.EnableFallocateDisk {
		// TODO
		// d.Fallocate
	}

	if utils.IsInStringArray(fsFormat, []string{"swap", "ext2", "ext3", "ext4", "xfs"}) {
		d.FormatFs(fsFormat, diskId)
	}

	return d.GetDiskDesc(), nil
}

func (d *SLocalDisk) FormatFs(fsFormat, diskId string) {
	log.Infof("Make disk %s fs %s", diskId, fsFormat)
	gd := NewKVMGuestDisk(d.GetPath())
	if gd.Connect() {
		defer gd.Disconnect()
		if err := gd.MakePartition(fsFormat); err == nil {
			err = gd.FormatPartition(fsFormat, diskId)
			if err != nil {
				log.Errorln(err)
			}
		} else {
			log.Errorln(err)
		}
	}
}

func (d *SLocalDisk) GetDiskDesc() jsonutils.JSONObject {
	qemuImg, err := qemuimg.NewQemuImage(d.getPath())
	if err != nil {
		log.Errorln(err)
		return nil
	}

	var desc = jsonutils.NewDict()
	desc.Set("disk_id", jsonutils.NewString(d.Id))
	desc.Set("disk_size", jsonutils.NewInt(qemuImg.SizeBytes/1024/1024))
	desc.Set("format", jsonutils.NewString(qemuImg.Format.String()))
	desc.Set("disk_path", jsonutils.NewString(d.getPath()))
	return desc
}
