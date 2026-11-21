package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func NewSLLMSkuBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SLLMSkuBaseManager {
	return SLLMSkuBaseManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
		),
	}
}

type SLLMSkuBaseManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SLLMSkuBase struct {
	db.SSharableVirtualResourceBase

	Bandwidth          int               `nullable:"false" default:"0" create:"optional" list:"user" update:"user"`
	Cpu                int               `nullable:"false" default:"1" create:"optional" list:"user" update:"user"`
	Memory             int               `nullable:"false" default:"512" create:"optional" list:"user" update:"user"`
	EnableCgroupCpu    *bool             `default:"true" create:"optional" list:"user" update:"user"`
	EnableCgroupMemory *bool             `default:"true" create:"optional" list:"user" update:"user"`
	Volumes            *api.Volumes      `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	HostPaths          *api.HostPaths    `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	PortMappings       *api.PortMappings `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	Devices            *api.Devices      `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	Envs               *api.Envs         `charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
	// Properties
	Properties map[string]string `charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
}

func (man *SLLMSkuBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.SharableVirtualResourceListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, errors.Wrapf(err, "SSharableBaseResourceManager.ListItemFilter")
	}
	return q, nil
}

func (man *SLLMSkuBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.LLMSKuBaseCreateInput) (api.LLMSKuBaseCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
	}
	if input.Cpu <= 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "cpu must > 0")
	}
	if input.Memory <= 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "mem must > 0")
	}
	// A data disk is optional at the base layer; the concrete llm type decides
	// whether it is required (see SLLMSkuManager.ValidateCreateData).

	if err := normalizeLLMSkuDevices(input.Devices); err != nil {
		return input, err
	}

	input.Status = api.STATUS_READY
	return input, nil
}

// normalizeLLMSkuDevices maps NVIDIA_* / HYGON_* / ASCEND_* / ILUVATAR_* / THEAD_* / KUNLUNXIN_* / MTHREADS_* DevTypes onto
// GPU|NPU + SharingMode, and defaults empty DevType/SharingMode appropriately.
func normalizeLLMSkuDevices(devices *api.Devices) error {
	if devices == nil || len(*devices) == 0 {
		return nil
	}
	mthreadsHami := 0
	for i := range *devices {
		normalizeLLMSkuDevice(&(*devices)[i])
		dev := (*devices)[i]
		if computeapi.IsMthreadsHami(dev.Vendor, dev.SharingMode) {
			mthreadsHami++
		}
	}
	if mthreadsHami > 1 {
		return httperrors.NewInputParameterError("Moore Threads sGPU does not support multiple cards in one task; use exclusive mode for multiple cards")
	}
	return nil
}

func canonicalizeLLMDeviceVendor(vendor string) string {
	vendor = strings.TrimSpace(vendor)
	if vendor == "" {
		return ""
	}
	for name := range computeapi.VENDOR_ID_MAP {
		if strings.EqualFold(name, vendor) {
			return name
		}
	}
	if name, ok := computeapi.ID_VENDOR_MAP[vendor]; ok {
		return name
	}
	return strings.ToUpper(vendor)
}

func normalizeLLMSkuDevice(dev *api.Device) {
	origDevType := dev.DevType
	switch dev.DevType {
	case "":
		// Decided after vendor canonicalize when Vendor is ASCEND.
		dev.DevType = computeapi.GPU_TYPE
	case computeapi.CONTAINER_DEV_NVIDIA_GPU:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
		}
	case computeapi.CONTAINER_DEV_NVIDIA_MPS:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_MPS
		}
	case computeapi.CONTAINER_DEV_NVIDIA_GPU_SHARE:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_UNLIMITED
		}
	case computeapi.CONTAINER_DEV_NVIDIA_HAMI:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_HAMI
		}
	case computeapi.CONTAINER_DEV_HYGON_DCU:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
		}
	case computeapi.CONTAINER_DEV_HYGON_DCU_HAMI:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_HAMI
		}
	case computeapi.CONTAINER_DEV_ASCEND_NPU:
		dev.DevType = computeapi.NPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
		}
	case computeapi.CONTAINER_DEV_ASCEND_NPU_HAMI:
		dev.DevType = computeapi.NPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_HAMI
		}
	case computeapi.CONTAINER_DEV_ILUVATAR_GPU:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
		}
	case computeapi.CONTAINER_DEV_THEAD_PPU:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
		}
	case computeapi.CONTAINER_DEV_KUNLUNXIN_XPU:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
		}
	case computeapi.CONTAINER_DEV_MTHREADS_GPU:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_EXCLUSIVE
		}
	case computeapi.CONTAINER_DEV_MTHREADS_GPU_HAMI:
		dev.DevType = computeapi.GPU_TYPE
		if dev.SharingMode == "" {
			dev.SharingMode = computeapi.DEVICE_SHARING_MODE_HAMI
		}
	}
	if dev.SharingMode == "" {
		dev.SharingMode = computeapi.DEVICE_SHARING_MODE_HAMI
	}
	if dev.Vendor == "" {
		switch origDevType {
		case computeapi.CONTAINER_DEV_HYGON_DCU, computeapi.CONTAINER_DEV_HYGON_DCU_HAMI:
			dev.Vendor = "HYGON"
		case computeapi.CONTAINER_DEV_NVIDIA_GPU, computeapi.CONTAINER_DEV_NVIDIA_MPS,
			computeapi.CONTAINER_DEV_NVIDIA_GPU_SHARE, computeapi.CONTAINER_DEV_NVIDIA_HAMI:
			dev.Vendor = "NVIDIA"
		case computeapi.CONTAINER_DEV_ASCEND_NPU, computeapi.CONTAINER_DEV_ASCEND_NPU_HAMI:
			dev.Vendor = "ASCEND"
		case computeapi.CONTAINER_DEV_ILUVATAR_GPU:
			dev.Vendor = "ILUVATAR"
		case computeapi.CONTAINER_DEV_THEAD_PPU:
			dev.Vendor = "THEAD"
		case computeapi.CONTAINER_DEV_KUNLUNXIN_XPU:
			dev.Vendor = "KUNLUNXIN"
		case computeapi.CONTAINER_DEV_MTHREADS_GPU, computeapi.CONTAINER_DEV_MTHREADS_GPU_HAMI:
			dev.Vendor = "MTHREADS"
		}
	}
	dev.Vendor = canonicalizeLLMDeviceVendor(dev.Vendor)
	// Ascend devices are NPUs; correct empty/legacy GPU defaults when vendor is ASCEND.
	if dev.Vendor == "ASCEND" && (origDevType == "" || origDevType == computeapi.GPU_TYPE) {
		dev.DevType = computeapi.NPU_TYPE
	}
}

func (skuBase *SLLMSkuBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMSkuBaseUpdateInput) (api.LLMSkuBaseUpdateInput, error) {
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = skuBase.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate SharableVirtualResourceBaseUpdateInput")
	}

	// Data disks are optional for most llm types, so the client may explicitly
	// clear them by sending an empty volumes array. When volumes are absent the
	// stored ones are rebuilt with the legacy disk_size / storage_type
	// overrides. Provided entries are merged onto the stored volume at the same
	// index so callers that only send {"size_mb": ...} keep storage_type,
	// template_id and the container mount relations.
	volumes := []api.Volume{}
	if input.Volumes != nil {
		dbVolumes := []api.Volume{}
		if err := jsonutils.Marshal(skuBase.Volumes).Unmarshal(&dbVolumes); err != nil {
			return input, errors.Wrapf(err, "Unmarshal Volumes")
		}
		for i, volume := range *input.Volumes {
			if i < len(dbVolumes) {
				if volume.StorageType == "" {
					volume.StorageType = dbVolumes[i].StorageType
				}
				if volume.TemplateId == "" {
					volume.TemplateId = dbVolumes[i].TemplateId
				}
				if len(volume.Containers) == 0 {
					volume.Containers = dbVolumes[i].Containers
				}
			}
			volumes = append(volumes, volume)
		}
	} else if err := jsonutils.Marshal(skuBase.Volumes).Unmarshal(&volumes); err != nil {
		return input, errors.Wrapf(err, "Unmarshal Volumes")
	}
	if len(volumes) == 0 && input.DiskSizeMB != nil && *input.DiskSizeMB > 0 {
		// Adding a data disk to a previously diskless SKU.
		volumes = append(volumes, api.Volume{SizeMB: *input.DiskSizeMB})
	}
	for i, volume := range volumes {
		if input.DiskSizeMB != nil && *input.DiskSizeMB > 0 {
			volume.SizeMB = *input.DiskSizeMB
		}
		// if input.TemplateId != nil {
		// 	if len(*input.TemplateId) > 0 {
		// 		s := auth.GetSession(ctx, userCred, "")
		// 		imgObj, err := imagemodules.Images.Get(s, *input.TemplateId, nil)
		// 		if err != nil {
		// 			return input, errors.Wrapf(err, "validate template_id %s", *input.TemplateId)
		// 		}
		// 		volume.TemplateId, _ = imgObj.GetString("id")
		// 	} else {
		// 		volume.TemplateId = ""
		// 	}
		// }
		if input.StorageType != nil && len(*input.StorageType) > 0 {
			volume.StorageType = *input.StorageType
		}
		volumes[i] = volume
	}
	input.Volumes = (*api.Volumes)(&volumes)

	if input.Devices != nil {
		if err := normalizeLLMSkuDevices(input.Devices); err != nil {
			return input, err
		}
	}

	return input, nil
}

// SkuHasDataDisk reports whether the SKU requests a data disk. Nil-safe.
func SkuHasDataDisk(sku *SLLMSku) bool {
	return sku != nil && sku.Volumes != nil && !sku.Volumes.IsZero()
}
