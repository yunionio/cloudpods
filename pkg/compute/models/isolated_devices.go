package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/pkg/httperrors"
	"github.com/yunionio/pkg/util/regutils"
	"github.com/yunionio/pkg/utils"
	"github.com/yunionio/sqlchemy"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
)

const (
	DIRECT_PCI_TYPE = "PCI"
	GPU_HPC_TYPE    = "GPU-HPC" // # for compute
	GPU_VGA_TYPE    = "GPU-VGA" // # for display
	USB_TYPE        = "USB"
	NIC_TYPE        = "NIC"

	NVIDIA_VENDOR_ID = "10de"
	AMD_VENDOR_ID    = "1002"
)

var VALID_GPU_TYPES = []string{GPU_HPC_TYPE, GPU_VGA_TYPE}

var VALID_PASSTHROUGH_TYPES = []string{DIRECT_PCI_TYPE, USB_TYPE, NIC_TYPE, GPU_HPC_TYPE, GPU_VGA_TYPE}

var ID_VENDOR_MAP = map[string]string{
	NVIDIA_VENDOR_ID: "NVIDIA",
	AMD_VENDOR_ID:    "AMD",
}

var VENDOR_ID_MAP = map[string]string{
	"NVIDIA": NVIDIA_VENDOR_ID,
	"AMD":    AMD_VENDOR_ID,
}

type SIsolatedDeviceManager struct {
	db.SStandaloneResourceBaseManager
}

var IsolatedDeviceManager *SIsolatedDeviceManager

func init() {
	IsolatedDeviceManager = &SIsolatedDeviceManager{SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(SIsolatedDevice{}, "isolated_devices_tbl", "isolated_device", "isolated_devices")}
}

type SIsolatedDevice struct {
	db.SStandaloneResourceBase

	// name = Column(VARCHAR(16, charset='utf8'), nullable=True, default='', server_default='') # not used

	HostId string `width:"36" charset:"ascii" nullable:"false" default:"" index:"true" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False, default='', server_default='', index=True)

	// # PCI / GPU-HPC / GPU-VGA / USB / NIC
	DevType string `width:"16" charset:"ascii" nullable:"false" default:"" index:"true" list:"admin" create:"admin_required"` // Column(VARCHAR(16, charset='ascii'), nullable=False, default='', server_default='', index=True)

	// # Specific device name read from lspci command, e.g. `Tesla K40m` ...
	Model string `width:"32" charset:"ascii" nullable:"false" default:"" index:"true" list:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False, default='', server_default='', index=True)

	GuestId string `width:"36" charset:"ascii" nullable:"true" index:"true" list:"admin"` // Column(VARCHAR(36, charset='ascii'), nullable=True, index=True)

	// # pci address of `Bus:Device.Function` format, or usb bus address of `bus.addr`
	Addr string `width:"16" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	VendorDeviceId string `width:"16" charset:"ascii" nullable:"true" list:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)
}

func (manager *SIsolatedDeviceManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	host, _ := query.GetString("host")
	if len(host) > 0 && !userCred.IsSystemAdmin() {
		return false
	}
	return true
}

func (manager *SIsolatedDeviceManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (manager *SIsolatedDeviceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	if jsonutils.QueryBoolean(query, "gpu", false) {
		q = q.Startswith("dev_type", "GPU")
	}
	if jsonutils.QueryBoolean(query, "usb", false) {
		q = q.Equals("dev_type", "USB")
	}
	hostStr, _ := query.GetString("host")
	if len(hostStr) > 0 {
		host, _ := HostManager.FetchByIdOrName("", hostStr)
		if host == nil {
			return nil, httperrors.NewResourceNotFoundError("Host %s not found", hostStr)
		}
		q = q.Equals("host_id", host.GetId())
	}
	if jsonutils.QueryBoolean(query, "unused", false) {
		q = q.IsEmpty("guest_id")
	}
	return q, nil
}

/*
func (self *SIsolatedDevice) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin()
}

func (self *SIsolatedDevice) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
} */

func (self *SIsolatedDevice) ValidateDeleteCondition(ctx context.Context) error {
	if len(self.GuestId) > 0 {
		return httperrors.NewNotEmptyError("Isolated device used by server")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SIsolatedDevice) getDetailedString() string {
	return fmt.Sprintf("%s:%s/%s/%s", self.Addr, self.Model, self.VendorDeviceId, self.DevType)
}

func (manager *SIsolatedDeviceManager) findAttachedDevicesOfGuest(guest *SGuest) []SIsolatedDevice {
	devs := make([]SIsolatedDevice, 0)
	q := manager.Query().Equals("guest_id", guest.Id)
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		log.Errorf("findAttachedDevicesOfGuest error %s", err)
		return nil
	}
	return devs
}

func (manager *SIsolatedDeviceManager) fuzzyMatchModel(fuzzyStr string) *SIsolatedDevice {
	dev := SIsolatedDevice{}
	dev.SetModelManager(manager)

	q := manager.Query().Contains("model", fuzzyStr)
	err := q.First(&dev)
	if err == nil {
		return &dev
	}
	return nil
}

func (self *SIsolatedDevice) getVendorId() string {
	parts := strings.Split(self.VendorDeviceId, ":")
	return parts[0]
}

func (self *SIsolatedDevice) getVendor() string {
	vendorId := self.getVendorId()
	vendor, ok := ID_VENDOR_MAP[vendorId]
	if ok {
		return vendor
	} else {
		return vendorId
	}
}

func (self *SIsolatedDevice) isGpu() bool {
	return strings.HasPrefix(self.DevType, "GPU")
}

type SIsolatedDeviceConfig struct {
	Id      string
	DevType string
	Model   string
	Vendor  string
}

func (manager *SIsolatedDeviceManager) parseDeviceInfo(userCred mcclient.TokenCredential, info jsonutils.JSONObject) (*SIsolatedDeviceConfig, error) {
	devConfig := SIsolatedDeviceConfig{}

	devJson, ok := info.(*jsonutils.JSONDict)
	if ok {
		err := devJson.Unmarshal(&devConfig)
		if err != nil {
			return nil, err
		}
		return &devConfig, nil
	}
	devStr, err := info.GetString()
	if err != nil {
		log.Errorf("invalid isolated device info format %s", err)
		return nil, err
	}
	var devId, devType, devVendor string
	var matchDev *SIsolatedDevice
	parts := strings.Split(devStr, ":")
	for _, p := range parts {
		if regutils.MatchUUIDExact(p) {
			devId = p
		} else if utils.IsInStringArray(p, VALID_PASSTHROUGH_TYPES) {
			devType = p
		} else if strings.HasPrefix(p, "vendor=") {
			devVendor = p[len("vendor="):]
		} else {
			matchDev = manager.fuzzyMatchModel(p)
		}
	}
	if len(devId) == 0 {
		if matchDev == nil {
			return nil, fmt.Errorf("Isolated device info not contains either deviceID or model name")
		}
		devConfig.Model = matchDev.Model
		if len(devVendor) > 0 {
			vendorId, ok := VENDOR_ID_MAP[devVendor]
			if ok {
				devConfig.Vendor = vendorId
			} else {
				devConfig.Vendor = devVendor
			}
		} else {
			devConfig.Vendor = matchDev.getVendorId()
		}

	} else {
		devObj, err := manager.FetchById(devId)
		if err != nil {
			return nil, fmt.Errorf("IsolatedDevice %s not found: %s", devId, err)
		}
		dev := devObj.(*SIsolatedDevice)

		devConfig.Id = dev.Id
		devConfig.Model = dev.Model
		devConfig.DevType = dev.DevType
		devConfig.Vendor = dev.getVendor()
		if dev.isGpu() && len(devType) > 0 {
			if !utils.IsInStringArray(devType, VALID_GPU_TYPES) {
				return nil, fmt.Errorf("%s not valid for GPU device", devType)
			}
		}
	}
	if len(devType) > 0 {
		devConfig.DevType = devType
	}
	return &devConfig, nil
}

func (manager *SIsolatedDeviceManager) isValidDeviceinfo(config *SIsolatedDeviceConfig) error {
	if len(config.Id) > 0 {
		devObj, err := manager.FetchById(config.Id)
		if err != nil {
			return httperrors.NewResourceNotFoundError("IsolatedDevice %s not found", config.Id)
		}
		dev := devObj.(*SIsolatedDevice)
		if len(dev.GuestId) > 0 {
			return httperrors.NewConflictError("Isolated device already attached to another guest: %s", dev.GuestId)
		}
	}
	return nil
}

func (manager *SIsolatedDeviceManager) attachHostDeviceToGuestByDesc(guest *SGuest, host *SHost, devConfig *SIsolatedDeviceConfig, userCred mcclient.TokenCredential) error {
	if len(devConfig.Id) > 0 {
		return manager.attachSpecificDeviceToGuest(guest, devConfig, userCred)
	} else {
		return manager.attachHostDeviceToGuestByModel(guest, host, devConfig, userCred)
	}
}

func (manager *SIsolatedDeviceManager) attachSpecificDeviceToGuest(guest *SGuest, devConfig *SIsolatedDeviceConfig, userCred mcclient.TokenCredential) error {
	devObj, err := manager.FetchById(devConfig.Id)
	if devObj == nil {
		return fmt.Errorf("Device %s not found: %s", devConfig.Id, err)
	}
	dev := devObj.(*SIsolatedDevice)
	if len(devConfig.DevType) > 0 && devConfig.DevType != dev.DevType {
		dev.DevType = devConfig.DevType
	}
	return guest.attachIsolatedDevice(userCred, dev)
}

func (manager *SIsolatedDeviceManager) attachHostDeviceToGuestByModel(guest *SGuest, host *SHost, devConfig *SIsolatedDeviceConfig, userCred mcclient.TokenCredential) error {
	if len(devConfig.Model) == 0 {
		return fmt.Errorf("Not found model from info: %s", devConfig)
	}
	devs, err := manager.findHostUnusedByModel(devConfig.Model, host.Id)
	if err != nil || len(devs) == 0 {
		return fmt.Errorf("Can't found %s model on host", host.Id)
	}
	selectedDev := devs[0]
	return guest.attachIsolatedDevice(userCred, &selectedDev)
}

func (manager *SIsolatedDeviceManager) findUnusedQuery() *sqlchemy.SQuery {
	isolateddevs := manager.Query().SubQuery()
	q := isolateddevs.Query().Filter(sqlchemy.OR(sqlchemy.IsNull(isolateddevs.Field("guest_id")),
		sqlchemy.IsEmpty(isolateddevs.Field("guest_id"))))
	return q
}

func (manager *SIsolatedDeviceManager) findHostUnusedByModel(model string, hostId string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.findUnusedQuery()
	q = q.Equals("model", model).Equals("host_id", hostId)
	err := q.All(&devs)
	if err != nil {
		return nil, err
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) ReleaseDevicesOfGuest(guest *SGuest, userCred mcclient.TokenCredential) error {
	devs := manager.findAttachedDevicesOfGuest(guest)
	if devs == nil {
		return fmt.Errorf("fail to find attached devices")
	}
	for _, dev := range devs {
		_, err := manager.TableSpec().Update(&dev, func() error {
			dev.GuestId = ""
			return nil
		})
		if err != nil {
			db.OpsLog.LogEvent(guest, db.ACT_GUEST_DETACH_ISOLATED_DEVICE_FAIL, dev.GetShortDesc(), userCred)
			return err
		}
		db.OpsLog.LogEvent(guest, db.ACT_GUEST_DETACH_ISOLATED_DEVICE, dev.GetShortDesc(), userCred)
	}
	return nil
}

func (manager *SIsolatedDeviceManager) totalCountQ(
	devType, hostTypes []string,
	rangeObj db.IStandaloneModel,
) *sqlchemy.SQuery {
	hosts := HostManager.Query().SubQuery()
	q := manager.Query().Join(hosts, sqlchemy.AND(
		sqlchemy.IsFalse(hosts.Field("deleted")),
		sqlchemy.IsTrue(hosts.Field("enabled")),
	))
	if len(devType) != 0 {
		q.In("dev_type", devType)
	}
	devs := manager.Query().SubQuery()
	return AttachUsageQuery(q, hosts, devs.Field("host_id"), hostTypes, rangeObj)
}

type IsolatedDeviceCountStat struct {
	Devices int
	Gpus    int
}

func (manager *SIsolatedDeviceManager) totalCount(devType, hostTypes []string, rangeObj db.IStandaloneModel) int {
	return manager.totalCountQ(devType, hostTypes, rangeObj).Count()
}

func (manager *SIsolatedDeviceManager) TotalCount(hostType []string, rangeObj db.IStandaloneModel) IsolatedDeviceCountStat {
	return IsolatedDeviceCountStat{
		Devices: manager.totalCount(nil, hostType, rangeObj),
		Gpus:    manager.totalCount(VALID_GPU_TYPES, hostType, rangeObj),
	}
}

func (self *SIsolatedDevice) getDesc() *jsonutils.JSONDict {
	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewString(self.Id), "id")
	desc.Add(jsonutils.NewString(self.DevType), "dev_type")
	desc.Add(jsonutils.NewString(self.Model), "model")
	desc.Add(jsonutils.NewString(self.Addr), "addr")
	desc.Add(jsonutils.NewString(self.VendorDeviceId), "vendor_device_id")
	desc.Add(jsonutils.NewString(self.getVendor()), "vendor")
	return desc
}

func (self *SIsolatedDevice) GetShortDesc() *jsonutils.JSONDict {
	desc := self.getDesc()
	desc.Add(jsonutils.NewString(self.Keyword()), "res_name")
	return desc
}

func (manager *SIsolatedDeviceManager) generateJsonDescForGuest(guest *SGuest) []jsonutils.JSONObject {
	ret := make([]jsonutils.JSONObject, 0)
	devs := manager.findAttachedDevicesOfGuest(guest)
	if devs != nil && len(devs) > 0 {
		for _, dev := range devs {
			ret = append(ret, dev.getDesc())
		}
	}
	return ret
}

func (self *SIsolatedDevice) getHost() *SHost {
	return HostManager.FetchHostById(self.HostId)
}

func (self *SIsolatedDevice) getGuest() *SGuest {
	if len(self.GuestId) > 0 {
		return GuestManager.FetchGuestById(self.GuestId)
	}
	return nil
}

func (self *SIsolatedDevice) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	host := self.getHost()
	if host != nil {
		extra.Add(jsonutils.NewString(host.Name), "host")
	}
	guest := self.getGuest()
	if guest != nil {
		extra.Add(jsonutils.NewString(guest.Name), "guest")
		extra.Add(jsonutils.NewString(guest.Status), "guest_status")
	}
	return extra
}

func (self *SIsolatedDevice) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	extra = self.getMoreDetails(extra)
	return extra
}

func (self *SIsolatedDevice) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = self.getMoreDetails(extra)
	return extra
}

func (self *SIsolatedDevice) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	host := self.getHost()
	if host != nil {
		db.OpsLog.LogEvent(host, db.ACT_HOST_DETACH_ISOLATED_DEVICE, self.GetShortDesc(), userCred)
	}
	return nil
}
