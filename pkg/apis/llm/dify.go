package llm

type DifyCustomizedEnv struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DifyCustomized struct {
	// Define custom environment variables here
	CustomizedEnvs []*DifyCustomizedEnv `json:"customized_envs,omitempty"`
	Registry       string               `json:"registry"`
}

type DifyListInput struct {
	LLMBaseListInput

	DifySku string `json:"dify_sku"`
}

type DifyCreateInput struct {
	LLMBaseCreateInput

	DifySkuId string
}
