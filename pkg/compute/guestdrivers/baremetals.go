package guestdrivers

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"

	"github.com/yunionio/oneclone/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/oneclone/pkg/compute/models"
)

type SBaremetalGuestDriver struct {
	SBaseGuestDriver
}

func init() {
	// driver := SBaremetalGuestDriver{}
	// models.RegisterGuestDriver(&driver)
}

func (self *SBaremetalGuestDriver) GetHypervisor() string {
	return "baremetal"
}

func (self *SBaremetalGuestDriver) PrepareDiskRaidConfig(host *models.SHost, params *jsonutils.JSONDict) {

}

func (self *SBaremetalGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	bs := host.GetBaremetalstorage()
	return bs.GetStorage()
}

func (self *SBaremetalGuestDriver) RequestGuestCreateAllDisks(guest *models.SGuest, task taskman.ITask) {
	task.ScheduleRun(nil) // skip
}

func (self *SBaremetalGuestDriver) RequestGuestCreateInsertIso(imageId string, guest *models.SGuest, task taskman.ITask) {
	task.ScheduleRun(nil) // skip
}

func (self *SBaremetalGuestDriver) ValidateCreateData(userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SBaremetalGuestDriver) ValidateCreateHostData(userCred mcclient.TokenCredential, bmName string, host *models.SHost, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SBaremetalGuestDriver) GetJsonDescAtHost(ctx context.Context, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	return guest.GetJsonDescAtBaremetal(ctx, host)
}
