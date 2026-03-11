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
	data, err := GetLLMBasePodCreateInput(ctx, userCred, &input.LLMBaseCreateInput, &llm.SLLMBase, &sku.SLLMSkuBase, eip)
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMBasePodCreateInput: ")
	}

	// generate post overlay info
	{
		err = llm.UpdateMountedModelFullNames(ctx, userCred, nil, true, input.LLMImageId, input.LLMSkuId)
		if err != nil {
			return nil, errors.Wrap(err, "UpdateMountedModelFullNames")
		}
	}

	lcd := llm.GetLLMContainerDriver()
	containers := GetDriverPodContainers(ctx, lcd, llm, llmImage, sku, nil, nil, "")

	data.Pod = &computeapi.PodCreateInput{
		HostIPC:    true,
		Containers: containers,
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
	podCtrs := GetDriverPodContainers(ctx, drv, input.LLM, input.Image, input.Sku, input.Input.Property, input.ServerDetails.IsolatedDevices, input.Disk.Id)
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
