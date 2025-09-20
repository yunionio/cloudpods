package compute

import (
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/pkg/errors"
)

type LLMModelOptions struct {
	MODEL string `help:"Chosen large language model"`
	LLMGgufOptions
}

func (o *LLMModelOptions) parseModel() (*computeapi.LLMPullModelInput, error) {
	ggufSpec, err := o.getGgufSpec()
	if err != nil {
		return nil, err
	}

	return &computeapi.LLMPullModelInput{
		Model: o.MODEL,
		Gguf:  ggufSpec,
	}, nil
}

type LLMChangeModelOptions struct {
	LLMIdOptions
	LLMModelOptions
}

func (o *LLMChangeModelOptions) Params() (jsonutils.JSONObject, error) {
	modelParams, err := o.LLMModelOptions.parseModel()
	if err != nil {
		return nil, err
	}

	return jsonutils.Marshal(modelParams), nil

}

type LLMCreateOptions struct {
	PodCreateOptions
	LLMModelOptions
}

func (o *LLMCreateOptions) Params() (jsonutils.JSONObject, error) {
	input, err := o.PodCreateOptions.Params()
	if err != nil {
		return nil, err
	}
	params := input.JSON(input)

	modelParams, err := o.LLMModelOptions.parseModel()
	if err != nil {
		return nil, err
	}

	params.Update(jsonutils.Marshal(modelParams))

	return params, nil
}

type LLMListOptions struct {
	options.BaseListOptions
	GuestId     string `json:"guest_id" help:"guest(pod) id or name"`
	ContainerId string `json:"container_id" help:"container id or name"`
}

func (o *LLMListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMIdsOptions struct {
	ID []string `help:"ID of llms to operate" metavar:"LLM" json:"-"`
}

func (o *LLMIdsOptions) GetIds() []string {
	return o.ID
}

func (o *LLMIdsOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type LLMIdOptions struct {
	ID string `help:"ID or name of the llm" json:"-"`
}

func (o *LLMIdOptions) GetId() string {
	return o.ID
}

func (o *LLMIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type LLMShowOptions struct {
	LLMIdOptions
}

// type LLMStopOptions struct {
// 	LLMIdsOptions
// 	Timeout int  `help:"Stopping timeout" json:"timeout"`
// 	Force   bool `help:"Force stop llm" json:"force"`
// }

// func (o *LLMStopOptions) Params() (jsonutils.JSONObject, error) {
// 	return jsonutils.Marshal(o), nil
// }

// type LLMStartOptions struct {
// 	LLMIdsOptions
// }

type LLMGgufOptions struct {
	Gguf string `help:"Import llm from gguf file.\nFormat: file=<path_or_url>,source=<host_or_web>,parameter=[key1=value1|key2=value2...]\nSource is default to be host\nSupported parameters: num_ctx, repeat_last_n, repeat_penalty, temperature, seed, stop, num_predict, top_k, top_p, min_p\nFor example:\n\t--gguf \"file=https://tmp.tmp.tmp/qwen3-0.6b.gguf,source=web,parameter=[temperature=0.6|stop=AI assistant:|num_ctx=2048|top_k=100]\"\n\t--gguf file=/root/Downloads/qwen3-0.6b.gguf"`
}

func (op *LLMGgufOptions) getGgufSpec() (*computeapi.LLMGgufSpec, error) {
	if op.Gguf == "" {
		return nil, nil
	}

	var filePath string
	var source string
	params := make(map[string]string)

	parts := strings.Split(op.Gguf, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "file="):
			filePath = strings.TrimPrefix(part, "file=")
		case strings.HasPrefix(part, "source="):
			source = strings.TrimPrefix(part, "source=")
		case strings.HasPrefix(part, "parameter=[") && strings.HasSuffix(part, "]"):
			paramStr := strings.TrimPrefix(strings.TrimSuffix(part, "]"), "parameter=[")
			for _, pair := range strings.Split(paramStr, "|") {
				kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
				if len(kv) != 2 {
					return nil, errors.Errorf("invalid parameter key=value pair: %s", pair)
				}
				params[kv[0]] = kv[1]
			}
		default:
			return nil, errors.Errorf("unrecognized part: %s", part)
		}
	}

	if filePath == "" {
		return nil, errors.Errorf("missing required 'file=' part")
	}

	paramStruct, err := buildLLMParameter(params)
	if err != nil {
		return nil, err
	}

	return &computeapi.LLMGgufSpec{
		GgufFile: filePath,
		Source:   source,
		ModelFile: &computeapi.LLMModelFileSpec{
			Parameter: paramStruct,
		},
	}, nil
}

func buildLLMParameter(params map[string]string) (*computeapi.LLMModelFileParameter, error) {
	if len(params) == 0 {
		return nil, nil
	}
	p := &computeapi.LLMModelFileParameter{}

	for key, valStr := range params {
		switch key {
		case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_NUM_CTX, computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_LAST_N, computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_SEED, computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_NUM_PREDICT, computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_TOP_K:
			if val, err := strconv.Atoi(valStr); err == nil {
				switch key {
				case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_NUM_CTX:
					p.NumCtx = &val
				case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_LAST_N:
					p.RepeatLastN = &val
				case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_SEED:
					p.Seed = &val
				case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_NUM_PREDICT:
					p.NumPredict = &val
				case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_TOP_K:
					p.TopK = &val
				}
			}
		case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_PENALTY, computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_TEMPERATURE, computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_TOP_P, computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_MIN_P:
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				switch key {
				case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_PENALTY:
					p.RepeatPenalty = &val
				case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_TEMPERATURE:
					p.Temperature = &val
				case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_TOP_P:
					p.TopP = &val
				case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_MIN_P:
					p.MinP = &val
				}
			}
		case computeapi.LLM_OLLAMA_MODELFILE_PARAMETER_STOP:
			p.Stop = &valStr
		default:
			return nil, errors.Errorf("unsupported parameter key: %s", key)
		}
	}

	return p, nil
}
