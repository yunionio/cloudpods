package storagedrivers

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNfsStorageDriver struct {
	SBaseStorageDriver
}

func init() {
	driver := SNfsStorageDriver{}
	models.RegisterStorageDriver(&driver)
}

func (self *SNfsStorageDriver) GetStorageType() string {
	return models.STORAGE_NFS
}

func (self *SNfsStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	conf := jsonutils.NewDict()
	for _, v := range []string{"nfs_host", "nfs_shared_dir"} {
		value, _ := data.GetString(v)
		if len(value) == 0 {
			return nil, httperrors.NewMissingParameterError(v)
		}
		conf.Set(v, jsonutils.NewString(value))
	}

	data.Set("storage_conf", conf)

	return data, nil
}

func (self *SNfsStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {
	sc := &models.SStoragecache{}
	sc.Path = options.Options.NfsDefaultImageCacheDir
	sc.ExternalId = storage.Id
	sc.Name = "nfs-" + storage.Name + time.Now().Format("2006-01-02 15:04:05")
	if err := models.StoragecacheManager.TableSpec().Insert(sc); err != nil {
		log.Errorf("insert storagecache for storage %s error: %v", storage.Name, err)
		return
	}
	_, err := storage.GetModelManager().TableSpec().Update(storage, func() error {
		storage.StoragecacheId = sc.Id
		storage.Status = models.STORAGE_ONLINE
		return nil
	})
	if err != nil {
		log.Errorf("update storagecache info for storage %s error: %v", storage.Name, err)
	}
}
