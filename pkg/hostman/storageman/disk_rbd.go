package storageman

import (
	"context"
	"fmt"

	"github.com/ceph/go-ceph/rbd"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

type SRBDDisk struct {
	SBaseDisk
}

func NewRBDDisk(storage IStorage, id string) *SRBDDisk {
	var ret = new(SRBDDisk)
	ret.SBaseDisk = *NewBaseDisk(storage, id)
	return ret
}

func (d *SRBDDisk) GetType() string {
	return storagetypes.STORAGE_RBD
}

func (d *SRBDDisk) Probe() error {
	storage := d.Storage.(*SRbdStorage)
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	_, err := storage.withImage(pool, d.Id, func(image *rbd.Image) (interface{}, error) {
		return image.GetSize()
	})
	return err
}

func (d *SRBDDisk) getPath() string {
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	return fmt.Sprintf("rbd:%s/%s", pool, d.Id)
}

func (d *SRBDDisk) GetPath() string {
	storage := d.Storage.(*SRbdStorage)
	return fmt.Sprintf("%s%s", d.getPath(), storage.getStorageConfString())
}

func (d *SRBDDisk) GetSnapshotDir() string {
	return ""
}

func (d *SRBDDisk) GetDiskDesc() jsonutils.JSONObject {
	storage := d.Storage.(*SRbdStorage)
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	desc := map[string]interface{}{
		"disk_id":     d.Id,
		"disk_format": "raw",
		"disk_path":   d.GetPath(),
		"disk_size":   storage.getImageSizeMb(pool, d.Id),
	}
	return jsonutils.Marshal(desc)
}

func (d *SRBDDisk) GetDiskSetupScripts(idx int) string {
	return fmt.Sprintf("DISK_%d=%s\n", idx, d.GetPath())
}

func (d *SRBDDisk) DeleteAllSnapshot() error {
	return fmt.Errorf("Not Impl")
}

func (d *SRBDDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	storage := d.Storage.(*SRbdStorage)
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	return nil, storage.deleteImage(pool, d.Id)
}

func (d *SRBDDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	diskInfo, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}
	storage := d.Storage.(*SRbdStorage)
	storageConf := d.Storage.GetStorageConf()
	pool, _ := storageConf.GetString("pool")
	sizeMb, _ := diskInfo.Int("size")
	if err := storage.resizeImage(pool, d.Id, uint64(sizeMb)); err != nil {
		return nil, err
	}

	if err := d.ResizeFs(); err != nil {
		return nil, err
	}

	return d.GetDiskDesc(), nil
}

func (d *SRBDDisk) ResizeFs() error {
	disk := NewKVMGuestDisk(d.GetPath())
	if disk.Connect() {
		defer disk.Disconnect()
		if err := disk.ResizePartition(); err != nil {
			return err
		}
	}
	return nil
}

func (d *SRBDDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	if err := d.Probe(); err != nil {
		return nil, err
	}
	imageName := fmt.Sprintf("image_cache_%s_%s", d.Id, appctx.AppContextTaskId(ctx))
	imageCache := storageManager.GetStoragecacheById(d.Storage.GetStoragecacheId())
	if imageCache == nil {
		return nil, fmt.Errorf("failed to find image cache for prepare save to glance")
	}
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.GetStorageConf().GetString("pool")
	if err := storage.copyImage(pool, d.Id, imageCache.GetPath(), imageName); err != nil {
		return nil, err
	}
	return jsonutils.Marshal(map[string]string{"backup": imageName}), nil
}

func (d *SRBDDisk) ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not impl")
}

func (d *SRBDDisk) CleanupSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	return nil, storage.deleteSnapshot(pool, d.Id, "")
}

func (d *SRBDDisk) PrepareMigrate(liveMigrate bool) (string, error) {
	return "", fmt.Errorf("Not support")
}

func (d *SRBDDisk) CreateFromUrl(context.Context, string) error {
	return fmt.Errorf("Not impl")
}

func (d *SRBDDisk) CreateFromTemplate(ctx context.Context, imageId string, format string, size int64) (jsonutils.JSONObject, error) {
	ret, err := d.createFromTemplate(ctx, imageId, format)
	if err != nil {
		return nil, err
	}

	retSize, _ := ret.Int("disk_size")
	log.Infof("REQSIZE: %d, RETSIZE: %d", size, retSize)
	if size > retSize {
		params := jsonutils.NewDict()
		params.Set("size", jsonutils.NewInt(size))
		return d.Resize(ctx, params)
	}

	return ret, nil
}

func (d *SRBDDisk) createFromTemplate(ctx context.Context, imageId, format string) (jsonutils.JSONObject, error) {
	var imageCacheManager = storageManager.GetStoragecacheById(d.Storage.GetStoragecacheId())
	if imageCacheManager == nil {
		return nil, fmt.Errorf("failed to find image cache manger for storage %s", d.Storage.GetStorageName())
	}
	imageCache := imageCacheManager.AcquireImage(ctx, imageId, d.GetZone(), "", "")
	if imageCache == nil {
		return nil, fmt.Errorf("failed to qcquire image for storage %s", d.Storage.GetStorageName())
	}
	defer imageCacheManager.ReleaseImage(imageId)
	storage := d.Storage.(*SRbdStorage)
	destPool, _ := storage.StorageConf.GetString("pool")
	if err := storage.copyImage(imageCacheManager.GetPath(), imageCache.GetName(), destPool, d.Id); err != nil {
		return nil, err
	}
	return d.GetDiskDesc(), nil
}

func (d *SRBDDisk) CreateFromImageFuse(context.Context, string) error {
	return fmt.Errorf("Not support")
}

func (d *SRBDDisk) CreateRaw(ctx context.Context, sizeMb int, diskFromat string, fsFormat string, encryption bool, diskId string, back string) (jsonutils.JSONObject, error) {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	if err := storage.createImage(pool, diskId, uint64(sizeMb)); err != nil {
		return nil, err
	}
	return d.GetDiskDesc(), nil
}

func (d *SRBDDisk) PostCreateFromImageFuse() {
	log.Errorf("Not support PostCreateFromImageFuse")
}

func (d *SRBDDisk) CreateSnapshot(snapshotId string) error {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	return storage.createSnapshot(pool, d.Id, snapshotId)
}

func (d *SRBDDisk) DeleteSnapshot(snapshotId, convertSnapshot string, pendingDelete bool) error {
	storage := d.Storage.(*SRbdStorage)
	pool, _ := storage.StorageConf.GetString("pool")
	return storage.deleteSnapshot(pool, d.Id, snapshotId)
}
