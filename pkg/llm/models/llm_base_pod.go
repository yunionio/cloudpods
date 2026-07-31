package models

import (
	"context"
	"fmt"

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

func GetLLMBasePodCreateInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMBaseCreateInput,
	llmBase *SLLMBase,
	skuBase *SLLMSkuBase,
	vramClaimMb int,
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
	effectiveDevices := getEffectiveDevices(llmBase, skuBase)
	if effectiveDevices != nil && !effectiveDevices.IsZero() {
		data.IsolatedDevices = make([]*computeapi.IsolatedDeviceConfig, 0)
		devices := make(api.Devices, len(*effectiveDevices))
		copy(devices, *effectiveDevices)
		for i := range devices {
			normalizeLLMSkuDevice(&devices[i])
		}
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
		// Evenly split estimated vram claim across requested devices when a
		// device does not set memory_mb. Ceiling division so the sum is never
		// less than the claim.
		perDevFromClaim := 0
		if vramClaimMb > 0 && len(devices) > 0 {
			perDevFromClaim = (vramClaimMb + len(devices) - 1) / len(devices)
		}
		for i := 0; i < len(devices); i++ {
			memMb := devices[i].MemoryMb
			if memMb <= 0 {
				memMb = perDevFromClaim
			}
			isolatedDevice := &computeapi.IsolatedDeviceConfig{
				DevType:       devices[i].DevType,
				SharingMode:   devices[i].SharingMode,
				Vendor:        devices[i].Vendor,
				Model:         devices[i].Model,
				DevicePath:    devices[i].DevicePath,
				MemoryMb:      memMb,
				MemoryRequest: memMb,
				SmUtilLimit:   devices[i].SmUtilLimit,
			}
			data.IsolatedDevices = append(data.IsolatedDevices, isolatedDevice)
		}
	}

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
