package llm

import "yunion.io/x/jsonutils"

type LLMInstantModelHuggingFaceSearchOptions struct {
	Q     string `help:"huggingface query string" json:"q"`
	Limit int    `help:"max number of search results" json:"limit"`
	Sort  string `help:"sort order, e.g. downloads|likes|updated" json:"sort"`
}

func (o *LLMInstantModelHuggingFaceSearchOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func (o *LLMInstantModelHuggingFaceSearchOptions) Property() string {
	return "huggingface-search"
}

type LLMInstantModelHuggingFaceRepoInfoOptions struct {
	REPO_ID  string `help:"huggingface repo id, e.g. Qwen/Qwen3-8B" json:"repo_id"`
	REVISION string `help:"huggingface revision, e.g. main" json:"revision"`
}

func (o *LLMInstantModelHuggingFaceRepoInfoOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func (o *LLMInstantModelHuggingFaceRepoInfoOptions) Property() string {
	return "huggingface-repo-info"
}
