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
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
//DIRECT_PCI_TYPE = api.DIRECT_PCI_TYPE
//GPU_HPC_TYPE = api.GPU_HPC_TYPE // # for compute
//GPU_VGA_TYPE = api.GPU_VGA_TYPE // # for display
//USB_TYPE        = api.USB_TYPE
//NIC_TYPE        = api.NIC_TYPE

// NVIDIA_VENDOR_ID = api.NVIDIA_VENDOR_ID
// AMD_VENDOR_ID    = api.AMD_VENDOR_ID
)

var VALID_GPU_TYPES = api.VALID_GPU_TYPES

var VALID_PASSTHROUGH_TYPES = api.VALID_PASSTHROUGH_TYPES

var ID_VENDOR_MAP = api.ID_VENDOR_MAP

var VENDOR_ID_MAP = api.VENDOR_ID_MAP

type SIsolatedDeviceManager struct {
	db.SStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SHostResourceBaseManager
}

var IsolatedDeviceManager *SIsolatedDeviceManager

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&api.IsolatedDevicePCIEInfo{}), func() gotypes.ISerializable {
		return &api.IsolatedDevicePCIEInfo{}
	})

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
	db.SExternalizedResourceBase
	SHostResourceBase `width:"36" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required"`

	// # PCI / GPU-HPC / GPU-VGA / USB / NIC
	// 设备类型
	DevType string `width:"36" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required" update:"domain"`

	// # Specific device name read from lspci command, e.g. `Tesla K40m` ...
	Model string `width:"512" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required" update:"domain"`

	// 云主机Id
	GuestId string `width:"36" charset:"ascii" nullable:"true" index:"true" list:"domain"`
	// guest network index
	NetworkIndex int `nullable:"true" default:"-1" list:"user" update:"user"`
	// Nic wire id
	WireId string `width:"36" charset:"ascii" nullable:"true" index:"true" list:"domain" update:"domain" create:"domain_optional"`
	// Offload interface name
	OvsOffloadInterface string `width:"16" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// Is infiniband nic
	IsInfinibandNic bool `nullable:"false" default:"false" list:"user" create:"optional"`
	// NVME disk size
	NvmeSizeMB int `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// guest disk index
	DiskIndex int8 `nullable:"true" default:"-1" list:"user" update:"user"`

	// # pci address of `Bus:Device.Function` format, or usb bus address of `bus.addr`
	Addr       string `width:"16" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	DevicePath string `width:"128" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"optional"`

	// Is vgpu physical funcion, That means it cannot be attached to guest
	// VGPUPhysicalFunction bool `nullable:"true" default:"false" list:"domain" create:"domain_optional"`
	// nvidia vgpu config
	// vgpu uuid generated on create
	MdevId string `width:"36" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// The frame rate limiter (FRL) configuration in frames per second
	FRL string `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// The frame buffer size in Mbytes
	Framebuffer string `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// The maximum resolution per display head, eg: 5120x2880
	MaxResolution string `width:"16" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// The maximum number of virtual display heads that the vGPU type supports
	// In computer graphics and display technology, the term "head" is commonly used to
	// describe the physical interface of a display device or display output.
	// It refers to a connection point on the monitor, such as HDMI, DisplayPort, or VGA interface.
	NumHeads string `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// The maximum number of vGPU instances per physical GPU
	MaxInstance string `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// MPS perdevice memory limit MB
	MpsMemoryLimit int `nullable:"true" default:"-1" list:"domain" update:"domain" create:"domain_optional"`
	// MPS device memory total MB
	MpsMemoryTotal int `nullable:"true" default:"-1" list:"domain" update:"domain" create:"domain_optional"`
	// MPS device thread percentage
	MpsThreadPercentage int `nullable:"true" default:"-1" list:"domain" update:"domain" create:"domain_optional"`

	VendorDeviceId string `width:"16" charset:"ascii" nullable:"true" list:"domain" create:"domain_optional"`

	// reserved memory size for isolated device
	ReservedMemory int `nullable:"true" default:"0" list:"domain" update:"domain" create:"domain_optional"`

	// reserved cpu count for isolated device
	ReservedCpu int `nullable:"true" default:"0" list:"domain" update:"domain" create:"domain_optional"`

	// reserved storage size for isolated device
	ReservedStorage int `nullable:"true" default:"0" list:"domain" update:"domain" create:"domain_optional"`

	// PciInfo stores extra PCIE information
	PcieInfo *api.IsolatedDevicePCIEInfo `nullable:"true" create:"optional" list:"user" get:"user" update:"domain"`
	// device numa node
	NumaNode int8 `nullable:"true" default:"-1" list:"domain" update:"domain" create:"domain_optional"`
}

func (manager *SIsolatedDeviceManager) ExtraSearchConditions(ctx context.Context, q *sqlchemy.SQuery, like string) []sqlchemy.ICondition {
	sq := HostManager.Query("id").Contains("name", like).SubQuery()
	return []sqlchemy.ICondition{sqlchemy.In(q.Field("host_id"), sq)}
}

func (manager *SIsolatedDeviceManager) ValidateCreateData(ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.IsolatedDeviceCreateInput,
) (api.IsolatedDeviceCreateInput, error) {
	var err error
	var host *SHost
	host, input.HostResourceInput, err = ValidateHostResourceInput(ctx, userCred, input.HostResourceInput)
	if err != nil {
		return input, errors.Wrap(err, "ValidateHostResourceInput")
	}
	if len(input.Name) == 0 {
		input.Name = fmt.Sprintf("dev_%s_%d", host.GetName(), time.Now().UnixNano())
	}

	//  validate DevType
	if input.DevType == "" {
		return input, httperrors.NewNotEmptyError("dev_type is empty")
	}
	if !utils.IsInStringArray(input.DevType, api.VALID_PASSTHROUGH_TYPES) {
		if _, err := IsolatedDeviceModelManager.GetByDevType(input.DevType); err != nil {
			return input, httperrors.NewInputParameterError("device type %q not supported", input.DevType)
		}
	}

	input.StandaloneResourceCreateInput, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneResourceBaseManager.ValidateCreateData")
	}

	if input.HostId != "" && input.Addr != "" {
		if hasDevAddr, err := manager.hostHasDevAddr(input.HostId, input.Addr, input.MdevId); err != nil {
			return input, errors.Wrap(err, "check hostHasDevAddr")
		} else if hasDevAddr {
			return input, httperrors.NewBadRequestError("dev addr %s registed", input.Addr)
		}
	}

	// validate reserverd resource
	// inject default reserverd resource for gpu:
	if utils.IsInStringArray(input.DevType, []string{api.GPU_HPC_TYPE, api.GPU_VGA_TYPE}) {
		defaultCPU := 8        // 8
		defaultMem := 8192     // 8g
		defaultStore := 102400 // 100g
		if input.ReservedCpu == nil {
			input.ReservedCpu = &defaultCPU
		}
		if input.ReservedMemory == nil {
			input.ReservedMemory = &defaultMem
		}
		if input.ReservedStorage == nil {
			input.ReservedStorage = &defaultStore
		}
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
	if input.DevType != "" && input.DevType != self.DevType {
		if !utils.IsInStringArray(input.DevType, api.VALID_GPU_TYPES) {
			if _, err := IsolatedDeviceModelManager.GetByDevType(input.DevType); err != nil {
				return input, httperrors.NewInputParameterError("device type %q not support update", input.DevType)
			}
		} else {
			if !self.IsGPU() {
				return input, httperrors.NewInputParameterError("Can't update for device %q", self.DevType)
			}
		}
	}

	return input, nil
}

func (device *SIsolatedDevice) isolateDeviceNotifyForHost(ctx context.Context, userCred mcclient.TokenCredential, action notify.SAction) {
	model, err := HostManager.FetchById(device.HostId)
	if err != nil {
		return
	}
	host := model.(*SHost)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Action: action,
		Obj:    host,
		ObjDetailsDecorator: func(ctx context.Context, details *jsonutils.JSONDict) {
			details.Set("customize_details", jsonutils.Marshal(device))
		},
	})
}

func (device *SIsolatedDevice) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	device.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	device.isolateDeviceNotifyForHost(ctx, userCred, notify.ActionIsolatedDeviceCreate)
}

func (device *SIsolatedDevice) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	device.SStandaloneResourceBase.PostDelete(ctx, userCred)
	device.isolateDeviceNotifyForHost(ctx, userCred, notify.ActionIsolatedDeviceDelete)
}

func (device *SIsolatedDevice) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	HostManager.ClearSchedDescCache(device.HostId)
	device.isolateDeviceNotifyForHost(ctx, userCred, notify.ActionIsolatedDeviceUpdate)
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
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
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
	if len(query.DevicePath) > 0 {
		q = q.In("device_path", query.DevicePath)
	}
	if len(query.VendorDeviceId) > 0 {
		q = q.In("vendor_device_id", query.VendorDeviceId)
	}
	if len(query.NumaNode) > 0 {
		q = q.In("numa_node", query.NumaNode)
	}

	if !query.ShowBaremetalIsolatedDevices {
		sq := HostManager.Query("id").In("host_type", []string{api.HOST_TYPE_HYPERVISOR, api.HOST_TYPE_CONTAINER, api.HOST_TYPE_ZETTAKIT}).SubQuery()
		q = q.In("host_id", sq)
	}

	if query.GuestId != "" {
		obj, err := GuestManager.FetchByIdOrName(ctx, userCred, query.GuestId)
		if err != nil {
			return nil, errors.Wrapf(err, "Fetch guest by %q", query.GuestId)
		}
		q = q.Equals("guest_id", obj.GetId())
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

func (self *SIsolatedDevice) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if len(self.GuestId) > 0 {
		return httperrors.NewNotEmptyError("Isolated device used by server")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
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

func (manager *SIsolatedDeviceManager) fuzzyMatchModel(fuzzyStr string, devType string) *SIsolatedDevice {
	dev := SIsolatedDevice{}
	dev.SetModelManager(manager, &dev)

	q := manager.Query()
	if devType != "" {
		q = q.Equals("dev_type", devType)
	}

	qe := q.Equals("model", fuzzyStr)
	cnt, err := qe.CountWithError()
	if err != nil || cnt == 0 {
		qe = q.Contains("model", fuzzyStr)
	}

	err = qe.First(&dev)
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

func GetVendorByVendorDeviceId(vendorDeviceId string) string {
	parts := strings.Split(vendorDeviceId, ":")
	vendorId := parts[0]
	vendor, ok := ID_VENDOR_MAP[vendorId]
	if ok {
		return vendor
	} else {
		return vendorId
	}
}

func (self *SIsolatedDevice) IsGPU() bool {
	return strings.HasPrefix(self.DevType, "GPU") || sets.NewString(api.CONTAINER_GPU_TYPES...).Has(self.DevType)
}

func (manager *SIsolatedDeviceManager) parseDeviceInfo(userCred mcclient.TokenCredential, devConfig *api.IsolatedDeviceConfig) (*api.IsolatedDeviceConfig, error) {
	var devId, devType, devVendor string
	var matchDev *SIsolatedDevice

	devId = devConfig.Id
	matchDev = manager.fuzzyMatchModel(devConfig.Model, devConfig.DevType)
	devVendor = devConfig.Vendor
	devType = devConfig.DevType

	if len(devId) == 0 {
		if matchDev == nil {
			return nil, httperrors.NewNotFoundError("Not found matched device by model: %q, dev_type: %q", devConfig.Model, devConfig.DevType)
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
		devConfig.WireId = dev.WireId
		if dev.IsGPU() && len(devType) > 0 {
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

func (manager *SIsolatedDeviceManager) isValidDeviceInfo(config *api.IsolatedDeviceConfig) error {
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

func (manager *SIsolatedDeviceManager) isValidNicDeviceInfo(config *api.IsolatedDeviceConfig) error {
	return manager._isValidDeviceInfo(config, api.NIC_TYPE)
}

func (manager *SIsolatedDeviceManager) isValidNVMEDeviceInfo(config *api.IsolatedDeviceConfig) error {
	return manager._isValidDeviceInfo(config, api.NVME_PT_TYPE)
}

func (manager *SIsolatedDeviceManager) _isValidDeviceInfo(config *api.IsolatedDeviceConfig, devType string) error {
	if len(config.Id) > 0 {
		devObj, err := manager.FetchById(config.Id)
		if err != nil {
			return httperrors.NewResourceNotFoundError("IsolatedDevice %s not found", config.Id)
		}
		dev := devObj.(*SIsolatedDevice)
		if len(dev.GuestId) > 0 {
			return httperrors.NewConflictError("Isolated device already attached to another guest: %s", dev.GuestId)
		}
		if dev.DevType != devType {
			return httperrors.NewBadRequestError("IsolatedDevice is not device type %s", devType)
		}
	} else if config.DevType != "" && config.DevType != devType {
		return httperrors.NewBadRequestError("request dev type %s not match %s", config.DevType, devType)
	}
	return nil
}

func (manager *SIsolatedDeviceManager) attachHostDeviceToGuestByDesc(
	ctx context.Context, guest *SGuest, host *SHost, devConfig *api.IsolatedDeviceConfig,
	userCred mcclient.TokenCredential, usedDevMap map[string]*SIsolatedDevice, preferNumaNodes []int,
) error {
	if len(devConfig.Id) > 0 {
		return manager.attachSpecificDeviceToGuest(ctx, guest, devConfig, userCred)
	} else if len(devConfig.DevicePath) > 0 {
		return manager.attachHostDeviceToGuestByDevicePath(ctx, guest, host, devConfig, userCred, usedDevMap, preferNumaNodes)
	} else {
		return manager.attachHostDeviceToGuestByModel(ctx, guest, host, devConfig, userCred, usedDevMap, preferNumaNodes)
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
	return guest.attachIsolatedDevice(ctx, userCred, dev, devConfig.NetworkIndex, devConfig.DiskIndex)
}

func (manager *SIsolatedDeviceManager) attachHostDeviceToGuestByDevicePath(ctx context.Context, guest *SGuest, host *SHost, devConfig *api.IsolatedDeviceConfig, userCred mcclient.TokenCredential, usedDevMap map[string]*SIsolatedDevice, preferNumaNodes []int) error {
	if len(devConfig.Model) == 0 || len(devConfig.DevicePath) == 0 {
		return fmt.Errorf("Model or DevicePath is empty: %#v", devConfig)
	}
	// if dev type is not nic, wire is empty string
	devs, err := manager.findHostUnusedByDevAttr(devConfig.Model, "device_path", devConfig.DevicePath, host.Id, devConfig.WireId)
	if err != nil || len(devs) == 0 {
		return fmt.Errorf("Can't found model %s device_path %s on host %s", devConfig.Model, devConfig.DevicePath, host.Id)
	}
	var selectedDev SIsolatedDevice
	for i := range devs {
		if _, ok := usedDevMap[devs[i].DevicePath]; !ok {
			selectedDev = devs[i]
			usedDevMap[devs[i].DevicePath] = &selectedDev
		}
	}
	if selectedDev.Id == "" {
		selectedDev = devs[0]
	}
	return guest.attachIsolatedDevice(ctx, userCred, &selectedDev, devConfig.NetworkIndex, devConfig.DiskIndex)
}

type GroupDevs struct {
	DevPath string
	Devs    []SIsolatedDevice
}

type SorttedGroupDevs []*GroupDevs

func (pq SorttedGroupDevs) Len() int { return len(pq) }

func (pq SorttedGroupDevs) Less(i, j int) bool {
	return len(pq[i].Devs) > len(pq[j].Devs)
}

func (pq SorttedGroupDevs) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *SorttedGroupDevs) Push(item interface{}) {
	*pq = append(*pq, item.(*GroupDevs))
}

func (pq *SorttedGroupDevs) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*pq = old[0 : n-1]
	return item
}

type SNodeIsolateDevicesInfo struct {
	TotalDevCount int
	ReservedRate  float32
}

func (manager *SIsolatedDeviceManager) getDevNodesUsedRate(
	ctx context.Context, host *SHost, devConfig *api.IsolatedDeviceConfig, topo *hostapi.HostTopology,
) (map[string]SNodeIsolateDevicesInfo, error) {
	devs, err := manager.findHostDevsByDevConfig(devConfig.Model, devConfig.DevType, host.Id, devConfig.WireId)
	if err != nil || len(devs) == 0 {
		return nil, fmt.Errorf("Can't found model %s on host %s", devConfig.Model, host.Id)
	}
	mapDevs := map[string][]SIsolatedDevice{}
	for i := range devs {
		dev := devs[i]
		devPath := dev.DevicePath
		var gdevs []SIsolatedDevice

		gdevs, ok := mapDevs[devPath]
		if !ok {
			gdevs = []SIsolatedDevice{dev}
		} else {
			gdevs = append(gdevs, dev)
		}
		mapDevs[devPath] = gdevs
	}
	nodesGroupDevs := map[string]SorttedGroupDevs{}
	for devPath, mappedDevs := range mapDevs {
		numaNode := strconv.Itoa(int(mappedDevs[0].NumaNode))
		if _, ok := nodesGroupDevs[numaNode]; ok {
			nodesGroupDevs[numaNode] = append(nodesGroupDevs[numaNode], &GroupDevs{
				DevPath: devPath,
				Devs:    mappedDevs,
			})
		} else {
			groupDevs := make(SorttedGroupDevs, 0)
			nodesGroupDevs[numaNode] = append(groupDevs, &GroupDevs{
				DevPath: devPath,
				Devs:    mappedDevs,
			})
		}
	}

	reserveRate := map[string]float32{}
	reserveRateStr := host.GetMetadata(ctx, api.HOSTMETA_RESERVED_CPUS_RATE, nil)
	reserveRateJ, err := jsonutils.ParseString(reserveRateStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse reserveRateStr")
	}
	err = reserveRateJ.Unmarshal(&reserveRate)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal reserveRateStr")
	}

	nodeNoDevIds := map[int]int{}
	for i := range topo.Nodes {
		nodeId := strconv.Itoa(topo.Nodes[i].ID)
		if _, ok := nodesGroupDevs[nodeId]; !ok {
			nodeInt, _ := strconv.Atoi(nodeId)
			nodeNoDevIds[nodeInt] = -1
		}
	}
	//
	//for nodeId, _ := range reserveRate {
	//	if _, ok := nodesGroupDevs[nodeId]; !ok {
	//		nodeInt, _ := strconv.Atoi(nodeId)
	//		nodeNoDevIds[nodeInt] = -1
	//	}
	//}

	reserveNodes := map[string][]string{}
	for i := range topo.Nodes {
		if _, ok := nodeNoDevIds[topo.Nodes[i].ID]; ok {
			minDistance := int(math.MaxInt16)
			selectNodeId := ""
			for nodeId, _ := range nodesGroupDevs {
				nodeInt, _ := strconv.Atoi(nodeId)
				if topo.Nodes[i].Distances[nodeInt] < minDistance {
					selectNodeId = strconv.Itoa(nodeInt)
					minDistance = topo.Nodes[i].Distances[nodeInt]
				}
			}
			noDevNodeId := strconv.Itoa(topo.Nodes[i].ID)
			log.Debugf("node %s select node %s", noDevNodeId, selectNodeId)
			if nodes, ok := reserveNodes[selectNodeId]; ok {
				reserveNodes[selectNodeId] = append(nodes, noDevNodeId)
			} else {
				reserveNodes[selectNodeId] = []string{noDevNodeId}
			}
		}
	}
	reserveRates := map[string]SNodeIsolateDevicesInfo{}
	for nodeId, devGroups := range nodesGroupDevs {
		nodeCnt := 1
		nodeReserveRate := reserveRate[nodeId]
		if nodes, ok := reserveNodes[nodeId]; ok {
			for i := range nodes {
				nodeReserveRate += reserveRate[nodes[i]]
				nodeCnt += 1
			}
		}
		nodeReserveRate = nodeReserveRate / float32(nodeCnt)
		devCnt := 0
		for i := range devGroups {
			devCnt += len(devGroups[i].Devs)
		}
		reserveRates[nodeId] = SNodeIsolateDevicesInfo{
			TotalDevCount: devCnt,
			ReservedRate:  nodeReserveRate,
		}
		log.Debugf("node %v nodeCnt %v nodeReserveRate %v", nodeId, nodeCnt, nodeReserveRate)
	}
	return reserveRates, nil
}

func (manager *SIsolatedDeviceManager) attachHostDeviceToGuestByModel(
	ctx context.Context, guest *SGuest, host *SHost, devConfig *api.IsolatedDeviceConfig,
	userCred mcclient.TokenCredential, usedDevMap map[string]*SIsolatedDevice, preferNumaNodes []int,
) error {
	if len(devConfig.Model) == 0 {
		return fmt.Errorf("Not found model from info: %#v", devConfig)
	}
	// if dev type is not nic, wire is empty string
	devs, err := manager.findHostUnusedByDevConfig(devConfig.Model, devConfig.DevType, host.Id, devConfig.WireId)
	if err != nil || len(devs) == 0 {
		return fmt.Errorf("Can't found model %s on host %s", devConfig.Model, host.Id)
	}
	// 1. group devices by device_path and numa nodes
	//groupDevs := make(SorttedGroupDevs, 0)
	mapDevs := map[string][]SIsolatedDevice{}
	for i := range devs {
		dev := devs[i]
		devPath := dev.DevicePath
		var gdevs []SIsolatedDevice

		gdevs, ok := mapDevs[devPath]
		if !ok {
			gdevs = []SIsolatedDevice{dev}
		} else {
			gdevs = append(gdevs, dev)
		}
		mapDevs[devPath] = gdevs
	}

	var groupDevs SorttedGroupDevs
	if len(preferNumaNodes) > 0 {
		groupDevs = make(SorttedGroupDevs, 0)
		for devPath, mappedDevs := range mapDevs {
			groupDevs = append(groupDevs, &GroupDevs{
				DevPath: devPath,
				Devs:    mappedDevs,
			})
		}
	} else {
		nodesGroupDevs := map[int8]SorttedGroupDevs{}
		for devPath, mappedDevs := range mapDevs {
			numaNode := mappedDevs[0].NumaNode
			if _, ok := nodesGroupDevs[numaNode]; ok {
				nodesGroupDevs[numaNode] = append(nodesGroupDevs[numaNode], &GroupDevs{
					DevPath: devPath,
					Devs:    mappedDevs,
				})
			} else {
				groupDevs := make(SorttedGroupDevs, 0)
				nodesGroupDevs[numaNode] = append(groupDevs, &GroupDevs{
					DevPath: devPath,
					Devs:    mappedDevs,
				})
			}
		}

		var selectedNode int8 = -1
		if len(nodesGroupDevs) == 1 {
			for nodeId := range nodesGroupDevs {
				selectedNode = nodeId
			}
		} else {
			reservedCpusStr := host.GetMetadata(ctx, api.HOSTMETA_RESERVED_CPUS_INFO, nil)
			if len(reservedCpusStr) > 0 {
				topoObj, err := host.SysInfo.Get("topology")
				if err != nil {
					return errors.Wrap(err, "get topology from host sys_info")
				}
				topo := new(hostapi.HostTopology)
				if err := topoObj.Unmarshal(topo); err != nil {
					return errors.Wrap(err, "Unmarshal host topology struct")
				}
				nodesReserveRate, err := manager.getDevNodesUsedRate(ctx, host, devConfig, topo)
				if err != nil {
					return err
				}
				var selectedNodeUtil float32 = 1.0
				for nodeId, gds := range nodesGroupDevs {
					freeDevCnt := 0
					for i := range gds {
						freeDevCnt += len(gds[i].Devs)
					}

					nodeTotalCnt := nodesReserveRate[strconv.Itoa(int(nodeId))].TotalDevCount
					usedDevCnt := nodeTotalCnt - freeDevCnt

					nodeReserveRate := nodesReserveRate[strconv.Itoa(int(nodeId))].ReservedRate
					nodeCnt := (1 - nodeReserveRate) * float32(nodeTotalCnt)
					nodeutil := float32(usedDevCnt) / nodeCnt
					log.Debugf("selectedNodeUtil node %v util %v usedDevCnt %v totalDevCnt %v", nodeId, nodeutil, usedDevCnt, nodeCnt)
					if nodeutil < selectedNodeUtil {
						selectedNodeUtil = nodeutil
						selectedNode = nodeId
					}
				}
			} else {
				var selectedNodeDevCnt = 0
				for nodeId, gds := range nodesGroupDevs {
					devCnt := 0
					for i := range gds {
						devCnt += len(gds[i].Devs)
					}
					if devCnt > selectedNodeDevCnt {
						selectedNodeDevCnt = devCnt
						selectedNode = nodeId
					}
				}
			}
		}
		log.Debugf("selectedNodeUtil node %v", selectedNode)
		groupDevs = nodesGroupDevs[selectedNode]
	}
	sort.Sort(groupDevs)

	var selectedDev *SIsolatedDevice
	if len(preferNumaNodes) > 0 {
		topoObj, err := host.SysInfo.Get("topology")
		if err != nil {
			return errors.Wrap(err, "get topology from host sys_info")
		}
		hostTopo := new(hostapi.HostTopology)
		if err := topoObj.Unmarshal(hostTopo); err != nil {
			return errors.Wrap(err, "Unmarshal host topology struct")
		}

		if len(groupDevs) == 1 && groupDevs[0].DevPath == "" {
			minDistancesDevIdx := -1
			minDistances := math.MaxInt32
			for i := range groupDevs[0].Devs {
				if groupDevs[0].Devs[i].NumaNode < 0 {
					continue
				}
				devNodeId := groupDevs[0].Devs[i].NumaNode
				for j := range hostTopo.Nodes {
					if hostTopo.Nodes[j].ID == int(devNodeId) {
						devDistance := 0
						for k := range preferNumaNodes {
							devDistance += hostTopo.Nodes[j].Distances[preferNumaNodes[k]]
						}
						if devDistance < minDistances {
							minDistances = devDistance
							minDistancesDevIdx = i
						}
					}
				}
			}
			if minDistancesDevIdx >= 0 {
				selectedDev = &groupDevs[0].Devs[minDistancesDevIdx]
			}
		} else {
			minDistancesGroupIdx := -1
			minDistances := math.MaxInt32
			log.Infof("devtype %s grouplength %d", groupDevs[0].Devs[0].DevType, len(groupDevs))

			for i := range groupDevs {
				if groupDevs[i].Devs[0].NumaNode < 0 {
					continue
				}
				devNodeId := groupDevs[i].Devs[0].NumaNode
				for j := range hostTopo.Nodes {
					if hostTopo.Nodes[j].ID == int(devNodeId) {
						devDistance := 0
						for k := range preferNumaNodes {
							devDistance += hostTopo.Nodes[j].Distances[preferNumaNodes[k]]
						}
						if devDistance < minDistances {
							minDistances = devDistance
							minDistancesGroupIdx = i
						}
					}
				}
			}
			if minDistancesGroupIdx >= 0 {
				selectedDev = &groupDevs[minDistancesGroupIdx].Devs[0]
			}
		}
	}
	if selectedDev == nil {
		for i := range groupDevs {
			if groupDevs[i].DevPath != "" {
				for j := range groupDevs[i].Devs {
					dev := groupDevs[i].Devs[j]
					devAddr := strings.Split(dev.Addr, "-")[0]
					if _, ok := usedDevMap[devAddr]; ok {
						continue
					} else {
						selectedDev = &groupDevs[i].Devs[j]
						break
					}
				}
			} else {
				dev := groupDevs[i].Devs[0]
				devAddr := strings.Split(dev.Addr, "-")[0]
				if _, ok := usedDevMap[devAddr]; ok {
					continue
				} else {
					selectedDev = &groupDevs[i].Devs[0]
					break
				}
			}
			if selectedDev != nil {
				break
			}
		}
	}

	if selectedDev == nil {
		selectedDev = &groupDevs[0].Devs[0]
	}
	devAddr := strings.Split(selectedDev.Addr, "-")[0]
	usedDevMap[devAddr] = selectedDev

	return guest.attachIsolatedDevice(ctx, userCred, selectedDev, devConfig.NetworkIndex, devConfig.DiskIndex)
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
		sqlchemy.Equals(q.Field("dev_type"), api.GPU_HPC_TYPE),
		sqlchemy.Equals(q.Field("dev_type"), api.GPU_VGA_TYPE)))
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

func (manager *SIsolatedDeviceManager) FindUnusedNicWiresByModel(modelName string) ([]string, error) {
	q := manager.Query().IsNullOrEmpty("guest_id").Equals("dev_type", api.NIC_TYPE)
	if len(modelName) > 0 {
		q = q.Equals("model", modelName)
	}
	q = q.GroupBy("wire_id")
	devs := make([]SIsolatedDevice, 0)
	err := q.All(&devs)
	if err != nil {
		return nil, err
	}
	wires := make([]string, len(devs))
	for i := 0; i < len(devs); i++ {
		wires[i] = devs[i].WireId
	}
	return wires, err
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

func (manager *SIsolatedDeviceManager) findHostUnusedByDevConfig(model, devType, hostId, wireId string) ([]SIsolatedDevice, error) {
	return manager.findHostUnusedByDevAttr(model, "dev_type", devType, hostId, wireId)
}

func (manager *SIsolatedDeviceManager) findHostDevsByDevConfig(model, devType, hostId, wireId string) ([]SIsolatedDevice, error) {
	return manager.findHostDevsByDevAttr(model, "dev_type", devType, hostId, wireId)
}
func (manager *SIsolatedDeviceManager) findHostDevsByDevAttr(model, attrKey, attrVal, hostId, wireId string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.Query()
	q = q.Equals("model", model).Equals("host_id", hostId)
	if attrVal != "" {
		q.Equals(attrKey, attrVal)
	}
	if wireId != "" {
		wire := WireManager.FetchWireById(wireId)
		if wire.VpcId == api.DEFAULT_VPC_ID {
			q = q.Equals("wire_id", wireId)
		}
	}
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) findHostUnusedByDevAttr(model, attrKey, attrVal, hostId, wireId string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.findUnusedQuery()
	q = q.Equals("model", model).Equals("host_id", hostId)
	if attrVal != "" {
		q.Equals(attrKey, attrVal)
	}
	if wireId != "" {
		wire := WireManager.FetchWireById(wireId)
		if wire.VpcId == api.DEFAULT_VPC_ID {
			q = q.Equals("wire_id", wireId)
		}
	}
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) ReleaseGPUDevicesOfGuest(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential) error {
	devs := manager.findAttachedDevicesOfGuest(guest)
	if devs == nil {
		return fmt.Errorf("fail to find attached devices")
	}
	for _, dev := range devs {
		if !utils.IsInStringArray(dev.DevType, api.VALID_GPU_TYPES) {
			continue
		}
		_, err := db.Update(&dev, func() error {
			dev.GuestId = ""
			dev.NetworkIndex = -1
			dev.DiskIndex = -1
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

func (manager *SIsolatedDeviceManager) ReleaseDevicesOfGuest(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential) error {
	devs := manager.findAttachedDevicesOfGuest(guest)
	if devs == nil {
		return fmt.Errorf("fail to find attached devices")
	}
	for _, dev := range devs {
		_, err := db.Update(&dev, func() error {
			dev.GuestId = ""
			dev.NetworkIndex = -1
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
	ctx context.Context,
	scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, devType []string, hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	rangeObjs []db.IStandaloneModel,
	policyResult rbacutils.SPolicyResult,
) *sqlchemy.SQuery {
	hq := HostManager.Query()
	if scope == rbacscope.ScopeDomain {
		hq = hq.Filter(sqlchemy.Equals(hq.Field("domain_id"), ownerId.GetProjectDomainId()))
	}
	hq = db.ObjectIdQueryWithPolicyResult(ctx, hq, HostManager, policyResult)
	hosts := hq.SubQuery()
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
	ctx context.Context,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	devType,
	hostTypes []string,
	resourceTypes []string,
	providers []string,
	brands []string,
	cloudEnv string,
	rangeObjs []db.IStandaloneModel,
	policyResult rbacutils.SPolicyResult,
) (int, error) {
	return manager.totalCountQ(
		ctx,
		scope,
		ownerId,
		devType,
		hostTypes,
		resourceTypes,
		providers,
		brands,
		cloudEnv,
		rangeObjs,
		policyResult,
	).CountWithError()
}

func (manager *SIsolatedDeviceManager) TotalCount(
	ctx context.Context,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	hostType []string,
	resourceTypes []string,
	providers []string,
	brands []string,
	cloudEnv string,
	rangeObjs []db.IStandaloneModel,
	policyResult rbacutils.SPolicyResult,
) (IsolatedDeviceCountStat, error) {
	stat := IsolatedDeviceCountStat{}
	devCnt, err := manager.totalCount(
		ctx,
		scope, ownerId, nil, hostType, resourceTypes,
		providers, brands, cloudEnv,
		rangeObjs, policyResult)
	if err != nil {
		return stat, err
	}
	gpuCnt, err := manager.totalCount(
		ctx,
		scope, ownerId, VALID_GPU_TYPES, hostType, resourceTypes,
		providers, brands, cloudEnv,
		rangeObjs, policyResult)
	if err != nil {
		return stat, err
	}
	stat.Devices = devCnt
	stat.Gpus = gpuCnt
	return stat, nil
}

func (self *SIsolatedDevice) getDesc() *api.IsolatedDeviceJsonDesc {
	return &api.IsolatedDeviceJsonDesc{
		Id:                  self.Id,
		DevType:             self.DevType,
		Model:               self.Model,
		Addr:                self.Addr,
		VendorDeviceId:      self.VendorDeviceId,
		Vendor:              self.getVendor(),
		NetworkIndex:        self.NetworkIndex,
		OvsOffloadInterface: self.OvsOffloadInterface,
		DiskIndex:           self.DiskIndex,
		NvmeSizeMB:          self.NvmeSizeMB,
		MdevId:              self.MdevId,
		NumaNode:            self.NumaNode,
	}
}

func (man *SIsolatedDeviceManager) GetSpecShouldCheckStatus(query *jsonutils.JSONDict) (bool, error) {
	return true, nil
}

func (man *SIsolatedDeviceManager) BatchGetModelSpecs(statusCheck bool) (jsonutils.JSONObject, error) {
	hostQ := HostManager.Query()
	q := man.Query("vendor_device_id", "model", "dev_type")
	if statusCheck {
		q = q.IsNullOrEmpty("guest_id")
		hostQ = hostQ.Equals("status", api.BAREMETAL_RUNNING).IsTrue("enabled").
			In("host_type", []string{api.HOST_TYPE_HYPERVISOR, api.HOST_TYPE_CONTAINER, api.HOST_TYPE_ZETTAKIT})
	}
	hostSQ := hostQ.SubQuery()
	q.Join(hostSQ, sqlchemy.Equals(q.Field("host_id"), hostSQ.Field("id")))

	q.AppendField(hostSQ.Field("host_type"))
	q.GroupBy(hostSQ.Field("host_type"), q.Field("vendor_device_id"), q.Field("model"), q.Field("dev_type"))
	q.AppendField(sqlchemy.COUNT("*"))

	rows, err := q.Rows()
	if err != nil {
		return nil, errors.Wrap(err, "failed get specs")
	}
	defer rows.Close()
	res := jsonutils.NewDict()

	for rows.Next() {
		var hostType, vendorDeviceId, m, t string
		var count int
		if err := rows.Scan(&vendorDeviceId, &m, &t, &hostType, &count); err != nil {
			return nil, errors.Wrap(err, "get model spec scan rows")
		}
		vendor := GetVendorByVendorDeviceId(vendorDeviceId)
		specKeys := man.getSpecKeys(vendor, m, t)
		specKey := GetSpecIdentKey(specKeys)
		spec := man.getSpecByRows(hostType, vendorDeviceId, m, t, &count)
		res.Set(specKey, spec)
	}

	return res, nil
}

func (man *SIsolatedDeviceManager) getSpecByRows(hostType, vendorDeviceId, model, devType string, count *int) *jsonutils.JSONDict {
	var vdev bool
	var hypervisor string
	if utils.IsInStringArray(devType, api.VITRUAL_DEVICE_TYPES) {
		vdev = true
	}
	if utils.IsInStringArray(devType, api.VALID_CONTAINER_DEVICE_TYPES) {
		hypervisor = api.HYPERVISOR_POD
	} else {
		hypervisor = api.HYPERVISOR_KVM
	}
	if hostType == api.HOST_TYPE_ZETTAKIT {
		hypervisor = api.HYPERVISOR_ZETTAKIT
	}

	ret := jsonutils.NewDict()
	ret.Set("virtual_dev", jsonutils.NewBool(vdev))
	ret.Set("hypervisor", jsonutils.NewString(hypervisor))
	ret.Set("dev_type", jsonutils.NewString(devType))
	ret.Set("model", jsonutils.NewString(model))
	ret.Set("pci_id", jsonutils.NewString(vendorDeviceId))
	ret.Set("vendor", jsonutils.NewString(GetVendorByVendorDeviceId(vendorDeviceId)))
	if count != nil {
		ret.Set("count", jsonutils.NewInt(int64(*count)))
	}

	return ret
}

type GpuSpec struct {
	DevType string `json:"dev_type,allowempty"`
	Model   string `json:"model,allowempty"`
	Amount  string `json:"amount,allowemtpy"`
	Vendor  string `json:"vendor,allowempty"`
	PciId   string `json:"pci_id,allowempty"`
}

func (self *SIsolatedDevice) GetSpec(statusCheck bool) *jsonutils.JSONDict {
	host := self.getHost()
	if statusCheck {
		if len(self.GuestId) > 0 {
			return nil
		}
		if host.Status != api.BAREMETAL_RUNNING || !host.GetEnabled() ||
			(host.HostType != api.HOST_TYPE_HYPERVISOR && host.HostType != api.HOST_TYPE_CONTAINER && host.HostType != api.HOST_TYPE_ZETTAKIT) {
			return nil
		}
	}
	return IsolatedDeviceManager.getSpecByRows(host.HostType, self.VendorDeviceId, self.Model, self.DevType, nil)
}

func (self *SIsolatedDevice) GetGpuSpec() *GpuSpec {
	return &GpuSpec{
		DevType: self.DevType,
		Model:   self.Model,
		PciId:   self.VendorDeviceId,
		Vendor:  self.getVendor(),
		Amount:  "1",
	}
}

func (man *SIsolatedDeviceManager) GetSpecIdent(spec *jsonutils.JSONDict) []string {
	devType, _ := spec.GetString("dev_type")
	vendor, _ := spec.GetString("vendor")
	model, _ := spec.GetString("model")
	return man.getSpecKeys(vendor, model, devType)
}

func (man *SIsolatedDeviceManager) getSpecKeys(vendor, model, devType string) []string {
	keys := []string{
		fmt.Sprintf("type:%s", devType),
		fmt.Sprintf("vendor:%s", vendor),
		fmt.Sprintf("model:%s", model),
	}
	return keys
}

func (self *SIsolatedDevice) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()
	desc.Update(jsonutils.Marshal(self.getDesc()))
	desc.Add(jsonutils.NewString(IsolatedDeviceManager.Keyword()), "res_name")
	return desc
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

func (manager *SIsolatedDeviceManager) GetAllDevsOnHost(hostId string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.Query().Equals("host_id", hostId)
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) GetUnusedDevsOnHost(hostId string, model string, count int) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.Query().Equals("host_id", hostId).Equals("model", model).IsNullOrEmpty("guest_id")
	if count > 0 {
		q = q.Limit(count)
	}
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) hostHasDevAddr(hostId, addr, mdevId string) (bool, error) {
	cnt, err := manager.Query().Equals("addr", addr).Equals("mdev_id", mdevId).
		Equals("host_id", hostId).CountWithError()
	if err != nil {
		return false, err
	}
	return cnt != 0, nil
}

func (manager *SIsolatedDeviceManager) CheckModelIsEmpty(model, vendor, device, devType string) (bool, error) {
	cnt, err := manager.Query().Equals("model", model).
		Equals("dev_type", devType).
		Equals("vendor_device_id", fmt.Sprintf("%s:%s", vendor, device)).
		IsNotEmpty("guest_id").CountWithError()
	if err != nil {
		return false, err
	}
	return cnt == 0, nil
}

func (manager *SIsolatedDeviceManager) GetHostsByModel(model, vendor, device, devType string) ([]string, error) {
	q := manager.Query("host_id").Equals("model", model).
		Equals("dev_type", devType).
		Equals("vendor_device_id", fmt.Sprintf("%s:%s", vendor, device)).GroupBy("host_id")

	rows, err := q.Rows()
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "q.Rows")
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	ret := make([]string, 0)
	for rows.Next() {
		var hostId string
		err = rows.Scan(&hostId)
		if err != nil {
			return nil, errors.Wrap(err, "rows.Scan")
		}
		ret = append(ret, hostId)
	}
	return ret, nil
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

func (manager *SIsolatedDeviceManager) NamespaceScope() rbacscope.TRbacScope {
	if consts.IsDomainizedNamespace() {
		return rbacscope.ScopeDomain
	} else {
		return rbacscope.ScopeSystem
	}
}

func (manager *SIsolatedDeviceManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (manager *SIsolatedDeviceManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacscope.ScopeProject, rbacscope.ScopeDomain:
			hostsQ := HostManager.Query("id")
			hostsQ = HostManager.FilterByOwner(ctx, hostsQ, HostManager, userCred, owner, scope)
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

func (model *SIsolatedDevice) syncWithCloudIsolateDevice(ctx context.Context, userCred mcclient.TokenCredential, dev cloudprovider.IsolateDevice) error {
	_, err := db.Update(model, func() error {
		model.Name = dev.GetName()
		model.Model = dev.GetModel()
		model.Addr = dev.GetAddr()
		model.DevType = dev.GetDevType()
		model.NumaNode = dev.GetNumaNode()
		model.VendorDeviceId = dev.GetVendorDeviceId()
		return nil
	})
	return err
}

func (model *SIsolatedDevice) SetNetworkIndex(idx int) error {
	_, err := db.Update(model, func() error {
		model.NetworkIndex = idx
		return nil
	})
	return err
}
