package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	schedulerapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/utils/vram"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	schedulermodules "yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
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

	Bandwidth int `nullable:"false" default:"0" create:"optional" list:"user" update:"user"`
	Cpu       int `nullable:"false" default:"1" create:"optional" list:"user" update:"user"`
	Memory    int `nullable:"false" default:"512" create:"optional" list:"user" update:"user"`
	// VramClaimMb is the heuristic VRAM (MiB) needed to start a single SLLM
	// instance from this SKU. Auto-filled from the largest mounted InstantModel's
	// WeightSizeBytes via EstimateVramClaimMb; user can override at create/update
	// time (any explicit non-zero value bypasses the auto-fill). 0 means unknown.
	VramClaimMb  int               `nullable:"false" default:"0" create:"optional" list:"user" update:"user"`
	Volumes      *api.Volumes      `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	HostPaths    *api.HostPaths    `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	PortMappings *api.PortMappings `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	Devices      *api.Devices      `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user" create:"optional"`
	Envs         *api.Envs         `charset:"utf8" nullable:"true" list:"user" update:"user" create:"optional"`
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
	if input.Volumes == nil {
		return input, errors.Wrap(httperrors.ErrInputParameter, "volumes cannot be empty")
	}

	// Default DevType to NVIDIA_GPU when callers omit it (UI's "auto by VRAM"
	// path posts {} for each device). Without this the scheduler's
	// (DevType, MemoryMb) aggregation key is empty and the VRAM filter
	// silently no-ops.
	if input.Devices != nil {
		for i := range *input.Devices {
			if (*input.Devices)[i].DevType == "" {
				(*input.Devices)[i].DevType = computeapi.CONTAINER_DEV_NVIDIA_GPU_SHARE
			}
		}
	}

	input.Status = api.STATUS_READY
	return input, nil
}

// GetDetailsSchedulableCheck is the per-row endpoint
// `GET /llm_skus/{id}/schedulable-check?gpu_count=1`.
// It delegates to the scheduler's forecast API so every predicate runs
// (IsolatedDevicePredicate with VRAM, CPU, memory, network, ...) —
// not just a bare VRAM scan. Mirrors GPUStack's `evaluate_models`.
func (sku *SLLMSku) GetDetailsSchedulableCheck(
	ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
) (*api.LLMSchedulableCheckOutput, error) {
	skuBase := &sku.SLLMSkuBase
	out := &api.LLMSchedulableCheckOutput{
		VramClaimMb: skuBase.VramClaimMb,
		GpuCount:    1,
	}

	if query != nil {
		if gc, _ := query.Int("gpu_count"); gc > 0 {
			out.GpuCount = int(gc)
		}
	}

	devCount := 0
	if skuBase.Devices != nil {
		devCount = len(*skuBase.Devices)
	}
	if devCount == 0 {
		out.Reason = "SKU has no devices configured"
		return out, nil
	}
	if out.VramClaimMb <= 0 {
		// Auto-compute from mounted InstantModels. Same logic as
		// llm_deployment_create_task.createSkuAndReconcile.
		mountedIds := sku.GetMountedModels()
		var maxWeight int64
		for _, id := range mountedIds {
			obj, err := GetInstantModelManager().FetchById(id)
			if err != nil {
				continue
			}
			if w := obj.(*SInstantModel).WeightSizeBytes; w > maxWeight {
				maxWeight = w
			}
		}
		if maxWeight > 0 {
			out.VramClaimMb = vram.EstimateClaimMb(maxWeight, sku.LLMType)
			out.PerDevMinMb = (out.VramClaimMb + out.GpuCount - 1) / out.GpuCount
		} else {
			out.Reason = "Auto VRAM calculation failed — mounted instant models have unknown weight (not yet backfilled)"
			return out, nil
		}
	}
	out.PerDevMinMb = (out.VramClaimMb + out.GpuCount - 1) / out.GpuCount

	// --- build a minimal ScheduleInput so the scheduler runs predicates
	cpu := skuBase.Cpu
	if cpu <= 0 {
		cpu = 4
	}
	mem := skuBase.Memory
	if mem <= 0 {
		mem = 4096
	}
	isoDevs := make([]*computeapi.IsolatedDeviceConfig, 0, out.GpuCount)
	for i := 0; i < out.GpuCount; i++ {
		// If the SKU pins specific device details, forward them; otherwise
		// GPU type-only (NVIDIA_GPU default) so the VRAM filter drives
		// placement.
		devSpec := computeapi.IsolatedDeviceConfig{
			DevType:  computeapi.CONTAINER_DEV_NVIDIA_GPU,
			MemoryMb: out.PerDevMinMb,
		}
		if i < devCount {
			src := (*skuBase.Devices)[i]
			if src.DevType != "" {
				devSpec.DevType = src.DevType
			}
			devSpec.Model = src.Model
			devSpec.DevicePath = src.DevicePath
		}
		isoDevs = append(isoDevs, &devSpec)
	}

	input := &schedulerapi.ScheduleInput{
		ServerConfig: schedulerapi.ServerConfig{
			ServerConfigs: &computeapi.ServerConfigs{
				Hypervisor:      computeapi.HOST_TYPE_CONTAINER,
				Count:           1,
				IsolatedDevices: isoDevs,
				Disks: []*computeapi.DiskConfig{
					{SizeMb: 10240, DiskType: "data"},
				},
			},
			Ncpu:   cpu,
			Memory: mem,
		},
	}

	s := auth.GetAdminSession(ctx, "")
	canCreate, raw, err := schedulermodules.SchedManager.DoScheduleForecast(s, input, 1)
	if err != nil {
		return nil, errors.Wrap(err, "scheduler forecast")
	}

	// --- translate forecast result → LLM output
	out.Schedulable = canCreate
	out.Reason = "Scheduler forecast completed — see hosts for qualifying candidates"

	candidates, _ := raw.GetArray("candidates")
	out.TotalGpuHosts = len(candidates)
	for _, c := range candidates {
		hostID, _ := c.GetString("host_id")
		hostName, _ := c.GetString("name")
		out.Hosts = append(out.Hosts, api.LLMSchedulableHostInfo{
			HostId:   hostID,
			HostName: hostName,
		})
		if hostID != "" {
			out.QualifiedHosts++
		}
	}

	if !canCreate {
		var reasons []string
		notAllow, _ := raw.GetArray("not_allow_reasons")
		for _, r := range notAllow {
			if s, _ := r.GetString(); s != "" {
				reasons = append(reasons, s)
			}
		}
		if len(reasons) > 0 {
			out.Reason = fmt.Sprintf("not schedulable: %s", reasons[0])
		} else {
			out.Reason = "not schedulable — no host satisfies all predicates"
		}
		fc, _ := raw.Get("filtered_candidates")
		out.FilteredCandidates = fc
	}

	return out, nil
}

func (skuBase *SLLMSkuBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMSkuBaseUpdateInput) (api.LLMSkuBaseUpdateInput, error) {
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = skuBase.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate SharableVirtualResourceBaseUpdateInput")
	}

	volumes := []api.Volume{}
	if err := jsonutils.Marshal(skuBase.Volumes).Unmarshal(&volumes); err != nil {
		return input, errors.Wrapf(err, "Unmarshal Volumes")
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

	return input, nil
}
