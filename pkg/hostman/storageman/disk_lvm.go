package storageman

import (
	"context"
	"fmt"
	"path"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SLVMDisk struct {
	SBaseDisk
}

func (d *SLVMDisk) GetSnapshotDir() string {
	return ""
}

// /dev/<vg>/<lvm>
func (d *SLVMDisk) GetPath() string {
	return path.Join("/dev", d.Storage.GetPath(), d.Id)
}

func (d *SLVMDisk) GetDiskSetupScripts(idx int) string {
	return fmt.Sprintf("DISK_%d='%s'\n", idx, d.GetPath())
}

func (d *SLVMDisk) GetDiskDesc() jsonutils.JSONObject {
	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return nil
	}

	var desc = jsonutils.NewDict()
	desc.Set("disk_id", jsonutils.NewString(d.Id))
	desc.Set("disk_size", jsonutils.NewInt(qemuImg.SizeBytes/1024/1024))
	desc.Set("format", jsonutils.NewString(string(qemuimgfmt.RAW)))
	desc.Set("disk_path", jsonutils.NewString(d.Storage.GetPath()))
	return desc
}

func (d *SLVMDisk) CreateRaw(
	ctx context.Context, sizeMb int, diskFormat string, fsFormat string,
	encryptInfo *apis.SEncryptInfo, diskId string, back string,
) (jsonutils.JSONObject, error) {
	if fileutils2.Exists(d.GetPath()) {
		if err := d.removeLVM(); err != nil {
			return nil, errors.Wrap(err, "failed remove exists lvm")
		}
	}

	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		"lvm", "lvcreate", "--size", fmt.Sprintf("%dM", sizeMb), "-n", d.Id, d.Storage.GetPath(), "-y",
	).Output()
	if err != nil {
		return nil, errors.Wrap(err, string(out))
	}

	diskInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if utils.IsInStringArray(fsFormat, []string{"swap", "ext2", "ext3", "ext4", "xfs"}) {
		d.FormatFs(fsFormat, diskId, diskInfo)
	}
	return d.GetDiskDesc(), nil
}

func (d *SLVMDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("lvm", "lvremove", d.GetPath(), "-y").Output()
	if err != nil {
		return nil, errors.Wrap(err, string(out))
	}
	d.Storage.RemoveDisk(d)
	return nil, nil
}

func (d *SLVMDisk) removeLVM() error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("lvm", "lvremove", d.GetPath(), "-y").Output()
	if err != nil {
		return errors.Wrap(err, string(out))
	}
	return nil
}

func (d *SLVMDisk) PostCreateFromImageFuse() {
}

func (d *SLVMDisk) IsFile() bool {
	return false
}

func (d *SLVMDisk) Probe() error {
	if !fileutils2.Exists(d.GetPath()) {
		return errors.Wrapf(cloudprovider.ErrNotFound, "%s", d.GetPath())
	}
	return nil
}

func (d *SLVMDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskInfo, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}
	sizeMb, _ := diskInfo.Int("size")
	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	if qemuImg.SizeBytes/1024/1024 < sizeMb {
		out, err := procutils.NewRemoteCommandAsFarAsPossible(
			"lvm", "lvresize", "--size", fmt.Sprintf("%dM", sizeMb), d.GetPath(), "-y",
		).Output()
		if err != nil {
			return nil, errors.Wrap(err, string(out))
		}
	}

	resizeFsInfo := &deployapi.DiskInfo{
		Path: d.GetPath(),
	}
	if err := d.ResizeFs(resizeFsInfo); err != nil {
		log.Errorf("Resize fs %s fail %s", d.GetPath(), err)
	}
	return d.GetDiskDesc(), nil
}

func (d *SLVMDisk) CreateFromTemplate(ctx context.Context, imageId, format string, size int64, encryptInfo *apis.SEncryptInfo) (jsonutils.JSONObject, error) {
	var imageCacheManager = storageManager.LocalStorageImagecacheManager
	ret, err := d.createFromTemplate(ctx, imageId, format, imageCacheManager, encryptInfo)
	if err != nil {
		return nil, err
	}
	retSize, _ := ret.Int("disk_size")
	log.Infof("REQSIZE: %d, RETSIZE: %d", size, retSize)
	if size > retSize {
		params := jsonutils.NewDict()
		params.Set("size", jsonutils.NewInt(size))
		if encryptInfo != nil {
			params.Set("encrypt_info", jsonutils.Marshal(encryptInfo))
		}
		return d.Resize(ctx, params)
	}
	return ret, nil
}

func (d *SLVMDisk) createFromTemplate(
	ctx context.Context, imageId, format string, imageCacheManager IImageCacheManger, encryptInfo *apis.SEncryptInfo,
) (jsonutils.JSONObject, error) {
	input := api.CacheImageInput{
		ImageId: imageId,
		Zone:    d.GetZoneId(),
	}
	imageCache, err := imageCacheManager.AcquireImage(ctx, input, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "AcquireImage")
	}

	defer imageCacheManager.ReleaseImage(ctx, imageId)
	cacheImagePath := imageCache.GetPath()

	if fileutils2.Exists(d.GetPath()) {
		if err := d.removeLVM(); err != nil {
			return nil, errors.Wrap(err, "failed remove exists lvm")
		}
	}
	cacheImage, err := qemuimg.NewQemuImage(cacheImagePath)
	if err != nil {
		return nil, errors.Wrapf(err, "NewQemuImage(%s)", cacheImagePath)
	}

	srcInfo := qemuimg.SImageInfo{
		Path:     cacheImagePath,
		Format:   cacheImage.Format,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	destInfo := qemuimg.SImageInfo{
		Path:     d.GetPath(),
		Format:   qemuimgfmt.RAW,
		IoLevel:  qemuimg.IONiceNone,
		Password: "",
	}
	err = qemuimg.Convert(srcInfo, destInfo, false, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "qemuimg.Convert from %s to %s", cacheImagePath, d.GetPath())
	}
	return d.GetDiskDesc(), nil
}

func NewLVMDisk(storage IStorage, id string) *SLVMDisk {
	return &SLVMDisk{
		SBaseDisk: *NewBaseDisk(storage, id),
	}
}
