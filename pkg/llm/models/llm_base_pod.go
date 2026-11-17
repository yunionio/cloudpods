package models

import (
	"context"
	"fmt"
	"math"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	POD_METADATA_POST_STOP_CLEANUP_CONFIG = "post_stop_cleanup_config"
)

type PodPostStopCleanupConfig struct {
	Dirs []string `json:"dirs"`
}

// isolatedDeviceVramPlan is the VRAM floor used when stamping scheduler
// IsolatedDeviceConfig entries.
//
// HAMI slices still use HamiClaimMb (weight + framework + KV), split across
// HAMI devices. Whole cards (UNLIMITED / EXCLUSIVE / MPS) use WholeCardClaimMb
// (weight + framework only, matching GPUStack estimate_model_vram). A card fits
// when MemorySize * WholeCardUtilization covers its share of that claim.
type isolatedDeviceVramPlan struct {
	HamiClaimMb          int
	WholeCardClaimMb     int
	WholeCardUtilization float64
}

// usesLegacySplit reports a call that only has the historical single claim.
// That claim is split across every device, which keeps HAMI tests and callers
// that have not opted into the GPUStack whole-card fit.
func (p isolatedDeviceVramPlan) usesLegacySplit() bool {
	return p.WholeCardClaimMb == 0 && p.WholeCardUtilization == 0
}

// wholeCardScheduleMemoryMb is the minimum physical memory of one whole card.
// ceil(claim / (count * utilization)) makes MemorySize >= result equivalent to
// sum(MemorySize) * utilization >= claim for equal cards.
func wholeCardScheduleMemoryMb(claimMb, count int, utilization float64) int {
	if claimMb <= 0 || count <= 0 {
		return 0
	}
	if utilization <= 0 || utilization > 1 {
		return (claimMb + count - 1) / count
	}
	return int(math.Ceil(float64(claimMb) / (float64(count) * utilization)))
}

// BuildIsolatedDeviceConfigs builds pod IsolatedDeviceConfig list from SKU/LLM
// devices. Empty or nil devices yield nil. HAMI devices without per-device
// memory_mb require a positive vramClaimMb (evenly split across devices).
func BuildIsolatedDeviceConfigs(effectiveDevices *api.Devices, vramClaimMb int) ([]*computeapi.IsolatedDeviceConfig, error) {
	return buildIsolatedDeviceConfigs(effectiveDevices, isolatedDeviceVramPlan{HamiClaimMb: vramClaimMb})
}

func buildIsolatedDeviceConfigs(effectiveDevices *api.Devices, plan isolatedDeviceVramPlan) ([]*computeapi.IsolatedDeviceConfig, error) {
	if effectiveDevices == nil || effectiveDevices.IsZero() {
		return nil, nil
	}
	devices := make(api.Devices, len(*effectiveDevices))
	copy(devices, *effectiveDevices)
	for i := range devices {
		normalizeLLMSkuDevice(&devices[i])
	}
	if plan.usesLegacySplit() {
		return buildLegacyIsolatedDeviceConfigs(devices, plan.HamiClaimMb)
	}

	hamiCount := 0
	wholeCount := 0
	hasHAMINeedingClaim := false
	for i := range devices {
		if devices[i].SharingMode == computeapi.DEVICE_SHARING_MODE_HAMI {
			hamiCount++
			if devices[i].MemoryMb <= 0 {
				hasHAMINeedingClaim = true
			}
			continue
		}
		wholeCount++
	}
	if hasHAMINeedingClaim && plan.HamiClaimMb <= 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter,
			"vram claim is 0 for HAMI devices: set devices[].memory_mb, mount InstantModel with weight_size_bytes, or use a non-HAMI sharing_mode")
	}
	perHami := 0
	if plan.HamiClaimMb > 0 && hamiCount > 0 {
		perHami = (plan.HamiClaimMb + hamiCount - 1) / hamiCount
	}
	perWhole := wholeCardScheduleMemoryMb(plan.WholeCardClaimMb, wholeCount, plan.WholeCardUtilization)

	out := make([]*computeapi.IsolatedDeviceConfig, 0, len(devices))
	for i := range devices {
		memMb := devices[i].MemoryMb
		if memMb <= 0 {
			if devices[i].SharingMode == computeapi.DEVICE_SHARING_MODE_HAMI {
				memMb = perHami
			} else {
				memMb = perWhole
			}
		}
		out = append(out, isolatedDeviceConfigFromDevice(&devices[i], memMb))
	}
	return out, nil
}

func buildLegacyIsolatedDeviceConfigs(devices api.Devices, vramClaimMb int) ([]*computeapi.IsolatedDeviceConfig, error) {
	hasHAMINeedingClaim := false
	for i := range devices {
		if devices[i].SharingMode == computeapi.DEVICE_SHARING_MODE_HAMI && devices[i].MemoryMb <= 0 {
			hasHAMINeedingClaim = true
			break
		}
	}
	if hasHAMINeedingClaim && vramClaimMb <= 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter,
			"vram claim is 0 for HAMI devices: set devices[].memory_mb, mount InstantModel with weight_size_bytes, or use a non-HAMI sharing_mode")
	}
	perDevFromClaim := 0
	if vramClaimMb > 0 && len(devices) > 0 {
		perDevFromClaim = (vramClaimMb + len(devices) - 1) / len(devices)
	}
	out := make([]*computeapi.IsolatedDeviceConfig, 0, len(devices))
	for i := range devices {
		memMb := devices[i].MemoryMb
		if memMb <= 0 {
			memMb = perDevFromClaim
		}
		out = append(out, isolatedDeviceConfigFromDevice(&devices[i], memMb))
	}
	return out, nil
}

func isolatedDeviceConfigFromDevice(dev *api.Device, memMb int) *computeapi.IsolatedDeviceConfig {
	return &computeapi.IsolatedDeviceConfig{
		DevType:       dev.DevType,
		SharingMode:   dev.SharingMode,
		Vendor:        dev.Vendor,
		Model:         dev.Model,
		DevicePath:    dev.DevicePath,
		MemoryMb:      memMb,
		MemoryRequest: memMb,
		SmUtilLimit:   dev.SmUtilLimit,
	}
}

func GetLLMBasePodCreateInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMBaseCreateInput,
	llmBase *SLLMBase,
	skuBase *SLLMSkuBase,
	plan isolatedDeviceVramPlan,
	eip string,
) (*computeapi.ServerCreateInput, error) {
	data := computeapi.ServerCreateInput{}
	data.AutoStart = input.AutoStart
	data.ServerConfigs = computeapi.NewServerConfigs()
	data.Hypervisor = computeapi.HYPERVISOR_POD

	postStopCleanupConfgi := PodPostStopCleanupConfig{
		Dirs: []string{
			GetTmpHostPath(llmBase.GetName()),
		},
	}
	data.Metadata = map[string]string{
		POD_METADATA_POST_STOP_CLEANUP_CONFIG: jsonutils.Marshal(postStopCleanupConfgi).String(),
	}

	data.VcpuCount = skuBase.Cpu
	data.VmemSize = skuBase.Memory
	// data.Name = input.Name + "-" + seclib.RandomPassword(6)
	data.Name = input.Name

	// disks
	data.Disks = make([]*computeapi.DiskConfig, 0)
	if skuBase.Volumes != nil && !skuBase.Volumes.IsZero() {
		for idx, volume := range *skuBase.Volumes {
			data.Disks = append(data.Disks, &computeapi.DiskConfig{
				DiskType: "data",
				Format:   "raw",
				Fs:       "ext4",
				SizeMb:   volume.SizeMB,
				Index:    idx,
			})
		}
	}

	// isolated devices
	isolatedDevices, err := buildIsolatedDeviceConfigs(getEffectiveDevices(llmBase, skuBase), plan)
	if err != nil {
		return nil, err
	}
	data.IsolatedDevices = isolatedDevices

	// port mappings
	// var portRange *computeapi.GuestPortMappingPortRange
	portMappings := computeapi.GuestPortMappings{}
	if skuBase.PortMappings != nil && !skuBase.PortMappings.IsZero() {
		// hostTcpPortRange := computeapi.GuestPortMappingPortRange{
		// 	Start: options.Options.HostTcpPortStart,
		// 	End:   options.Options.HostTcpPortEnd,
		// }
		// hostUdpPortRange := computeapi.GuestPortMappingPortRange{
		// 	Start: options.Options.HostUdpPortStart,
		// 	End:   options.Options.HostUdpPortEnd,
		// }
		for _, portInfo := range *skuBase.PortMappings {
			remoteIps := portInfo.RemoteIps
			if len(remoteIps) == 0 {
				remoteIps = nil
			}
			// if portInfo.Protocol == "tcp" {
			// 	portRange = &hostTcpPortRange
			// } else {
			// 	portRange = &hostUdpPortRange
			// }
			portMappings = append(portMappings, &computeapi.GuestPortMapping{
				Port:      portInfo.ContainerPort,
				Protocol:  computeapi.GuestPortMappingProtocol(portInfo.Protocol),
				RemoteIps: remoteIps,
				// HostPortRange: portRange,
				Rule: &computeapi.GuestPortMappingRule{
					FirstPortOffset: portInfo.FirstPortOffset,
				},
				Envs: portInfo.Envs,
			})
		}
	}
	var network *computeapi.NetworkConfig
	if len(input.Nets) > 0 {
		network = input.Nets[0]
		networkCopy := *network
		network = &networkCopy
		network.Index = 0
	}

	bandwidth := input.BandwidthMB
	if bandwidth == 0 && network.BwLimit != 0 {
		bandwidth = network.BwLimit
	}
	if bandwidth == 0 && skuBase.Bandwidth != 0 {
		bandwidth = skuBase.Bandwidth
	}
	network.BwLimit = bandwidth

	if len(network.PortMappings) == 0 {
		network.PortMappings = portMappings
	}

	data.Networks = []*computeapi.NetworkConfig{
		network,
	}

	data.Count = 1
	data.PreferHost = input.PreferHost

	data.ProjectId = input.ProjectId
	if len(data.ProjectId) == 0 {
		data.ProjectId = userCred.GetProjectId()
		data.TenantId = userCred.GetTenantId()
	}

	return &data, nil
}

func NewHostDev(path string) *computeapi.ContainerDevice {
	return &computeapi.ContainerDevice{
		Type: apis.CONTAINER_DEVICE_TYPE_HOST,
		Host: &computeapi.ContainerHostDevice{
			HostPath:      path,
			ContainerPath: path,
			Permissions:   "rwm",
		},
	}
}

func NewEnv(key, val string) *apis.ContainerKeyValue {
	return &apis.ContainerKeyValue{
		Key:   key,
		Value: val,
	}
}

func GetTmpHostPath(name string) string {
	return fmt.Sprintf("/tmp/%s", name)
}

func GetSvrLLMContainer(ctrs []*computeapi.PodContainerDesc) *computeapi.PodContainerDesc {
	return ctrs[0]
}
