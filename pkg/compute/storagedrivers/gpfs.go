package storagedrivers

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/timeutils"
)

type SGpfsStorageDriver struct {
	SBaseStorageDriver
}

func init() {
	driver := SGpfsStorageDriver{}
	models.RegisterStorageDriver(&driver)
}

func (self *SGpfsStorageDriver) GetStorageType() string {
	return api.STORAGE_GPFS
}

func (self *SGpfsStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SGpfsStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {
	sc := &models.SStoragecache{}
	sc.Path = options.Options.DefaultImageCacheDir
	sc.ExternalId = storage.Id
	timeutils.IsoTime(time.Now())
	sc.Name = "gpfs-" + storage.Name + timeutils.IsoTime(time.Now())
	if err := models.StoragecacheManager.TableSpec().Insert(sc); err != nil {
		log.Errorf("insert storagecache for storage %s error: %v", storage.Name, err)
		return
	}
	_, err := db.Update(storage, func() error {
		storage.StoragecacheId = sc.Id
		storage.Status = api.STORAGE_ONLINE
		return nil
	})
	if err != nil {
		log.Errorf("update storagecache info for storage %s error: %v", storage.Name, err)
	}
}
