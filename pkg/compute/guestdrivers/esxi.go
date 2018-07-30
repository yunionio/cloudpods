package guestdrivers

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"

	"github.com/yunionio/onecloud/pkg/compute/models"
)

type SESXiGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SESXiGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SESXiGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_ESXI
}

func (self *SESXiGuestDriver) GetGuestVncInfo(userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	data := jsonutils.NewDict()
	// TODO
	return data, nil
}
