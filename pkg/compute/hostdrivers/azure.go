package hostdrivers

import (
	"context"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAzureHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SAzureHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SAzureHostDriver) GetHostType() string {
	return models.HOST_TYPE_AZURE
}

func (self *SAzureHostDriver) ValidateUpdateDisk(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("name") {
		return nil, httperrors.NewInputParameterError("cannot support change azure disk name")
	}
	return data, nil
}

func (self *SAzureHostDriver) RequestDeleteSnapshotWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("not implement")
}
