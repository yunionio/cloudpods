package models

import (
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/apis/llm"
)

type ContainersToUpdateInput struct {
	LLM           *SLLM
	Input         *llm.LLMRestartTaskInput
	Sku           *SLLMSku
	Image         *SLLMImage
	ServerDetails *computeapi.ServerDetails
	Disk          *computeapi.DiskDetails
	Containers    []*computeapi.SContainer
}

func NewContainersToUpdateInput(
	llm *SLLM,
	input *llm.LLMRestartTaskInput,
	sku *SLLMSku,
	image *SLLMImage,
	serverDetails *computeapi.ServerDetails,
	disk *computeapi.DiskDetails,
	containers []*computeapi.SContainer,
) *ContainersToUpdateInput {
	return &ContainersToUpdateInput{
		LLM:           llm,
		Input:         input,
		Sku:           sku,
		Image:         image,
		ServerDetails: serverDetails,
		Disk:          disk,
		Containers:    containers,
	}
}
