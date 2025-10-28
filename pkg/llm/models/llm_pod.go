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

func GetLLMPodCreateInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMCreateInput,
	llm *SLLM,
	sku *SLLMModel,
	llmImage *SLLMImage,
	eip string,
) (*computeapi.ServerCreateInput, error) {
	data := computeapi.ServerCreateInput{}
	data.AutoStart = input.AutoStart
	data.ServerConfigs = computeapi.NewServerConfigs()
	data.Hypervisor = computeapi.HYPERVISOR_POD

	postStopCleanupConfgi := PodPostStopCleanupConfig{
		Dirs: []string{
			GetTmpHostPath(llm.GetName()),
		},
	}
	data.Metadata = map[string]string{
		POD_METADATA_POST_STOP_CLEANUP_CONFIG: jsonutils.Marshal(postStopCleanupConfgi).String(),
	}

	data.VcpuCount = sku.Cpu
	data.VmemSize = sku.Memory + 1
	data.Name = input.Name

	// disks
	data.Disks = make([]*computeapi.DiskConfig, 0)
	if sku.Volumes != nil && !sku.Volumes.IsZero() {
		for idx, volume := range *sku.Volumes {
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
	if sku.Devices != nil && !sku.Devices.IsZero() {
		data.IsolatedDevices = make([]*computeapi.IsolatedDeviceConfig, 0)
		devices := *sku.Devices
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
	if sku.PortMappings != nil && !sku.PortMappings.IsZero() {
		// hostTcpPortRange := computeapi.GuestPortMappingPortRange{
		// 	Start: options.Options.HostTcpPortStart,
		// 	End:   options.Options.HostTcpPortEnd,
		// }
		// hostUdpPortRange := computeapi.GuestPortMappingPortRange{
		// 	Start: options.Options.HostUdpPortStart,
		// 	End:   options.Options.HostUdpPortEnd,
		// }
		for _, portInfo := range *sku.PortMappings {
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
	bandwidth := llm.BandwidthMb
	if bandwidth == 0 {
		bandwidth = sku.BandwidthMb
	}
	network := &computeapi.NetworkConfig{
		// NetType:      computeapi.NETWORK_TYPE_HOSTLOCAL,
		BwLimit:      bandwidth,
		PortMappings: portMappings,
	}
	if len(sku.NetworkId) > 0 {
		network.Network = sku.NetworkId
	} else {
		network.NetType = computeapi.TNetworkType(sku.NetworkType)
	}

	data.Networks = []*computeapi.NetworkConfig{
		network,
	}

	data.Count = 1
	data.PreferHost = input.PreferHost

	// enableLxcfs := true

	lcd := llm.GetLLMContainerDriver()
	llmContainer := lcd.GetContainerSpec(ctx, llm, llmImage, sku, nil, nil, "")

	data.Pod = &computeapi.PodCreateInput{
		HostIPC: true,
		Containers: []*computeapi.PodContainerCreateInput{
			llmContainer,
		},
	}

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
