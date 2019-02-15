package storagedrivers

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLocalStorageDriver struct {
	SBaseStorageDriver
}

func init() {
	driver := SLocalStorageDriver{}
	models.RegisterStorageDriver(&driver)
}

func (self *SLocalStorageDriver) GetStorageType() string {
	return models.STORAGE_LOCAL
}

func (self *SLocalStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SLocalStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {

}
