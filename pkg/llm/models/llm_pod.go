package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	llmutil "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

func GetLLMPodCreateInput(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMCreateInput,
	llm *SLLM,
	sku *SLLMSku,
	llmImage *SLLMImage,
	eip string,
) (*computeapi.ServerCreateInput, error) {
	data, err := GetLLMBasePodCreateInput(ctx, userCred, &input.LLMBaseCreateInput, &llm.SLLMBase, &sku.SLLMSkuBase, sku.isolatedDeviceVramPlan(), eip)
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMBasePodCreateInput: ")
	}

	// On a hostlocal network the pod IP is only reachable inside its own host.
	// The leader accepts cross-host distributed traffic. DP only needs the fixed
	// DP coordinator ports, so those are the only ones mapped; PP/TP require
	// arbitrary NCCL ports and must use a VPC (guest) network instead.
	if llm.IsDistLeader() && isHostLocalLLMCreateInput(input) &&
		deploymentParallelModeForLLM(llm) == api.LLM_DEPLOYMENT_PARALLEL_MODE_DP {
		appendDistributedPortMappings(data, api.LLMDistMasterPort(llm.DistGroup), api.LLMDistDPRPCPort(llm.DistGroup))
	}

	// generate post overlay info
	{
		err = llm.UpdateMountedModelFullNames(ctx, userCred, nil, true, input.LLMImageId, input.LLMSkuId)
		if err != nil {
			return nil, errors.Wrap(err, "UpdateMountedModelFullNames")
		}
	}

	lcd := llm.GetLLMContainerDriver()
	containers, err := GetDriverPodContainers(ctx, lcd, llm, llmImage, sku, nil, nil, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetDriverPodContainers")
	}

	data.Pod = &computeapi.PodCreateInput{
		HostIPC:                  true,
		Containers:               containers,
		DisableCgroupCpuLimit:    sku == nil || !skuCgroupLimitEnabled(sku.EnableCgroupCpu),
		DisableCgroupMemoryLimit: sku == nil || !skuCgroupLimitEnabled(sku.EnableCgroupMemory),
	}

	return data, nil
}

func GetLLMContainers(ctx context.Context, llm *SLLM) ([]*computeapi.SContainer, error) {
	admSession := auth.GetAdminSession(ctx, "")
	resp, err := compute.Containers.List(admSession, jsonutils.Marshal(map[string]string{"guest_id": llm.CmpId, "scope": "max"}))
	if err != nil {
		return nil, errors.Wrap(err, "Containers.List")
	}
	ctrs := make([]*computeapi.SContainer, 0)
	for _, c := range resp.Data {
		container := computeapi.SContainer{}
		if err := c.Unmarshal(&container); err != nil {
			return nil, errors.Wrap(err, "Unmarshal")
		}
		ctrs = append(ctrs, &container)
	}
	return ctrs, nil
}

func UpdateContainerIfNeeded(ctx context.Context, userCred mcclient.TokenCredential, ctrId string, input *computeapi.PodContainerCreateInput) (*computeapi.SContainer, error) {
	ctr, err := llmutil.UpdateContainer(ctx, ctrId, func(ctr *computeapi.SContainer) *computeapi.ContainerSpec {
		return &input.ContainerSpec
	})
	if err != nil {
		return nil, errors.Wrapf(err, "UpdateContainer %s", ctrId)
	}
	return ctr, nil
}

func GetContainersToUpdate(
	drv ILLMContainerDriver,
	ctx context.Context,
	input *ContainersToUpdateInput,
) (map[string]*computeapi.PodContainerCreateInput, error) {
	podCtrs, err := GetDriverPodContainers(ctx, drv, input.LLM, input.Image, input.Sku, input.Input.Property, input.ServerDetails.IsolatedDevices, input.Disk.Id)
	if err != nil {
		return nil, errors.Wrap(err, "GetDriverPodContainers")
	}
	ret := make(map[string]*computeapi.PodContainerCreateInput)
	for i := range input.Containers {
		currentCtr := input.Containers[i]
		podCtr, err := drv.MatchContainerToUpdate(currentCtr, podCtrs)
		if err != nil {
			return nil, errors.Wrapf(err, "match container %s to update", currentCtr.Name)
		}
		ret[currentCtr.Id] = podCtr
	}
	return ret, nil
}

// isHostLocalLLMCreateInput reports whether the instance is created on a
// hostlocal network.
func isHostLocalLLMCreateInput(input *api.LLMCreateInput) bool {
	if input == nil || len(input.Nets) == 0 || input.Nets[0] == nil {
		return false
	}
	return input.Nets[0].NetType == computeapi.NETWORK_TYPE_HOSTLOCAL
}

// appendDistributedPortMappings adds TCP port mappings for the given in-pod
// ports to the pod's first NIC. Host ports are allocated by the host agent and
// reported back in the server NIC mappings.
func appendDistributedPortMappings(data *computeapi.ServerCreateInput, ports ...int) {
	if data == nil || len(data.Networks) == 0 || data.Networks[0] == nil {
		return
	}
	net := data.Networks[0]
	if net.PortMappings == nil {
		net.PortMappings = computeapi.GuestPortMappings{}
	}
	for _, port := range ports {
		if port <= 0 || hasGuestPortMapping(net.PortMappings, port) {
			continue
		}
		net.PortMappings = append(net.PortMappings, &computeapi.GuestPortMapping{
			Port:     port,
			Protocol: computeapi.GuestPortMappingProtocolTCP,
		})
	}
}

func hasGuestPortMapping(mappings computeapi.GuestPortMappings, port int) bool {
	for _, pm := range mappings {
		if pm != nil && pm.Port == port {
			return true
		}
	}
	return false
}
