package models

type LLMInstanceModelPendingTask struct {
	LLMId  string
	TaskId string
	SizeGb float64
}

var llmPendingInstantModelTasks []LLMInstanceModelPendingTask

func init() {
	llmPendingInstantModelTasks = make([]LLMInstanceModelPendingTask, 0)
}

func (llm *SLLM) GetInstantModelSizeGb() float64 {
	boolTrue := true
	models, err := llm.FetchModels(nil, &boolTrue, nil)
	if err != nil {
		return 0
	}
	totalSizeGb := 0.0
	for _, model := range models {
		instModel, _ := GetInstantModelManager().GetInstantModelById(model.InstantModelId)
		if instModel == nil {
			continue
		}
		totalSizeGb += float64(instModel.GetActualSizeMb()) * 1024 * 1024 / 1000 / 1000 / 1000
	}
	return totalSizeGb
}

func (llm *SLLM) GetPendingInstantModelSizeGb() float64 {
	var totalSizeGb float64
	for _, task := range llmPendingInstantModelTasks {
		totalSizeGb += task.SizeGb
	}
	return totalSizeGb
}

func (llm *SLLM) GetTotalInstantModelSizeGb() float64 {
	return llm.GetInstantModelSizeGb() + llm.GetPendingInstantModelSizeGb()
}

func (llm *SLLM) insertPendingInstantModelQuota(taskId string, sizeGb float64) {
	llmPendingInstantModelTasks = append(llmPendingInstantModelTasks, LLMInstanceModelPendingTask{
		LLMId:  llm.Id,
		TaskId: taskId,
		SizeGb: sizeGb,
	})
}

func (llm *SLLM) ClearPendingInstantModelQuota(taskId string) {
	for i := range llmPendingInstantModelTasks {
		if llmPendingInstantModelTasks[i].TaskId == taskId {
			llmPendingInstantModelTasks = append(llmPendingInstantModelTasks[:i], llmPendingInstantModelTasks[i+1:]...)
			break
		}
	}
}
