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
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
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

type SIsolatedDeviceManager struct {
	db.SStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	db.SSharableBaseResourceManager
	SHostResourceBaseManager
}

var IsolatedDeviceManager *SIsolatedDeviceManager

const (
	isolatedDeviceInitializeDataObjType = "system"
	isolatedDeviceInitializeDataObjId   = "compute_isolated_device_initialize_data"
	isolatedDeviceInitializeDataKey     = "__initialized"
)

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
	db.SSharableBaseResource `"is_public->create":"domain_optional" "public_scope->create":"domain_optional"`
	SHostResourceBase        `width:"36" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required"`

	// # PCI / GPU / USB / NIC ...
	// 设备类型
	DevType string `width:"128" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required" update:"domain"`
	// EXCLUSIVE / SRIOV / MPS / HAMI / SHARE / MIG
	SharingMode string `width:"36" charset:"ascii" nullable:"true" index:"true" list:"domain" update:"domain" create:"domain_required"`
	// Device is hot pluggable
	HotPluggable bool `default:"false" list:"domain" create:"domain_optional" update:"domain"`
	// # Specific device name read from lspci command, e.g. `Tesla K40m` ...
	Model string `width:"512" charset:"ascii" nullable:"false" default:"" index:"true" list:"domain" create:"domain_required" update:"domain"`

	// 云主机Id, keep for backward compatibility
	// swagger:deprecated
	GuestId string `width:"36" charset:"ascii" nullable:"true"`
	// guest network index, keep for backward compatibility
	// swagger:deprecated
	NetworkIndex int `nullable:"true" default:"-1"`

	// Nic wire id
	WireId string `width:"36" charset:"ascii" nullable:"true" index:"true" list:"domain" update:"domain" create:"domain_optional"`
	// Offload interface name
	OvsOffloadInterface string `width:"16" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	// Is infiniband nic
	IsInfinibandNic bool `nullable:"false" default:"false" list:"user" create:"optional"`
	// NVME disk size
	NvmeSizeMB int `nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	// guest disk index, keep for backward compatibility
	// swagger:deprecated
	DiskIndex int8 `nullable:"true" default:"-1"`

	// # pci address of `Bus:Device.Function` format, or usb bus address of `bus:addr:port`
	Addr       string `width:"16" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	DevicePath string `width:"128" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"optional"`

	// GPU card path, like /dev/dri/cardX
	CardPath string `width:"128" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"optional"`
	// GPU render path, like /dev/dri/renderDX
	RenderPath string `width:"128" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"optional"`
	// Nvidia GPU index
	Index int `nullable:"true" default:"-1" list:"user" update:"domain"`
	// Nvidia GPU minor number, parsing from /proc/driver/nvidia/gpus/*/information
	DeviceMinor int `nullable:"true" default:"-1" list:"user" update:"domain"`

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

	// On-device memory in MiB (NVIDIA GPU VRAM via `nvidia-smi memory.total`,
	// or per-slice quota for MPS share mode). 0 means unknown / not applicable.
	MemorySize int `nullable:"true" default:"0" list:"domain" update:"domain" create:"domain_optional"`
	// some of isolated device type support virtual num, like NVIDIA_GPU_SHARE, NVIDIA_MPS
	VirtualNum int `nullable:"true" default:"1" list:"user" update:"domain" create:"domain_optional"`
}

func (manager *SIsolatedDeviceManager) GetIVirtualModelManager() db.IVirtualModelManager {
	return manager
}

func (manager *SIsolatedDeviceManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	return []db.SScopeResourceCount{}, nil
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
	if !utils.IsInStringArray(input.DevType, api.VALID_TYPES) {
		if _, err := IsolatedDeviceModelManager.GetByDevType(input.DevType); err != nil {
			return input, httperrors.NewInputParameterError("device type %q is not supported", input.DevType)
		}
	}

	if !utils.IsInStringArray(input.SharingMode, api.VAILD_SHARING_MODES) {
		return input, httperrors.NewNotEmptyError("sharing_mode %s is not valid", input.SharingMode)
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
	if host.HostType == api.HOST_TYPE_KVM && input.SharingMode == api.DEVICE_SHARING_MODE_EXCLUSIVE && input.DevType == api.GPU_TYPE {
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
		q = manager.queryWithoutGuest(q)
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
	if len(query.Vendor) > 0 {
		conds := make([]sqlchemy.ICondition, 0, len(query.Vendor))
		for _, v := range query.Vendor {
			conds = append(conds, sqlchemy.Startswith(q.Field("vendor_device_id"), vendorDeviceIdPrefixForFilter(v)))
		}
		q = q.Filter(sqlchemy.OR(conds...))
	}
	if len(query.NumaNode) > 0 {
		q = q.In("numa_node", query.NumaNode)
	}
	if query.Index != nil && *query.Index >= 0 {
		q = q.Equals("index", query.Index)
	}
	if query.DeviceMinor != nil && *query.DeviceMinor >= 0 {
		q = q.Equals("device_minor", query.DeviceMinor)
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
		gq := GuestIsolatedDeviceManager.Query().Equals("guest_id", obj.GetId()).SubQuery()
		q = q.Join(gq, sqlchemy.Equals(q.Field("id"), gq.Field("isolated_device_id")))
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
		gq := GuestIsolatedDeviceManager.Query().SubQuery()
		q = q.Join(gq, sqlchemy.Equals(q.Field("id"), gq.Field("isolated_device_id")))
		q.LeftJoin(guestNameQuery, sqlchemy.Equals(gq.Field("guest_id"), guestNameQuery.Field("id")))
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
	gdevs, err := self.GetAllGuestIsolatedDevices()
	if err != nil {
		return err
	}
	if len(gdevs) > 0 {
		return httperrors.NewNotEmptyError("Isolated device used by server")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SIsolatedDevice) getDetailedString() string {
	return fmt.Sprintf("%s:%s/%s/%s", self.Addr, self.Model, self.VendorDeviceId, self.DevType)
}

func (manager *SIsolatedDeviceManager) fuzzyMatchModel(fuzzyStr, devType, sharingMode string) *SIsolatedDevice {
	dev := SIsolatedDevice{}
	dev.SetModelManager(manager, &dev)

	q := manager.Query()
	if devType != "" {
		q = q.Equals("dev_type", devType)
	}
	if sharingMode != "" {
		q = q.Equals("sharing_mode", sharingMode)
	}

	if fuzzyStr != "" {
		qe := q.Equals("model", fuzzyStr)
		cnt, err := qe.CountWithError()
		if err != nil || cnt == 0 {
			qe = q.Contains("model", fuzzyStr)
		}
		q = qe
	}

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
	vendor, ok := api.ID_VENDOR_MAP[vendorId]
	if ok {
		return vendor
	} else {
		return vendorId
	}
}

func GetVendorByVendorDeviceId(vendorDeviceId string) string {
	parts := strings.Split(vendorDeviceId, ":")
	vendorId := parts[0]
	vendor, ok := api.ID_VENDOR_MAP[vendorId]
	if ok {
		return vendor
	} else {
		return vendorId
	}
}

func vendorDeviceIdPrefixForFilter(vendor string) string {
	return resolveVendorIdForFilter(vendor) + ":"
}

func resolveVendorIdForFilter(vendor string) string {
	if id, ok := api.VENDOR_ID_MAP[vendor]; ok {
		return id
	}
	for name, id := range api.VENDOR_ID_MAP {
		if strings.EqualFold(name, vendor) {
			return id
		}
	}
	return strings.ToLower(vendor)
}

func (self *SIsolatedDevice) IsGPU() bool {
	return self.DevType == api.GPU_TYPE
}

func (manager *SIsolatedDeviceManager) parseDeviceInfo(userCred mcclient.TokenCredential, devConfig *api.IsolatedDeviceConfig) (*api.IsolatedDeviceConfig, error) {
	var devId, devType, devVendor string
	var matchDev *SIsolatedDevice

	devId = devConfig.Id
	matchDev = manager.fuzzyMatchModel(devConfig.Model, devConfig.DevType, devConfig.SharingMode)
	devVendor = devConfig.Vendor
	devType = devConfig.DevType

	if len(devId) == 0 {
		if matchDev == nil {
			return nil, httperrors.NewNotFoundError("Not found matched device by model: %q, dev_type: %q", devConfig.Model, devConfig.DevType)
		}
		devConfig.Model = matchDev.Model
		devConfig.SharingMode = matchDev.SharingMode

		if len(devVendor) > 0 {
			vendorId, ok := api.VENDOR_ID_MAP[devVendor]
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
		devConfig.SharingMode = dev.SharingMode
		devConfig.Vendor = dev.getVendor()
		devConfig.WireId = dev.WireId
		if devType != "" && devType != dev.DevType {
			return nil, fmt.Errorf("request dev_type %s not match dev %s type %s", devType, dev.Id, dev.DevType)
		}
	}
	if len(devType) > 0 {
		devConfig.DevType = devType
	}
	if devConfig.SharingMode == api.DEVICE_SHARING_MODE_HAMI {
		if devConfig.MemoryRequest <= 0 {
			return nil, httperrors.NewBadRequestError("dev sharing_mode %s must give memory request", devConfig.SharingMode)
		}
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
		if dev.IsFull() {
			return httperrors.NewConflictError("Isolated device already attached")
		}
	}
	if config.GpuType != "" && !utils.IsInStringArray(config.GpuType, []string{api.GPU_HPC, api.GPU_VGA}) {
		return httperrors.NewInputParameterError("Input gpu_type %s not valid", config.GpuType)
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
		if dev.IsFull() {
			return httperrors.NewConflictError("Isolated device already attached")
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
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return httperrors.NewResourceNotFoundError2(manager.Keyword(), devConfig.Id)
		} else {
			return errors.Wrap(err, "SIsolatedDeviceManager.FetchById")
		}
	}
	dev := devObj.(*SIsolatedDevice)
	if len(devConfig.DevType) > 0 && devConfig.DevType != dev.DevType {
		dev.DevType = devConfig.DevType
	}
	if !dev.IsEnough(devConfig.MemoryRequest) {
		return errors.Errorf("Dev %s not enough", dev.Id)
	}
	return guest.attachIsolatedDevice(ctx, userCred, dev, devConfig.NetworkIndex, devConfig.DiskIndex, &devConfig.MemoryRequest, devConfig.GpuType)
}

func (manager *SIsolatedDeviceManager) attachHostDeviceToGuestByDevicePath(ctx context.Context, guest *SGuest, host *SHost, devConfig *api.IsolatedDeviceConfig, userCred mcclient.TokenCredential, usedDevMap map[string]*SIsolatedDevice, preferNumaNodes []int) error {
	if len(devConfig.Model) == 0 || len(devConfig.DevicePath) == 0 {
		return fmt.Errorf("Model or DevicePath is empty: %#v", devConfig)
	}
	// if dev type is not nic, wire is empty string
	devs, err := manager.findHostAvailableByDevAttr(devConfig.Model, "device_path", devConfig.DevicePath, host.Id, devConfig.WireId, devConfig.SharingMode)
	if err != nil || len(devs) == 0 {
		return fmt.Errorf("Can't found model %s device_path %s on host %s", devConfig.Model, devConfig.DevicePath, host.Id)
	}
	devs = filterDevicesBySharingMode(devs, devConfig.SharingMode)
	if len(devs) == 0 {
		return fmt.Errorf("Can't found model %s device_path %s sharing_mode %s on host %s",
			devConfig.Model, devConfig.DevicePath, devConfig.SharingMode, host.Id)
	}
	devs = filterDevicesByMemoryMb(devs, devConfig.MemoryMb)
	if len(devs) == 0 {
		return fmt.Errorf("device_path %s on host %s does not satisfy memory_mb=%d",
			devConfig.DevicePath, host.Id, devConfig.MemoryMb)
	}

	var selectedDev SIsolatedDevice
	for i := range devs {
		if !devs[i].IsEnough(devConfig.MemoryRequest) {
			continue
		}

		if _, ok := usedDevMap[devs[i].DevicePath]; !ok {
			selectedDev = devs[i]
			usedDevMap[devs[i].DevicePath] = &selectedDev
		}
	}
	if selectedDev.Id == "" {
		selectedDev = devs[0]
	}
	if !selectedDev.IsEnough(devConfig.MemoryRequest) {
		return errors.Errorf("Dev %s not enough", selectedDev.Id)
	}
	return guest.attachIsolatedDevice(ctx, userCred, &selectedDev, devConfig.NetworkIndex, devConfig.DiskIndex, &devConfig.MemoryRequest, devConfig.GpuType)
}

// filterDevicesByMemoryMb drops devices whose MemorySize > 0 and is below the
// requested minMemMb. MemorySize == 0 means the host hasn't reported it yet
// and is treated as unknown (allowed through) to avoid mass-excluding rows
// pending backfill. minMemMb <= 0 short-circuits — no filtering.
func filterDevicesByMemoryMb(devs []SIsolatedDevice, minMemMb int) []SIsolatedDevice {
	if minMemMb <= 0 {
		return devs
	}
	out := devs[:0]
	for _, d := range devs {
		if d.MemorySize > 0 && d.MemorySize < minMemMb {
			continue
		}
		out = append(out, d)
	}
	return out
}

func filterDevicesBySharingMode(devs []SIsolatedDevice, sharingMode string) []SIsolatedDevice {
	if sharingMode == "" {
		return devs
	}
	out := make([]SIsolatedDevice, 0, len(devs))
	for _, d := range devs {
		if d.SharingMode == sharingMode {
			out = append(out, d)
		}
	}
	return out
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
	devs, err := manager.findHostDevsByDevConfig(devConfig.Model, devConfig.DevType, host.Id, devConfig.WireId, devConfig.SharingMode)
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
	devs, err := manager.findHostAvailableByDevConfig(devConfig.Model, devConfig.DevType, host.Id, devConfig.WireId, devConfig.SharingMode)
	if err != nil || len(devs) == 0 {
		return fmt.Errorf("Can't found model %s on host %s", devConfig.Model, host.Id)
	}
	devs = filterDevicesBySharingMode(devs, devConfig.SharingMode)
	if len(devs) == 0 {
		return fmt.Errorf("Can't found model %s sharing_mode %s on host %s",
			devConfig.Model, devConfig.SharingMode, host.Id)
	}
	// Honour the request's VRAM floor. Predicate already verified enough
	// fitting devices exist on the host; here we make sure attach picks one
	// of them rather than a same-Model but smaller-VRAM card.
	devs = filterDevicesByMemoryMb(devs, devConfig.MemoryMb)
	if len(devs) == 0 {
		return fmt.Errorf("model %s on host %s has no device with memory_mb>=%d",
			devConfig.Model, host.Id, devConfig.MemoryMb)
	}
	// 1. group devices by device_path and numa nodes
	//groupDevs := make(SorttedGroupDevs, 0)
	mapDevs := map[string][]SIsolatedDevice{}
	for i := range devs {
		if !devs[i].IsEnough(devConfig.MemoryRequest) {
			continue
		}

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

	return guest.attachIsolatedDevice(ctx, userCred, selectedDev, devConfig.NetworkIndex, devConfig.DiskIndex, &devConfig.MemoryRequest, devConfig.GpuType)
}

func (manager *SIsolatedDeviceManager) FindAvailableByModels(models []string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.GetAvailableIsolatedDeviceQuery(nil)
	q = q.In("model", models)
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) FindAvailableNicWiresByModel(modelName string) ([]string, error) {
	q := manager.Query().Equals("dev_type", api.NIC_TYPE)
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
		if devs[i].IsFull() {
			continue
		}

		wires[i] = devs[i].WireId
	}
	return wires, err
}

func (manager *SIsolatedDeviceManager) FindAvailableGpusOnHost(hostId string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.GetAvailableIsolatedDeviceQuery(nil)
	q = q.Equals("dev_type", api.GPU_TYPE).Equals("host_id", hostId)
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return nil, err
	}
	return devs, nil
}

func (manager *SIsolatedDeviceManager) findHostAvailableByDevConfig(model, devType, hostId, wireId, sharingMode string) ([]SIsolatedDevice, error) {
	return manager.findHostAvailableByDevAttr(model, "dev_type", devType, hostId, wireId, sharingMode)
}

func (manager *SIsolatedDeviceManager) findHostDevsByDevConfig(model, devType, hostId, wireId, sharingMode string) ([]SIsolatedDevice, error) {
	return manager.findHostDevsByDevAttr(model, "dev_type", devType, hostId, wireId, sharingMode)
}
func (manager *SIsolatedDeviceManager) findHostDevsByDevAttr(model, attrKey, attrVal, hostId, wireId, sharingMode string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.Query()
	q = q.Equals("model", model).Equals("host_id", hostId)
	if attrVal != "" {
		q = q.Equals(attrKey, attrVal)
	}
	if sharingMode != "" {
		q = q.Equals("sharing_mode", sharingMode)
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

func (manager *SIsolatedDeviceManager) findHostAvailableByDevAttr(model, attrKey, attrVal, hostId, wireId, sharingMode string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.GetAvailableIsolatedDeviceQuery(nil)
	q = q.Equals("model", model).Equals("host_id", hostId)
	if attrVal != "" {
		q = q.Equals(attrKey, attrVal)
	}
	if sharingMode != "" {
		q = q.Equals("sharing_mode", sharingMode)
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
	gdevs, err := guest.GetGuestIsolatedDevices()
	if err != nil {
		return err
	}
	if len(gdevs) == 0 {
		return fmt.Errorf("fail to find attached devices")
	}
	for _, gdev := range gdevs {
		dev := gdev.GetIsolatedDevice()
		if !dev.IsKvmExclusiveGPU() {
			continue
		}
		err := gdev.Detach(ctx, userCred)
		if err != nil {
			db.OpsLog.LogEvent(guest, db.ACT_GUEST_DETACH_ISOLATED_DEVICE_FAIL, dev.GetShortDesc(ctx), userCred)
			return err
		}
		db.OpsLog.LogEvent(guest, db.ACT_GUEST_DETACH_ISOLATED_DEVICE, dev.GetShortDesc(ctx), userCred)
	}
	return nil
}

func (manager *SIsolatedDeviceManager) ReleaseDevicesOfGuest(ctx context.Context, guest *SGuest, userCred mcclient.TokenCredential) error {
	gdevs, err := guest.GetGuestIsolatedDevices()
	if err != nil {
		return err
	}
	if len(gdevs) == 0 {
		return fmt.Errorf("fail to find attached devices")
	}
	for _, gdev := range gdevs {
		dev := gdev.GetIsolatedDevice()
		err := gdev.Detach(ctx, userCred)
		if err != nil {
			db.OpsLog.LogEvent(guest, db.ACT_GUEST_DETACH_ISOLATED_DEVICE_FAIL, dev.GetShortDesc(ctx), userCred)
			return err
		}
		db.OpsLog.LogEvent(guest, db.ACT_GUEST_DETACH_ISOLATED_DEVICE, dev.GetShortDesc(ctx), userCred)
	}
	return nil
}

func (manager *SIsolatedDeviceManager) queryWithoutGuest(q *sqlchemy.SQuery) *sqlchemy.SQuery {
	gq := GuestIsolatedDeviceManager.Query().SubQuery()
	q = q.LeftJoin(gq, sqlchemy.Equals(q.Field("id"), gq.Field("isolated_device_id")))
	q = q.Filter(sqlchemy.IsNull(gq.Field("isolated_device_id")))
	return q
}

func (manager *SIsolatedDeviceManager) queryWithGuest(q *sqlchemy.SQuery) *sqlchemy.SQuery {
	gq := GuestIsolatedDeviceManager.Query().SubQuery()
	q = q.Join(gq, sqlchemy.Equals(q.Field("id"), gq.Field("isolated_device_id")))
	return q
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
	Devices     int
	Gpus        int
	DevicesUsed int
	GpusUsed    int
}

type IsolatedDeviceStat struct {
	DevType string
	GuestId string
	Count   int
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
) ([]IsolatedDeviceStat, error) {
	iq := manager.totalCountQ(
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
	)
	sq := iq.SubQuery()
	guestIdevs := GuestIsolatedDeviceManager.Query().SubQuery()
	q := sq.Query(
		sq.Field("dev_type"),
		guestIdevs.Field("guest_id"),
		sqlchemy.COUNT("count", sq.Field("id")),
	)
	q = q.LeftJoin(guestIdevs, sqlchemy.Equals(sq.Field("id"), guestIdevs.Field("isolated_device_id")))
	q = q.GroupBy(sq.Field("dev_type"), guestIdevs.Field("guest_id"))
	ret := []IsolatedDeviceStat{}
	err := q.All(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
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
	ret := IsolatedDeviceCountStat{}
	stat, err := manager.totalCount(
		ctx,
		scope, ownerId, nil, hostType, resourceTypes,
		providers, brands, cloudEnv,
		rangeObjs, policyResult)
	if err != nil {
		return ret, err
	}
	for _, s := range stat {
		ret.Devices += s.Count
		if s.DevType == api.GPU_TYPE {
			ret.Gpus += s.Count
		}
		if len(s.GuestId) > 0 {
			ret.DevicesUsed += s.Count
			if s.DevType == api.GPU_TYPE {
				ret.GpusUsed += s.Count
			}
		}
	}
	return ret, nil
}

func (self *SIsolatedDevice) getDesc() *api.IsolatedDeviceJsonDesc {
	return &api.IsolatedDeviceJsonDesc{
		Id:                  self.Id,
		DevType:             self.DevType,
		Model:               self.Model,
		SharingMode:         self.SharingMode,
		Addr:                self.Addr,
		VendorDeviceId:      self.VendorDeviceId,
		Vendor:              self.getVendor(),
		OvsOffloadInterface: self.OvsOffloadInterface,
		NvmeSizeMB:          self.NvmeSizeMB,
		MemorySize:          self.MemorySize,
		MdevId:              self.MdevId,
		NumaNode:            self.NumaNode,
	}
}

func (man *SIsolatedDeviceManager) GetSpecShouldCheckStatus(query *jsonutils.JSONDict) (bool, error) {
	return true, nil
}

func (man *SIsolatedDeviceManager) BatchGetModelSpecs(statusCheck bool) (jsonutils.JSONObject, error) {
	hostQ := HostManager.Query()
	q := man.Query("vendor_device_id", "model", "dev_type", "sharing_mode", "nvme_size_mb", "memory_size")
	if statusCheck {
		q = man.GetAvailableIsolatedDeviceQuery(q)
		hostQ = hostQ.Equals("status", api.BAREMETAL_RUNNING).IsTrue("enabled").
			In("host_type", []string{api.HOST_TYPE_HYPERVISOR, api.HOST_TYPE_CONTAINER, api.HOST_TYPE_ZETTAKIT})
	}
	hostSQ := hostQ.SubQuery()
	q.Join(hostSQ, sqlchemy.Equals(q.Field("host_id"), hostSQ.Field("id")))

	q.AppendField(hostSQ.Field("host_type"))
	q.GroupBy(hostSQ.Field("host_type"), q.Field("vendor_device_id"), q.Field("model"), q.Field("dev_type"), q.Field("sharing_mode"), q.Field("nvme_size_mb"), q.Field("memory_size"))
	q.AppendField(sqlchemy.COUNT("*"))

	rows, err := q.Rows()
	if err != nil {
		return nil, errors.Wrap(err, "failed get specs")
	}
	defer rows.Close()
	res := jsonutils.NewDict()

	for rows.Next() {
		var hostType, vendorDeviceId, m, t, s string
		var nvmeSize, memorySize int
		var count int
		if err := rows.Scan(&vendorDeviceId, &m, &t, &s, &nvmeSize, &memorySize, &hostType, &count); err != nil {
			return nil, errors.Wrap(err, "get model spec scan rows")
		}
		vendor := GetVendorByVendorDeviceId(vendorDeviceId)
		specKeys := man.getSpecKeys(vendor, m, t, s)
		specKey := GetSpecIdentKey(specKeys)
		spec := man.getSpecByRows(hostType, vendorDeviceId, m, t, s, &nvmeSize, &memorySize, &count)
		res.Set(specKey, spec)
	}

	return res, nil
}

func (man *SIsolatedDeviceManager) getSpecByRows(hostType, vendorDeviceId, model, devType, sharingMode string, nvmeSize, memorySize, count *int) *jsonutils.JSONDict {
	var vdev bool
	var hypervisor string
	if utils.IsInStringArray(sharingMode, api.VIRTUAL_SHARING_MODES) {
		vdev = true
	}
	switch hostType {
	case api.HOST_TYPE_CONTAINER:
		hypervisor = api.HYPERVISOR_POD
	case api.HOST_TYPE_ZETTAKIT:
		hypervisor = api.HYPERVISOR_ZETTAKIT
	default:
		hypervisor = api.HYPERVISOR_KVM
	}

	ret := jsonutils.NewDict()
	ret.Set("virtual_dev", jsonutils.NewBool(vdev))
	ret.Set("hypervisor", jsonutils.NewString(hypervisor))
	ret.Set("dev_type", jsonutils.NewString(devType))
	ret.Set("sharing_mode", jsonutils.NewString(sharingMode))
	ret.Set("model", jsonutils.NewString(model))
	ret.Set("pci_id", jsonutils.NewString(vendorDeviceId))
	ret.Set("vendor", jsonutils.NewString(GetVendorByVendorDeviceId(vendorDeviceId)))
	if count != nil {
		ret.Set("count", jsonutils.NewInt(int64(*count)))
	}
	if nvmeSize != nil {
		ret.Set("nvme_size_mb", jsonutils.NewInt(int64(*nvmeSize)))
	}
	if memorySize != nil {
		ret.Set("memory_size_mb", jsonutils.NewInt(int64(*memorySize)))
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
		gdevs, _ := self.GetAllGuestIsolatedDevices()
		if len(gdevs) > 0 {
			return nil
		}
		if host.Status != api.BAREMETAL_RUNNING || !host.GetEnabled() ||
			(host.HostType != api.HOST_TYPE_HYPERVISOR && host.HostType != api.HOST_TYPE_CONTAINER && host.HostType != api.HOST_TYPE_ZETTAKIT) {
			return nil
		}
	}
	return IsolatedDeviceManager.getSpecByRows(host.HostType, self.VendorDeviceId, self.Model, self.DevType, self.SharingMode, &self.NvmeSizeMB, &self.MemorySize, nil)
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
	sharingMode, _ := spec.GetString("sharing_mode")
	return man.getSpecKeys(vendor, model, devType, sharingMode)
}

func (man *SIsolatedDeviceManager) getSpecKeys(vendor, model, devType, sharingMode string) []string {
	keys := []string{
		fmt.Sprintf("type:%s", devType),
		fmt.Sprintf("vendor:%s", vendor),
		fmt.Sprintf("model:%s", model),
		fmt.Sprintf("sharing_mode:%s", sharingMode),
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
	shareRows := manager.SSharableBaseResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	guestIds := make([][]string, len(rows))
	guestIdsAll := make([]string, 0)
	for i := range rows {
		rows[i] = api.IsolateDeviceDetails{
			StandaloneResourceDetails: stdRows[i],
			HostResourceInfo:          hostRows[i],
			SharableResourceBaseInfo:  shareRows[i],
		}
		dev := objs[i].(*SIsolatedDevice)
		rows[i].Vendor = dev.getVendor()
		if dev.SharingMode == api.DEVICE_SHARING_MODE_HAMI {
			rows[i].MemoryAllocated, _ = dev.getAllocatedMemorySize()
		} else {
			rows[i].AllocatedCount, _ = dev.getAllocatedCount()
		}
		guestIds[i] = dev.getAttachedGuestIds()
		if len(guestIds[i]) > 0 {
			guestIdsAll = append(guestIdsAll, guestIds[i]...)
		}
	}

	guests := make(map[string]SGuest)
	err := db.FetchStandaloneObjectsByIds(GuestManager, guestIdsAll, &guests)
	if err != nil {
		log.Errorf("db.FetchStandaloneObjectsByIds fail %s", err)
		return rows
	}

	for i := range rows {
		nguests := guestIds[i]
		if len(nguests) > 0 {
			rows[i].Guest = make([]string, len(nguests))
			rows[i].GuestIds = make([]string, len(nguests))
			rows[i].GuestStatus = make([]string, len(nguests))
		}

		for j := range nguests {
			if guest, ok := guests[nguests[j]]; ok {
				rows[i].Guest[j] = guest.Name
				rows[i].GuestIds[j] = guest.Id
				rows[i].GuestStatus[j] = guest.Status
			}
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
	guestIsolatedDevices := self.getAttachedGuests()
	if len(guestIsolatedDevices) > 0 {
		if !jsonutils.QueryBoolean(data, "purge", false) {
			return httperrors.NewBadRequestError("%s", api.ErrMsgIsolatedDeviceUsedByServer)
		}

		for i := range guestIsolatedDevices {
			err := guestIsolatedDevices[i].Detach(ctx, userCred)
			if err != nil {
				return err
			}
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

func (manager *SIsolatedDeviceManager) GetAvailableIsolatedDeviceOnHost(hostId string, model, sharingMode string) ([]SIsolatedDevice, error) {
	devs := make([]SIsolatedDevice, 0)
	q := manager.GetAvailableIsolatedDeviceQuery(manager.Query().Equals("host_id", hostId).Equals("model", model)).Equals("sharing_mode", sharingMode)
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
	cnt, err := manager.queryWithGuest(manager.Query().Equals("model", model).
		Equals("dev_type", devType).
		Equals("vendor_device_id", fmt.Sprintf("%s:%s", vendor, device))).
		CountWithError()
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
	return rbacscope.ScopeProject
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
	if err != nil {
		return err
	}
	sharedProjectIds, err := dev.GetSharedProjectIds()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented {
			return nil
		}
		return err
	}
	log.Infof("share projectIds: %s", sharedProjectIds)
	if len(sharedProjectIds) == 0 {
		return nil
	}
	host := model.getHost()
	if host == nil {
		return nil
	}
	if len(sharedProjectIds) > 0 {
		projectIds, err := db.FetchField(ExternalProjectManager, "tenant_id", func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			return q.Equals("manager_id", host.ManagerId).In("external_id", sharedProjectIds)
		})
		if err != nil {
			return err
		}
		input := apis.PerformPublicProjectInput{SharedProjectIds: projectIds}
		input.Scope = "project"
		err = db.SharablePerformPublic(model, ctx, userCred, input)
		if err != nil {
			return errors.Wrapf(err, "SharablePerformPublic")
		}
	}
	return nil
}

func (model *SIsolatedDevice) GetRequiredSharedDomainIds() []string {
	host := model.getHost()
	if host != nil {
		return []string{host.DomainId}
	}
	return []string{}
}

func (model *SIsolatedDevice) GetSharableTargetDomainIds() []string {
	return nil
}

func (model *SIsolatedDevice) GetSharedDomains() []string {
	return db.SharableGetSharedProjects(model, db.SharedTargetDomain)
}

func (model *SIsolatedDevice) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicProjectInput) (jsonutils.JSONObject, error) {
	err := db.SharablePerformPublic(model, ctx, userCred, input)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPublic")
	}
	return nil, nil
}

func (model *SIsolatedDevice) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	err := db.SharablePerformPrivate(model, ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "SharablePerformPrivate")
	}
	return nil, nil
}

type HostIsolatedDevicesNumaStat struct {
	NumaNode         int8
	NumaNodeDevCount int
}

func (manager *SIsolatedDeviceManager) GetHostAllocatedIsolatedDeviceNumaStats(devModel, hostId string) ([]HostIsolatedDevicesNumaStat, error) {
	q := GuestIsolatedDeviceManager.Query()
	guestQ := GuestManager.Query().NotEquals("status", api.VM_READY).SubQuery()
	isq := manager.Query().Equals("host_id", hostId).Equals("model", devModel).SubQuery()

	q = q.Join(isq, sqlchemy.Equals(q.Field("isolated_device_id"), isq.Field("id")))
	q = q.Join(guestQ, sqlchemy.Equals(q.Field("guest_id"), guestQ.Field("id")))
	q = q.GroupBy(isq.Field("numa_node"))
	q = q.AppendField(sqlchemy.COUNT("numa_node_dev_count", isq.Field("numa_node")))

	subQ := q.SubQuery()
	numaQ := subQ.Query(subQ.Field("numa_node"), subQ.Field("numa_node_dev_count"))
	stats := make([]HostIsolatedDevicesNumaStat, 0)
	err := numaQ.All(&stats)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (host *SHost) VirtualDeviceNumaBalance(devModel string, numaNode int8) (bool, error) {
	if numaNode < 0 {
		return true, nil
	}
	stats, err := IsolatedDeviceManager.GetHostAllocatedIsolatedDeviceNumaStats(devModel, host.Id)
	if err != nil {
		return true, err
	}
	log.Debugf("VirtualDeviceNumaBalance numa stats %s", jsonutils.Marshal(stats))

	if len(stats) <= 1 {
		return true, nil
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].NumaNodeDevCount > stats[j].NumaNodeDevCount
	})

	if stats[0].NumaNodeDevCount-stats[len(stats)-1].NumaNodeDevCount > 1 {
		return false, nil
	}
	return true, nil
}

func (manager *SIsolatedDeviceManager) GetAvailableIsolatedDeviceQuery(isq *sqlchemy.SQuery) *sqlchemy.SQuery {
	guestIdevs := GuestIsolatedDeviceManager.Query().SubQuery()
	guestIsQ := guestIdevs.Query(
		guestIdevs.Field("isolated_device_id"),
		sqlchemy.SUM("memory_allocated", guestIdevs.Field("device_memory_size")),
		sqlchemy.COUNT("guest_count", guestIdevs.Field("guest_id")),
	).GroupBy("isolated_device_id").SubQuery()

	if isq == nil {
		isq = manager.Query()
	}

	isq = isq.LeftJoin(guestIsQ, sqlchemy.Equals(isq.Field("id"), guestIsQ.Field("isolated_device_id")))
	cond1 := sqlchemy.AND(
		sqlchemy.Equals(isq.Field("sharing_mode"), api.DEVICE_SHARING_MODE_HAMI),
		sqlchemy.OR(
			sqlchemy.IsNull(guestIsQ.Field("memory_allocated")),
			sqlchemy.GT(isq.Field("memory_size"), guestIsQ.Field("memory_allocated")),
		),
	)
	cond2 := sqlchemy.AND(
		sqlchemy.NotEquals(isq.Field("sharing_mode"), api.DEVICE_SHARING_MODE_HAMI),
		sqlchemy.OR(
			sqlchemy.IsNull(guestIsQ.Field("guest_count")),
			sqlchemy.GT(isq.Field("virtual_num"), guestIsQ.Field("guest_count")),
		),
	)

	isq = isq.Filter(sqlchemy.OR(cond1, cond2))

	return isq
}

type IsolatedDeviceAllocateStat struct {
	SIsolatedDevice

	GuestCount      int
	MemoryAllocated int
}

func (manager *SIsolatedDeviceManager) GetHostsIsolatedDeviceStats(hostIds []string) []IsolatedDeviceAllocateStat {
	guestIdevs := GuestIsolatedDeviceManager.Query().SubQuery()
	guestIsQ := guestIdevs.Query(
		guestIdevs.Field("isolated_device_id"),
		sqlchemy.SUM("memory_allocated", guestIdevs.Field("device_memory_size")),
		sqlchemy.COUNT("guest_count", guestIdevs.Field("guest_id")),
	).GroupBy("isolated_device_id").SubQuery()

	isq := manager.Query().In("host_id", hostIds)
	isq = isq.LeftJoin(guestIsQ, sqlchemy.Equals(isq.Field("id"), guestIsQ.Field("isolated_device_id")))
	isq.AppendField(isq.QueryFields()...)
	isq.AppendField(guestIsQ.Field("memory_allocated"), guestIsQ.Field("guest_count"))
	stats := make([]IsolatedDeviceAllocateStat, 0)
	err := isq.All(&stats)
	if err != nil {
		log.Errorf("GetHostsIsolatedDevicesDetails %s", err)
	}

	return stats
}

func (manager *SIsolatedDeviceManager) GetHostsGuestIsolatedDevices(hostIds []string) map[string][]string {
	gidq := GuestIsolatedDeviceManager.Query().SubQuery()
	isq := manager.Query().SubQuery()

	q := gidq.Query()
	q = q.Join(isq, sqlchemy.Equals(isq.Field("id"), gidq.Field("isolated_device_id")))
	q = q.Filter(sqlchemy.In(isq.Field("host_id"), hostIds))

	result := []struct {
		IsolatedDeviceId string
		GuestId          string
	}{}
	err := q.All(&result)
	if err != nil {
		log.Errorf("GetHostsGuestIsolatedDevices query %s", err)
		return nil
	}
	ret := map[string][]string{}
	for i := range result {
		if guests, ok := ret[result[i].IsolatedDeviceId]; ok {
			ret[result[i].IsolatedDeviceId] = append(guests, result[i].GuestId)
		} else {
			ret[result[i].IsolatedDeviceId] = []string{result[i].GuestId}
		}
	}
	return ret
}

func (dev *SIsolatedDevice) IsEnough(memoryRequest int) bool {
	if dev.SharingMode != api.DEVICE_SHARING_MODE_HAMI {
		cnt, err := dev.getAllocatedCount()
		if err != nil {
			log.Errorf("failed getAllocatedCount %s", err)
			return false
		}
		return dev.VirtualNum > cnt
	} else {
		allocated, err := dev.getAllocatedMemorySize()
		if err != nil {
			log.Errorf("failed getAllocatedMemorySize %s", err)
			return false
		}
		return (dev.MemorySize - allocated) >= memoryRequest
	}
}

func (dev *SIsolatedDevice) IsFull() bool {
	if dev.SharingMode != api.DEVICE_SHARING_MODE_HAMI {
		cnt, err := dev.getAllocatedCount()
		if err != nil {
			log.Errorf("failed getAllocatedCount %s", err)
			return true
		}
		return dev.VirtualNum <= cnt
	} else {
		allocated, err := dev.getAllocatedMemorySize()
		if err != nil {
			log.Errorf("failed getAllocatedMemorySize %s", err)
			return true
		}
		return dev.MemorySize <= allocated
	}
}

func (manager *SIsolatedDeviceManager) InitializeData() error {
	ctx := context.Background()
	inited, err := manager.isInitializeDataDone()
	if err != nil {
		return errors.Wrap(err, "isInitializeDataDone")
	}
	if inited {
		return nil
	}

	if err := manager.migrateGuestIsolatedDevices(); err != nil {
		return errors.Wrap(err, "migrateGuestIsolatedDevices")
	}
	if err := manager.mergeVirtualIsolatedDevices(); err != nil {
		return errors.Wrap(err, "mergeVirtualIsolatedDevices")
	}
	if err := manager.initVirtualNum(); err != nil {
		return errors.Wrap(err, "initVirtualNum")
	}
	if err := manager.migrateDevType(); err != nil {
		return errors.Wrap(err, "migrateDevType")
	}
	if err := manager.markInitializeDataDone(ctx); err != nil {
		return errors.Wrap(err, "markInitializeDataDone")
	}
	return nil
}

func (manager *SIsolatedDeviceManager) getInitializeDataMetadataId() string {
	return fmt.Sprintf("%s%s%s", isolatedDeviceInitializeDataObjType, db.OBJECT_TYPE_ID_SEP, isolatedDeviceInitializeDataObjId)
}

func (manager *SIsolatedDeviceManager) isInitializeDataDone() (bool, error) {
	md := db.SMetadata{}
	err := db.Metadata.RawQuery("value", "deleted").
		Equals("id", manager.getInitializeDataMetadataId()).
		Equals("key", isolatedDeviceInitializeDataKey).
		First(&md)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return !md.Deleted && md.Value == "true", nil
}

func (manager *SIsolatedDeviceManager) markInitializeDataDone(ctx context.Context) error {
	md := db.SMetadata{}
	md.SetModelManager(db.Metadata, &md)
	err := db.Metadata.RawQuery().
		Equals("id", manager.getInitializeDataMetadataId()).
		Equals("key", isolatedDeviceInitializeDataKey).
		First(&md)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return err
		}
		md.ObjType = isolatedDeviceInitializeDataObjType
		md.ObjId = isolatedDeviceInitializeDataObjId
		md.Id = manager.getInitializeDataMetadataId()
		md.Key = isolatedDeviceInitializeDataKey
		md.Value = "true"
		return db.Metadata.TableSpec().Insert(ctx, &md)
	}
	_, err = db.Update(&md, func() error {
		md.ObjType = isolatedDeviceInitializeDataObjType
		md.ObjId = isolatedDeviceInitializeDataObjId
		md.Value = "true"
		md.Deleted = false
		return nil
	})
	return err
}

func getIsolatedDeviceBaseAddr(addr string) string {
	return strings.Split(addr, "-")[0]
}

func isMergeableVirtualDevType(devType string) bool {
	return utils.IsInStringArray(devType, api.VITRUAL_DEVICE_TYPES)
}

type isolatedDeviceMergeKey struct {
	HostId         string
	Addr           string
	MdevId         string
	VendorDeviceId string
	DevType        string
}

func (manager *SIsolatedDeviceManager) migrateGuestIsolatedDevices() error {
	rows, err := sqlchemy.GetDB().Query(fmt.Sprintf("SELECT id, guest_id, dev_type, network_index, disk_index FROM %s WHERE deleted = 0 AND guest_id IS NOT NULL AND LENGTH(guest_id) > 0", manager.TableSpec().Name()))
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return errors.Wrap(err, "migrateGuestIsolatedDevices QueryRows")
	}
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		return nil
	}
	defer rows.Close()

	ctx := context.Background()
	migrated := 0
	guestIndexMap := map[string]int8{}
	for rows.Next() {
		var devId, guestId, devType string
		var networkIndex int
		var diskIndex int8
		if err = rows.Scan(&devId, &guestId, &devType, &networkIndex, &diskIndex); err != nil {
			return errors.Wrap(err, "migrateGuestIsolatedDevices Scan")
		}
		cnt, err := GuestIsolatedDeviceManager.Query().
			Equals("guest_id", guestId).
			Equals("isolated_device_id", devId).
			CountWithError()
		if err != nil {
			return errors.Wrapf(err, "count guest isolated device for device %s", devId)
		}
		if cnt == 0 {
			idx, ok := guestIndexMap[guestId]
			if !ok {
				maxIdx, err := manager.getGuestIsolatedDeviceMaxIndex(guestId)
				if err != nil {
					return errors.Wrapf(err, "getGuestIsolatedDeviceMaxIndex guest %s", guestId)
				}
				idx = maxIdx + 1
			}
			guestIndexMap[guestId] = idx + 1

			guestIsolatedDevice := SGuestIsolatedDevice{}
			guestIsolatedDevice.SetModelManager(GuestIsolatedDeviceManager, &guestIsolatedDevice)
			guestIsolatedDevice.GuestId = guestId
			guestIsolatedDevice.IsolatedDeviceId = devId
			guestIsolatedDevice.Index = idx
			if devType == api.GPU_VGA_TYPE {
				guestIsolatedDevice.GpuType = api.GPU_VGA
			} else if utils.IsInStringArray(devType, api.GPU_TYPES) {
				guestIsolatedDevice.GpuType = api.GPU_HPC
			}
			if networkIndex >= 0 {
				guestIsolatedDevice.NetworkIndex = networkIndex
			}
			if diskIndex >= 0 {
				guestIsolatedDevice.DiskIndex = diskIndex
			}
			if err := GuestIsolatedDeviceManager.TableSpec().Insert(ctx, &guestIsolatedDevice); err != nil {
				return errors.Wrapf(err, "insert guest isolated device for device %s guest %s", devId, guestId)
			}
			migrated++
		}
	}

	log.Infof("migrated %d legacy isolated device guest assign to guest_isolated_devices_tbl", migrated)
	return nil
}

func (manager *SIsolatedDeviceManager) getGuestIsolatedDeviceMaxIndex(guestId string) (int8, error) {
	type maxIdxResult struct {
		MaxIndex int8
	}
	sq := GuestIsolatedDeviceManager.Query().Equals("guest_id", guestId).SubQuery()
	q := sq.Query(sqlchemy.MAX("max_index", sq.Field("index")))
	ret := maxIdxResult{MaxIndex: -1}
	err := q.First(&ret)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return -1, err
	}
	return ret.MaxIndex, nil
}

func (manager *SIsolatedDeviceManager) mergeVirtualIsolatedDevices() error {
	devs := make([]SIsolatedDevice, 0)
	q := manager.Query().In("dev_type", api.VITRUAL_DEVICE_TYPES).NotEquals("dev_type", api.CONTAINER_DEV_NVIDIA_HAMI)
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return errors.Wrap(err, "FetchModelObjects")
	}
	if len(devs) == 0 {
		return nil
	}

	grouped := map[isolatedDeviceMergeKey][]*SIsolatedDevice{}
	for i := range devs {
		dev := &devs[i]
		key := isolatedDeviceMergeKey{
			HostId:         dev.HostId,
			Addr:           getIsolatedDeviceBaseAddr(dev.Addr),
			MdevId:         dev.MdevId,
			VendorDeviceId: dev.VendorDeviceId,
			DevType:        dev.DevType,
		}
		grouped[key] = append(grouped[key], dev)
	}

	mergedGroups := 0
	deletedDevs := 0
	for _, group := range grouped {
		if len(group) <= 1 {
			continue
		}
		sort.Slice(group, func(i, j int) bool {
			return group[i].CreatedAt.Before(group[j].CreatedAt)
		})
		keeper := group[0]
		ids := make([]string, len(group)-1)
		for i := 1; i < len(group); i++ {
			ids[i-1] = group[i].Id
		}
		err := manager.doMergeGuestIsolatedDevices(keeper.Id, getIsolatedDeviceBaseAddr(keeper.Addr), ids)
		if err != nil {
			return err
		}
		host := HostManager.FetchHostById(group[0].HostId)
		if host != nil && host.HostType == api.HOST_TYPE_CONTAINER {
			err = manager.doReplaceContainerIsolatedDeviceId(keeper.Id, ids)
			if err != nil {
				return err
			}
		}
		mergedGroups++
	}
	log.Infof("merged %d duplicate virtual isolated device groups, deleted %d duplicate devices", mergedGroups, deletedDevs)
	return nil
}

func (manager *SIsolatedDeviceManager) doReplaceContainerIsolatedDeviceId(keeperId string, originIds []string) error {
	log.Infof("start replace contaienr isolated device id, originIds %v, keeper %s", originIds, keeperId)
	ids := []string{keeperId}
	ids = append(ids, originIds...)
	gdevs := make([]SGuestIsolatedDevice, 0)
	q := GuestIsolatedDeviceManager.Query().In("isolated_device_id", ids)
	err := db.FetchModelObjects(GuestIsolatedDeviceManager, q, &gdevs)
	if err != nil {
		return errors.Wrap(err, "GuestIsolatedDeviceManager.FetchModelObjects")
	}
	for i := range gdevs {
		ctrs, err := GetContainerManager().GetContainersByPod(gdevs[i].GuestId)
		if err != nil {
			return errors.Wrapf(err, "GetContainerManager().GetContainersByPod")
		}
		for j := range ctrs {
			ctrPtr := &ctrs[j]

			spec := new(api.ContainerSpec)
			if err := jsonutils.Marshal(ctrPtr.Spec).Unmarshal(spec); err != nil {
				return errors.Wrap(err, "deep copy spec")
			}

			updated := false
			for k := range spec.Devices {
				if spec.Devices[k].IsolatedDevice == nil {
					continue
				}
				if !utils.IsInStringArray(spec.Devices[k].IsolatedDevice.Id, ids) {
					continue
				}
				spec.Devices[k].IsolatedDevice.Id = keeperId
				spec.Devices[k].IsolatedDevice.GuestIsolatedDeviceIndex = int(gdevs[i].Index)
				updated = true
			}
			if !updated {
				continue
			}
			_, err = db.Update(ctrPtr, func() error {
				ctrPtr.Spec = spec
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "update ctr isolated device id")
			}
			log.Infof("replace container isolated device id %s to %s", ctrPtr.Id, keeperId)
		}
	}
	return nil
}

func (manager *SIsolatedDeviceManager) doMergeGuestIsolatedDevices(keeperId, keeperNewAddr string, originIds []string) error {
	tx, err := sqlchemy.GetDB().Begin()
	if err != nil {
		return errors.Wrap(err, "failed begin TRANSACTION")
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	buildInPlaceholders := func(n int) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = "?"
		}
		return strings.Join(parts, ",")
	}
	var res sql.Result

	if len(originIds) > 0 {
		sql := fmt.Sprintf(
			"update %s set isolated_device_id = ? where isolated_device_id in (%s)",
			GuestIsolatedDeviceManager.TableSpec().Name(), buildInPlaceholders(len(originIds)),
		)
		args := make([]interface{}, 1, 1+len(originIds))
		args[0] = keeperId
		for i := range originIds {
			args = append(args, originIds[i])
		}
		res, err = tx.Exec(sql, args...)
		if err != nil {
			return errors.Wrapf(err, "failed exec TRANSACTION: %s", sql)
		}
		affected, _ := res.RowsAffected()
		log.Infof("sql %s effect %d", sql, affected)

		sql = fmt.Sprintf(
			"update %s set deleted = 1 where id in (%s)",
			IsolatedDeviceManager.TableSpec().Name(), buildInPlaceholders(len(originIds)),
		)
		args = make([]interface{}, len(originIds))
		for i := range originIds {
			args[i] = originIds[i]
		}
		res, err = tx.Exec(sql, args...)
		if err != nil {
			return errors.Wrapf(err, "failed exec TRANSACTION: %s", sql)
		}
		affected, _ = res.RowsAffected()
		log.Infof("sql %s effect %d", sql, affected)
		if affected != int64(len(originIds)) {
			return errors.Errorf("TRANSACTION: %s affected rows %d not equal to originIds length", sql, affected)
		}
	}

	virtualNum := 1 + len(originIds)
	sql := fmt.Sprintf(
		"update %s set virtual_num = ?, addr = ? where id = ?",
		IsolatedDeviceManager.TableSpec().Name(),
	)
	res, err = tx.Exec(sql, virtualNum, keeperNewAddr, keeperId)
	if err != nil {
		return errors.Wrapf(err, "failed exec TRANSACTION: %s", sql)
	}
	affected, _ := res.RowsAffected()
	log.Infof("sql %s effect %d", sql, affected)

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed commit TRANSACTION")
	}
	return nil
}

func (manager *SIsolatedDeviceManager) initVirtualNum() error {
	devs := make([]SIsolatedDevice, 0)
	q := manager.Query().NotEquals("dev_type", api.CONTAINER_DEV_NVIDIA_HAMI)
	q = q.Filter(sqlchemy.OR(
		sqlchemy.IsNull(q.Field("virtual_num")),
		sqlchemy.LE(q.Field("virtual_num"), 0),
	))
	err := db.FetchModelObjects(manager, q, &devs)
	if err != nil {
		return errors.Wrap(err, "FetchModelObjects")
	}
	updated := 0
	for i := range devs {
		dev := &devs[i]
		virtualNum := 1
		if isMergeableVirtualDevType(dev.DevType) {
			cnt, err := dev.getAllocatedCount()
			if err != nil {
				return errors.Wrapf(err, "getAllocatedCount device %s", dev.Id)
			}
			if cnt > virtualNum {
				virtualNum = cnt
			}
		}
		if _, err := db.Update(dev, func() error {
			dev.VirtualNum = virtualNum
			return nil
		}); err != nil {
			return errors.Wrapf(err, "set virtual_num on device %s", dev.Id)
		}
		updated++
	}
	log.Infof("initialized virtual_num for %d isolated devices", updated)
	return nil
}

func (manager *SIsolatedDeviceManager) migrateDevType() error {
	hotPluggableDevTypes := []string{
		api.DIRECT_PCI_TYPE, api.USB_TYPE, api.GPU_VGA_TYPE, api.GPU_HPC_TYPE,
		api.SRIOV_VGPU_TYPE, api.LEGACY_VGPU_TYPE,
	}

	for _, devType := range api.VALID_PASSTHROUGH_TYPES {
		var sharingMode string
		switch devType {
		case api.DIRECT_PCI_TYPE, api.USB_TYPE, api.GPU_VGA_TYPE, api.GPU_HPC_TYPE:
			sharingMode = api.DEVICE_SHARING_MODE_EXCLUSIVE
		case api.SRIOV_VGPU_TYPE:
			sharingMode = api.DEVICE_SHARING_MODE_SRIOV
		case api.CONTAINER_DEV_NVIDIA_MPS:
			sharingMode = api.DEVICE_SHARING_MODE_MPS
		case api.CONTAINER_DEV_NVIDIA_HAMI:
			sharingMode = api.DEVICE_SHARING_MODE_HAMI
		case api.LEGACY_VGPU_TYPE:
			sharingMode = api.DEVICE_SHARING_MODE_MDEV
		default:
			sharingMode = api.DEVICE_SHARING_MODE_UNLIMITED
		}
		var hotPluggable = 0
		if utils.IsInStringArray(devType, hotPluggableDevTypes) {
			hotPluggable = 1
		}
		var targetDevType string
		switch {
		case utils.IsInStringArray(devType, api.GPU_TYPES):
			targetDevType = api.GPU_TYPE
		case utils.IsInStringArray(devType, api.NETINT_TYPES):
			targetDevType = api.NETINT_TYPE
		case devType == api.CONTAINER_DEV_CPH_AOSP_BINDER:
			targetDevType = api.BINDER_TYPE
		case devType == api.CONTAINER_DEV_ASCEND_NPU:
			targetDevType = api.NPU_TYPE
		default:
			targetDevType = devType
		}

		sql := fmt.Sprintf(
			"update %s set dev_type = ?, sharing_mode = ?, hot_pluggable = ? where dev_type = ? and deleted = 0",
			manager.TableSpec().Name(),
		)
		res, err := sqlchemy.GetDB().Exec(sql, targetDevType, sharingMode, hotPluggable, devType)
		if err != nil {
			return errors.Wrapf(err, "update dev_type from %v to %s", devType, targetDevType)
		}
		effects, _ := res.RowsAffected()
		log.Infof("updated dev_type from %v to %s, effects: %d", devType, targetDevType, effects)
	}
	return nil
}

func (dev *SIsolatedDevice) IsValidAttachDev() bool {
	if dev.DevType == api.NIC_TYPE || dev.DevType == api.NVME_PT_TYPE {
		return false
	}
	return true
}

func (dev *SIsolatedDevice) IsKvmExclusiveGPU() bool {
	if dev.DevType != api.GPU_TYPE {
		return false
	}
	if dev.SharingMode != api.DEVICE_SHARING_MODE_EXCLUSIVE {
		return false
	}
	host := dev.GetHost()
	if host.HostType != api.HOST_TYPE_KVM {
		return false
	}
	return true
}
