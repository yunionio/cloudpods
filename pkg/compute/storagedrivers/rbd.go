package storagedrivers

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SRbdStorageDriver struct {
	SBaseStorageDriver
}

func init() {
	driver := SRbdStorageDriver{}
	models.RegisterStorageDriver(&driver)
}

func (self *SRbdStorageDriver) GetStorageType() string {
	return models.STORAGE_RBD
}

func (self *SRbdStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	conf := jsonutils.NewDict()
	for _, v := range []string{"rbd_mon_host", "rbd_pool"} {
		if !data.Contains(v) {
			return nil, httperrors.NewMissingParameterError(v)
		}
		value, _ := data.GetString(v)
		conf.Add(jsonutils.NewString(value), strings.TrimPrefix(v, "rbd_"))
	}
	if key, _ := data.GetString("rbd_key"); len(key) > 0 {
		conf.Add(jsonutils.NewString(key), "key")
	}

	if timeout, _ := data.Int("rbd_timeout"); timeout > 0 {
		conf.Add(jsonutils.NewInt(timeout), "rados_osd_op_timeout")
		conf.Add(jsonutils.NewInt(timeout), "rados_mon_op_timeout")
		conf.Add(jsonutils.NewInt(timeout), "client_mount_timeout")
	}

	storages := []models.SStorage{}
	q := models.StorageManager.Query().Equals("storage_type", models.STORAGE_RBD)
	if err := db.FetchModelObjects(models.StorageManager, q, &storages); err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	inputHost, _ := conf.GetString("mon_host")
	inputPool, _ := conf.GetString("pool")
	for i := 0; i < len(storages); i++ {
		host, _ := storages[i].StorageConf.GetString("mon_host")
		pool, _ := storages[i].StorageConf.GetString("pool")
		if inputHost == host && inputPool == pool {
			return nil, httperrors.NewDuplicateResourceError("This RBD Storage[%s/%s] has already exist", storages[i].Name, inputPool)
		}
	}

	data.Set("storage_conf", conf)

	return data, nil
}

func (self *SRbdStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {
	storages := []models.SStorage{}
	q := models.StorageManager.Query().Equals("storage_type", models.STORAGE_RBD)
	if err := db.FetchModelObjects(models.StorageManager, q, &storages); err != nil {
		log.Errorf("fetch storages error: %v", err)
		return
	}
	newRbdHost, _ := data.GetString("rbd_mon_host")
	newRbdKey, _ := data.GetString("rbd_key")
	for i := 0; i < len(storages); i++ {
		rbdHost, _ := storages[i].StorageConf.GetString("mon_host")
		rbdKey, _ := storages[i].StorageConf.GetString("key")
		if newRbdHost == rbdHost && newRbdKey == rbdKey {
			_, err := storage.GetModelManager().TableSpec().Update(storage, func() error {
				storage.StoragecacheId = storages[i].StoragecacheId
				return nil
			})
			if err != nil {
				log.Errorf("Update storagecacheId error: %v", err)
				return
			}
		}
	}
	if len(storage.StoragecacheId) == 0 {
		sc := &models.SStoragecache{}
		sc.SetModelManager(models.StoragecacheManager)
		sc.Name = fmt.Sprintf("imagecache-%s", storage.Id)
		pool, _ := data.GetString("rbd_pool")
		sc.Path = fmt.Sprintf("rbd:%s", pool)
		if err := models.StoragecacheManager.TableSpec().Insert(sc); err != nil {
			log.Errorf("insert storagecache for storage %s error: %v", storage.Name, err)
			return
		}
		_, err := storage.GetModelManager().TableSpec().Update(storage, func() error {
			storage.StoragecacheId = sc.Id
			return nil
		})
		if err != nil {
			log.Errorf("update storagecache info for storage %s error: %v", storage.Name, err)
		}
	}
}
