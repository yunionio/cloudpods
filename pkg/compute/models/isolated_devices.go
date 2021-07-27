// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	DIRECT_PCI_TYPE = api.DIRECT_PCI_TYPE
	GPU_HPC_TYPE    = api.GPU_HPC_TYPE // # for compute
	GPU_VGA_TYPE    = api.GPU_VGA_TYPE // # for display
	USB_TYPE        = api.USB_TYPE
	NIC_TYPE        = api.NIC_TYPE

	NVIDIA_VENDOR_ID = api.NVIDIA_VENDOR_ID
	AMD_VENDOR_ID    = api.AMD_VENDOR_ID
)

var VALID_GPU_TYPES = api.VALID_GPU_TYPES

var VALID_PASSTHROUGH_TYPES = api.VALID_PASSTHROUGH_TYPES

var ID_VENDOR_MAP = api.ID_VENDOR_MAP

var VENDOR_ID_MAP = api.VENDOR_ID_MAP

type SIsolatedDeviceManager struct {
	db.SStandaloneResourceBaseManager
	SHostResourceBaseManager
}

var IsolatedDeviceManager *SIsolatedDeviceManager

func init() {
	IsolatedDeviceManager = &SIsolatedDeviceManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SIsolatedDevice{},
			"isolated_devices_tbl",
			"isolated_device",
			"isolated_devices",
		),
	}
	IsolatedDeviceManager.SetVirtualObject(IsolatedDeviceManager)
}

type SIsolatedDevice struct {
	db.SStandaloneResourceBase
	SHostResourceBase `width:"36" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required"`

	// 宿主机Id
	// HostId string `width:"36" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required"`

	// # PCI / GPU-HPC / GPU-VGA / USB / NIC
	// 设备类型
	DevType string `width:"16" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required" update:"domain"`

	// # Specific device name read from lspci command, e.g. `Tesla K40m` ...
	Model string `width:"32" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required" update:"domain"`

	// 云主机Id
	GuestId string `width:"36" charset:"ascii" nullable:"true" index:"true" list:"domain"`

	// # pci address of `Bus:Device.Function` format, or usb bus address of `bus.addr`
	Addr string `width:"16" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	VendorDeviceId string `width:"16" charset:"ascii" nullable:"true" list:"domain" create:"domain_optional"`

	// reserved memory size for isolated device, default 8G
	ReservedMemory int `nullable:"true" default:"8192" list:"domain" update:"domain" create:"domain_optional"`

	// reserved cpu count for isolated device, default 8
	ReservedCpu int `nullable:"true" default:"8" list:"domain" update:"domain" create:"domain_optional"`

	// reserved storage size for isolated device, default 100G
	ReservedStorage int `nullable:"true" default:"102400" list:"domain" update:"domain" create:"domain_optional"`
}

func (manager *SIsolatedDeviceManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	host, _ := query.GetString("host")
	if len(host) > 0 && !db.IsAdminAllowList(userCred, manager) {
		return false
	}
	return true
}

func (manager *SIsolatedDeviceManager) ExtraSearchConditions(ctx context.Context, q *sqlchemy.SQuery, like string) []sqlchemy.ICondition {
	sq := HostManager.Query("id").Contains("name", like).SubQuery()
	return []sqlchemy.ICondition{sqlchemy.In(q.Field("host_id"), sq)}
}

func (manager *SIsolatedDeviceManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (manager *SIsolatedDeviceManager) ValidateCreateData(ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.IsolatedDeviceCreateInput,
) (api.IsolatedDeviceCreateInput, error) {
	var err error
	var host *SHost
	host, input.HostResourceInput, err = ValidateHostResourceInput(userCred, input.HostResourceInput)
	if err != nil {
		return input, errors.Wrap(err, "ValidateHostResourceInput")
	}
	if len(input.Name) == 0 {
		input.Name = fmt.Sprintf("dev_%s_%d", host.GetName(), time.Now().UnixNano())
	}

	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBaseManager.ValidateCreateData")
	}

	if input.ReservedCpu != nil && *input.ReservedCpu < 0 {
		return input, httperrors.NewInputParameterError("reserved cpu must >= 0")
	}
	if input.ReservedMemory != nil && *input.ReservedMemory < 0 {
		return input, httperrors.NewInputParameterError("reserved memory must >= 0")
	}
	if input.ReservedStorage != nil && *input.ReservedStorage < 0 {
		return input, httperrors.NewInputParameterError("reserved storage must >= 0")
	}
	return input, nil
}

func (self *SIsolatedDevice) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SIsolatedDevice) ValidateUpdateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input api.IsolatedDeviceUpdateInput,
) (api.IsolatedDeviceUpdateInput, error) {
	var err error
	input.StandaloneResourceBaseUpdateInput, err = self.SStandaloneResourceBase.ValidateUpdateData(
		ctx, userCred, query, input.StandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	if input.ReservedCpu != nil && *input.ReservedCpu < 0 {
		return input, httperrors.NewInputParameterError("reserved cpu must >= 0")
	}
	if input.ReservedMemory != nil && *input.ReservedMemory < 0 {
		return input, httperrors.NewInputParameterError("reserved memory must >= 0")
	}
	if input.ReservedStorage != nil && *input.ReservedStorage < 0 {
		return input, httperrors.NewInputParameterError("reserved storage must >= 0")
	}
	return input, nil
}

func (self *SIsolatedDevice) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	HostManager.ClearSchedDescCache(self.HostId)
}

// 直通设备（GPU等）列表
func (manager *SIsolatedDeviceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IsolatedDeviceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SHostResourceBaseManager.ListItemFilter(ctx, q, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemFilter")
	}

	if query.Gpu != nil && *query.Gpu {
		q = q.Startswith("dev_type", "GPU")
	}
	if query.Usb != nil && *query.Usb {
		q = q.Equals("dev_type", "USB")
	}
	if query.Unused != nil && *query.Unused {
		q = q.IsEmpty("guest_id")
	}

	if len(query.DevType) > 0 {
		q = q.In("dev_type", query.DevType)
	}
	if len(query.Model) > 0 {
		q = q.In("model", query.Model)
	}
	if len(query.Addr) > 0 {
		q = q.In("addr", query.Addr)
	}
	if len(query.VendorDeviceId) > 0 {
		q = q.In("vendor_device_id", query.VendorDeviceId)
	}

	if !query.ShowBaremetalIsolatedDevices {
		sq := HostManager.Query("id").Equals("host_type", api.HOST_TYPE_HYPERVISOR).SubQuery()
		q = q.In("host_id", sq)
	}

	return q, nil
}

func (manager *SIsolatedDeviceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.IsolatedDeviceListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SHostResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SIsolatedDeviceManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SHostResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

/*
func (self *SIsolatedDevice) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin()
}

func (self *SIsolatedDevice) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
} */

func (manager *SIsolatedDeviceManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, err
	}
	if keys.Contains("guest") {
		guestNameQuery := GuestManager.Query("name", "id").SubQuery()
		q.LeftJoin(guestNameQuery, sqlchemy.Equals(q.Field("guest_id"), guestNameQuery.Field("id")))
		q.AppendField(guestNameQuery.Field("name", "guest"))
	}
	if keys.Contains("host") {
		hostNameQuery := HostManager.Query("name", "id").SubQuery()
		q.LeftJoin(hostNameQuery, sqlchemy.Equals(q.Field("host_id"), hostNameQuery.Field("id")))
		q.AppendField(hostNameQuery.Field("name", "host"))
	}
	return q, nil
}

func (manager *SIsolatedDeviceManager) GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	res := manager.SStandaloneResourceBaseManager.GetExportExtraKeys(ctx, keys, rowMap)
	if guest, ok := rowMap["guest"]; ok && len(guest) > 0 {
		res.Set("guest", jsonutils.NewString(guest))
	}
	if host, ok := rowMap["host"]; ok {
		res.Set("host", jsonutils.NewString(host))
	}
	return res
}

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
	dev.SetModelManager(manager, &dev)

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

func (manager *SIsolatedDeviceManager) parseDeviceInfo(userCred mcclient.TokenCredential, devConfig *api.IsolatedDeviceConfig) (*api.IsolatedDeviceConfig, error) {
	var devId, devType, devVendor string
	var matchDev *SIsolatedDevice

	devId = devConfig.Id
	matchDev = manager.fuzzyMatchModel(devConfig.Model)
	devVendor = devConfig.Vendor
	devType = devConfig.DevType

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
	return devConfig, nil
}

func (manager *SIsolatedDeviceManager) isValidDeviceinfo(config *api.IsolatedDeviceConfig) error {
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

func (manager *SIsolatedDeviceManager) attachHostDeviceToGuestByDesc(ctx context.Context, guest *SGuest, host *SHost, devConfig *api.IsolatedDeviceConfig, userCred mcclient.TokenCredential) error {
	if len(devConfig.Id) > 0 {
		return manager.attachSpecificDeviceToGuest(ctx, guest, devConfig, userCred)
	} else {
		return manager.attachHostDeviceToGuestByModel(ctx, guest, host, devConfig, userCred)
	}
}

func (manager *SIsolatedDeviceManager) attachSpecificDeviceToGuest(ctx context.Context, guest *SGuest, devConfig *api.IsolatedDeviceConfig, userCred mcclient.TokenCredential) error {
	devObj, err := manager.FetchById(devConfig.Id)
	if devObj == nil {
		return fmt.Errorf("Device %s not found: %s", devConfig.Id, err)
	}
	dev := devObj.(*SIsolatedDevice)
	if len(devConfig.DevType) > 0 && devConfig.DevType != dev.DevType {
		dev.DevType = devConfig.DevType
	}
	return guest.attachIsolatedDevice(ctx, userCred, dev)
}

func (manager *SIsolatedDeviceManager) attachHostDeviceToGuestByModel(ctx context.Context, guest *SGuest, host *SHost, devConfig *api.IsolatedDeviceConfig, userCred mcclient.TokenCredential) error {
	if len(devConfig.Model) == 0 {
		return fmt.Errorf("Not found model from info: %#v", devConfig)
	}
	devs, err := manager.findHostUnusedByModel(devConfig.Model, host.Id)
	if err != nil || len(devs) == 0 {
		return fmt.Errorf("Can't found %s model on host", host.Id)
	}
	selectedDev := devs[0]
	return guest.attachIsolatedDevice(ctx, userCred, &selectedDev)
}

func (manager *SIsolatedDeviceManager) findUnusedQuery() *sqlchemy.SQuery {
	isolateddevs := manager.Query().SubQuery()
	q := isolateddevs.Query().Filter(sqlchemy.OR(sqlchemy.IsNull(isolateddevs.Field("guest_id")),
		sqlchemy.IsEmpty(isolateddevs.Field("guest_id"))))
	return q
}

func (manager *SIsolatedDeviceManager) UnusedGpuQuery() *sqlchemy.SQuery {
	q := manager.findUnusedQuery()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.Equals(q.Field("dev_type"), GPU_HPC_TYPE),
		sqlchemy.Equals(q.Field("dev_type"), GPU_VGA_TYPE)))
	return q
}

func (manager *SIsolatedDeviceManager) FindUnusedByModels(models []string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.findUnusedQuery()
	q = q.In("model", models)
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) FindUnusedGpusOnHost(hostId string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.UnusedGpuQuery()
	q = q.Equals("host_id", hostId)
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) findHostUnusedByModel(model string, hostId string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.findUnusedQuery()
	q = q.Equals("model", model).Equals("host_id", hostId)
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) ReleaseDevicesOfGuest(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential) error {
	devs := manager.findAttachedDevicesOfGuest(guest)
	if devs == nil {
		return fmt.Errorf("fail to find attached devices")
	}
	for _, dev := range devs {
		_, err := db.Update(&dev, func() error {
			dev.GuestId = ""
			return nil
		})
		if err != nil {
			db.OpsLog.LogEvent(guest, db.ACT_GUEST_DETACH_ISOLATED_DEVICE_FAIL, dev.GetShortDesc(ctx), userCred)
			return err
		}
		db.OpsLog.LogEvent(guest, db.ACT_GUEST_DETACH_ISOLATED_DEVICE, dev.GetShortDesc(ctx), userCred)
	}
	return nil
}

func (manager *SIsolatedDeviceManager) totalCountQ(
	devType []string, hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	rangeObjs []db.IStandaloneModel,
) *sqlchemy.SQuery {
	hosts := HostManager.Query().SubQuery()
	devs := manager.Query().SubQuery()
	q := devs.Query().Join(hosts, sqlchemy.Equals(devs.Field("host_id"), hosts.Field("id")))
	q = q.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
	if len(devType) != 0 {
		q = q.Filter(sqlchemy.In(devs.Field("dev_type"), devType))
	}
	return AttachUsageQuery(q, hosts, hostTypes, resourceTypes, providers, brands, cloudEnv, rangeObjs)
}

type IsolatedDeviceCountStat struct {
	Devices int
	Gpus    int
}

func (manager *SIsolatedDeviceManager) totalCount(
	devType,
	hostTypes []string,
	resourceTypes []string,
	providers []string,
	brands []string,
	cloudEnv string,
	rangeObjs []db.IStandaloneModel,
) (int, error) {
	return manager.totalCountQ(
		devType,
		hostTypes,
		resourceTypes,
		providers,
		brands,
		cloudEnv,
		rangeObjs,
	).CountWithError()
}

func (manager *SIsolatedDeviceManager) TotalCount(
	hostType []string,
	resourceTypes []string,
	providers []string,
	brands []string,
	cloudEnv string,
	rangeObjs []db.IStandaloneModel,
) (IsolatedDeviceCountStat, error) {
	stat := IsolatedDeviceCountStat{}
	devCnt, err := manager.totalCount(
		nil, hostType, resourceTypes,
		providers, brands, cloudEnv,
		rangeObjs)
	if err != nil {
		return stat, err
	}
	gpuCnt, err := manager.totalCount(
		VALID_GPU_TYPES, hostType, resourceTypes,
		providers, brands, cloudEnv,
		rangeObjs)
	if err != nil {
		return stat, err
	}
	stat.Devices = devCnt
	stat.Gpus = gpuCnt
	return stat, nil
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

func (man *SIsolatedDeviceManager) GetSpecShouldCheckStatus(query *jsonutils.JSONDict) (bool, error) {
	return true, nil
}

func (self *SIsolatedDevice) GetSpec(statusCheck bool) *jsonutils.JSONDict {
	if statusCheck {
		if len(self.GuestId) > 0 {
			return nil
		}
		host := self.getHost()
		if host.Status != api.BAREMETAL_RUNNING || !host.GetEnabled() || host.HostType != api.HOST_TYPE_HYPERVISOR {
			return nil
		}
	}
	spec := jsonutils.NewDict()
	spec.Set("dev_type", jsonutils.NewString(self.DevType))
	spec.Set("model", jsonutils.NewString(self.Model))
	spec.Set("pci_id", jsonutils.NewString(self.VendorDeviceId))
	spec.Set("vendor", jsonutils.NewString(self.getVendor()))
	return spec
}

func (man *SIsolatedDeviceManager) GetSpecIdent(spec *jsonutils.JSONDict) []string {
	devType, _ := spec.GetString("dev_type")
	vendor, _ := spec.GetString("vendor")
	model, _ := spec.GetString("model")
	keys := []string{
		fmt.Sprintf("type:%s", devType),
		fmt.Sprintf("vendor:%s", vendor),
		fmt.Sprintf("model:%s", model),
	}
	return keys
}

func (self *SIsolatedDevice) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.getDesc()
	desc.Add(jsonutils.NewString(IsolatedDeviceManager.Keyword()), "res_name")
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

func (manager *SIsolatedDeviceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.IsolateDeviceDetails {
	rows := make([]api.IsolateDeviceDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	hostRows := manager.SHostResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	guestIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.IsolateDeviceDetails{
			StandaloneResourceDetails: stdRows[i],
			HostResourceInfo:          hostRows[i],
		}
		guestIds[i] = objs[i].(*SIsolatedDevice).GuestId
	}

	guests := make(map[string]SGuest)
	err := db.FetchStandaloneObjectsByIds(GuestManager, guestIds, &guests)
	if err != nil {
		log.Errorf("db.FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		if guest, ok := guests[guestIds[i]]; ok {
			rows[i].Guest = guest.Name
			rows[i].GuestStatus = guest.Status
		}
	}

	return rows
}

func (self *SIsolatedDevice) ClearSchedDescCache() error {
	if len(self.HostId) == 0 {
		return nil
	}
	host := self.getHost()
	return host.ClearSchedDescCache()
}

func (self *SIsolatedDevice) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.SStandaloneResourceBase.Delete(ctx, userCred)
	if err != nil {
		return err
	}
	return self.ClearSchedDescCache()
}

func (self *SIsolatedDevice) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SIsolatedDevice) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, self.CustomizeDelete(ctx, userCred, query, data)
}

func (self *SIsolatedDevice) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if len(self.GuestId) > 0 {
		if !jsonutils.QueryBoolean(data, "purge", false) {
			return httperrors.NewBadRequestError("Isolated device used by server: %s", self.GuestId)
		}
		iGuest, err := GuestManager.FetchById(self.GuestId)
		if err != nil {
			return err
		}
		guest := iGuest.(*SGuest)
		err = guest.detachIsolateDevice(ctx, userCred, self)
		if err != nil {
			return err
		}
	}
	host := self.getHost()
	if host != nil {
		db.OpsLog.LogEvent(host, db.ACT_HOST_DETACH_ISOLATED_DEVICE, self.GetShortDesc(ctx), userCred)
	}
	return self.RealDelete(ctx, userCred)
}

func (manager *SIsolatedDeviceManager) FindByHost(id string) []SIsolatedDevice {
	return manager.FindByHosts([]string{id})
}

func (manager *SIsolatedDeviceManager) FindByHosts(ids []string) []SIsolatedDevice {
	dest := make([]SIsolatedDevice, 0)
	q := manager.Query().In("host_id", ids)
	err := db.FetchModelObjects(manager, q, &dest)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	return dest
}

func (manager *SIsolatedDeviceManager) DeleteDevicesByHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost) {
	for _, dev := range manager.FindByHost(host.Id) {
		dev.Delete(ctx, userCred)
	}
}

func (manager *SIsolatedDeviceManager) GetDevsOnHost(hostId string, model string, count int) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.Query().Equals("host_id", hostId).Equals("model", model).IsNullOrEmpty("guest_id").Limit(count)
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}

func (self *SIsolatedDevice) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{"host_id": self.HostId})
}

func (manager *SIsolatedDeviceManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	hostId, _ := data.GetString("host_id")
	return jsonutils.Marshal(map[string]string{"host_id": hostId})
}

func (manager *SIsolatedDeviceManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	hostId, _ := values.GetString("host_id")
	if len(hostId) > 0 {
		q = q.Equals("host_id", hostId)
	}
	return q
}

func (manager *SIsolatedDeviceManager) NamespaceScope() rbacutils.TRbacScope {
	if consts.IsDomainizedNamespace() {
		return rbacutils.ScopeDomain
	} else {
		return rbacutils.ScopeSystem
	}
}

func (manager *SIsolatedDeviceManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (manager *SIsolatedDeviceManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeProject, rbacutils.ScopeDomain:
			hostsQ := HostManager.Query("id")
			hostsQ = HostManager.FilterByOwner(hostsQ, owner, scope)
			hosts := hostsQ.SubQuery()
			q = q.Join(hosts, sqlchemy.Equals(q.Field("host_id"), hosts.Field("id")))
		}
	}
	return q
}

func (manager *SIsolatedDeviceManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchDomainInfo(ctx, data)
}

func (model *SIsolatedDevice) GetOwnerId() mcclient.IIdentityProvider {
	host := model.getHost()
	if host != nil {
		return host.GetOwnerId()
	}
	return nil
}
