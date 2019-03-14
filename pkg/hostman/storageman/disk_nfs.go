package storageman

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts/storagetypes"
)

type SNFSDisk struct {
	SLocalDisk
}

func NewNFSDisk(storage IStorage, id string) *SNFSDisk {
	return &SNFSDisk{
		SLocalDisk: *NewLocalDisk(storage, id),
	}
}

func (d *SNFSDisk) GetType() string {
	return storagetypes.STORAGE_NFS
}

func (d *SNFSDisk) CreateFromTemplate(ctx context.Context, imageId, format string, size int64) (jsonutils.JSONObject, error) {
	imageCacheManager := storageManager.GetStoragecacheById(d.Storage.GetStoragecacheId())
	ret, err := d.SLocalDisk.createFromTemplate(ctx, imageId, format, imageCacheManager)
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
