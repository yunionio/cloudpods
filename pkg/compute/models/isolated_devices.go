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
	"database/sql"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

	// name = Column(VARCHAR(16, charset='utf8'), nullable=True, default='', server_default='') # not used

	HostId string `width:"36" charset:"ascii" nullable:"false" default:"" index:"true" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False, default='', server_default='', index=True)

	// # PCI / GPU-HPC / GPU-VGA / USB / NIC
	DevType string `width:"16" charset:"ascii" nullable:"false" default:"" index:"true" list:"admin" create:"admin_required" update:"admin"` // Column(VARCHAR(16, charset='ascii'), nullable=False, default='', server_default='', index=True)

	// # Specific device name read from lspci command, e.g. `Tesla K40m` ...
	Model string `width:"32" charset:"ascii" nullable:"false" default:"" index:"true" list:"admin" create:"admin_required" update:"admin"` // Column(VARCHAR(32, charset='ascii'), nullable=False, default='', server_default='', index=True)

	GuestId string `width:"36" charset:"ascii" nullable:"true" index:"true" list:"admin"` // Column(VARCHAR(36, charset='ascii'), nullable=True, index=True)

	// # pci address of `Bus:Device.Function` format, or usb bus address of `bus.addr`
	Addr string `width:"16" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	VendorDeviceId string `width:"16" charset:"ascii" nullable:"true" list:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)
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

func (manager *SIsolatedDeviceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	hostId, _ := data.GetString("host_id")
	host := HostManager.FetchHostById(hostId)
	if host == nil {
		return nil, httperrors.NewNotFoundError("Host %s not found", hostId)
	}
	if name, _ := data.GetString("name"); len(name) == 0 {
		name = fmt.Sprintf("dev_%s_%d", host.GetName(), time.Now().UnixNano())
		data.Set("name", jsonutils.NewString(name))
	}

	input := apis.StandaloneResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneRes  ourceCreateInput fail %s", err)
	}
	input, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (self *SIsolatedDevice) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (manager *SIsolatedDeviceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	q, err = managedResourceFilterByDomain(q, query, "host_id", func() *sqlchemy.SQuery {
		return HostManager.Query("id")
	})

	if jsonutils.QueryBoolean(query, "gpu", false) {
		q = q.Startswith("dev_type", "GPU")
	}
	if jsonutils.QueryBoolean(query, "usb", false) {
		q = q.Equals("dev_type", "USB")
	}
	hostStr, _ := query.GetString("host")
	var sq *sqlchemy.SSubQuery
	if len(hostStr) > 0 {
		hosts := HostManager.Query().SubQuery()
		sq = hosts.Query(hosts.Field("id")).Filter(sqlchemy.OR(
			sqlchemy.Equals(hosts.Field("id"), hostStr),
			sqlchemy.Equals(hosts.Field("name"), hostStr))).SubQuery()
	}
	if sq != nil {
		q = q.Filter(sqlchemy.In(q.Field("host_id"), sq))
	}
	if jsonutils.QueryBoolean(query, "unused", false) {
		q = q.IsEmpty("guest_id")
	}
	regionStr := jsonutils.GetAnyString(query, []string{"region", "region_id"})
	if len(regionStr) > 0 {
		region, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudregionManager.Keyword(), regionStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		hosts := HostManager.Query().SubQuery()
		subq := ZoneManager.Query("id").Equals("cloudregion_id", region.GetId()).SubQuery()
		q.Join(hosts, sqlchemy.Equals(q.Field("host_id"), hosts.Field("id"))).Filter(sqlchemy.In(hosts.Field("zone_id"), subq))
	}
	zoneStr := jsonutils.GetAnyString(query, []string{"zone", "zone_id"})
	if len(zoneStr) > 0 {
		zone, _ := ZoneManager.FetchByIdOrName(nil, zoneStr)
		if zone == nil {
			return nil, httperrors.NewResourceNotFoundError("Zone %s not found", zoneStr)
		}
		hosts := HostManager.Query().SubQuery()
		sq := hosts.Query(hosts.Field("id")).Filter(sqlchemy.Equals(hosts.Field("zone_id"), zone.GetId()))
		q = q.Filter(sqlchemy.In(q.Field("host_id"), sq))
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

func (manager *SIsolatedDeviceManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SModelBaseManager.ListItemExportKeys(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	exportKeys, _ := query.GetString("export_keys")
	keys := strings.Split(exportKeys, ",")
	if utils.IsInStringArray("guest", keys) {
		guestNameQuery := GuestManager.Query("name", "id").SubQuery()
		q.LeftJoin(guestNameQuery, sqlchemy.Equals(q.Field("guest_id"), guestNameQuery.Field("id")))
		q.AppendField(guestNameQuery.Field("name", "guest"))
	}
	if utils.IsInStringArray("host", keys) {
		hostNameQuery := HostManager.Query("name", "id").SubQuery()
		q.LeftJoin(hostNameQuery, sqlchemy.Equals(q.Field("host_id"), hostNameQuery.Field("id")))
		q.AppendField(hostNameQuery.Field("name", "host"))
	}
	return q, nil
}

func (manager *SIsolatedDeviceManager) GetExportExtraKeys(ctx context.Context, query jsonutils.JSONObject, rowMap map[string]string) *jsonutils.JSONDict {
	res := manager.SStandaloneResourceBaseManager.GetExportExtraKeys(ctx, query, rowMap)
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
		if host.Status != api.BAREMETAL_RUNNING || !host.Enabled {
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

func (self *SIsolatedDevice) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	extra = self.getMoreDetails(extra)
	return extra, nil
}

func (self *SIsolatedDevice) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra = self.getMoreDetails(extra)
	return extra
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
