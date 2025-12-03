package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
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
	data.VmemSize = skuBase.Memory + 1
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
	if skuBase.Devices != nil && !skuBase.Devices.IsZero() {
		data.IsolatedDevices = make([]*computeapi.IsolatedDeviceConfig, 0)
		devices := *skuBase.Devices
		for i := 0; i < len(devices); i++ {
			isolatedDevice := &computeapi.IsolatedDeviceConfig{
				DevType:    devices[i].DevType,
				Model:      devices[i].Model,
				DevicePath: devices[i].DevicePath,
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
	bandwidth := llmBase.BandwidthMb
	if bandwidth == 0 {
		bandwidth = skuBase.BandwidthMb
	}

	network := &computeapi.NetworkConfig{
		BwLimit: bandwidth,
		NetType: computeapi.TNetworkType(skuBase.NetworkType),
	}
	if skuBase.NetworkType == string(computeapi.NETWORK_TYPE_HOSTLOCAL) {
		network.PortMappings = portMappings
	}
	if len(skuBase.NetworkId) > 0 {
		network.Network = skuBase.NetworkId
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
