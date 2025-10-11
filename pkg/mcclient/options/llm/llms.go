package llm

import (
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	llmapi "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/pkg/errors"
)

type LLMModelOptions struct {
	MODEL string `help:"Chosen large language model"`
	LLMGgufOptions
}

func (o *LLMModelOptions) parseModel() (*llmapi.OllamaPullModelInput, error) {
	ggufSpec, err := o.getGgufSpec()
	if err != nil {
		return nil, err
	}

	return &llmapi.OllamaPullModelInput{
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
	compute.PodCreateOptions
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
	Gguf string `help:"Import llm from gguf file.\nFormat: file=<path_or_url>,source=<host_or_web>,parameter=[key1=value1|key2=value2...],template=<string>,system=<string>,license=<string>,message=[role1=content1|role2=content2...]\nSource is default to be host\nSupported parameters: num_ctx, repeat_last_n, repeat_penalty, temperature, seed, stop, num_predict, top_k, top_p, min_p\nTemplate: Model prompt template, used to define conversation format\nSystem: System-level prompt, used to set model behavior or role\nLicense: Model license information\nMessage: Predefined conversation messages, format: role=content, supported roles: user, assistant, system, multiple entries separated by |\nFor example:\n\t--gguf \"file=https://tmp.tmp.tmp/qwen3-0.6b.gguf,source=web,parameter=[temperature=0.6|stop=AI assistant:|num_ctx=2048|top_k=100],template={{.System}}\\n{{.Prompt}},system=You are a helpful assistant.,license=MIT,message=[system=Hello|user=Hi there]\"\n\t--gguf file=/root/Downloads/qwen3-0.6b.gguf"`
}

func (op *LLMGgufOptions) getGgufSpec() (*llmapi.OllamaGgufSpec, error) {
	if op.Gguf == "" {
		return nil, nil
	}

	var filePath string
	var source string
	var template *string
	var system *string
	var license *string
	var messages []*llmapi.OllamaModelFileMessage
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
		case strings.HasPrefix(part, "template="):
			val := strings.TrimPrefix(part, "template=")
			template = &val
		case strings.HasPrefix(part, "system="):
			val := strings.TrimPrefix(part, "system=")
			system = &val
		case strings.HasPrefix(part, "license="):
			val := strings.TrimPrefix(part, "license=")
			license = &val
		case strings.HasPrefix(part, "message=[") && strings.HasSuffix(part, "]"):
			msgStr := strings.TrimPrefix(strings.TrimSuffix(part, "]"), "message=[")
			messages = parseMessages(msgStr)
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

	return &llmapi.OllamaGgufSpec{
		GgufFile: filePath,
		Source:   source,
		ModelFile: &llmapi.OllamaModelFileSpec{
			Parameter: paramStruct,
			Template:  template,
			System:    system,
			License:   license,
			Message:   messages,
		},
	}, nil
}

func buildLLMParameter(params map[string]string) (*llmapi.OllamaModelFileParameter, error) {
	if len(params) == 0 {
		return nil, nil
	}
	p := &llmapi.OllamaModelFileParameter{}

	for key, valStr := range params {
		switch key {
		case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_NUM_CTX, llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_LAST_N, llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_SEED, llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_NUM_PREDICT, llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_TOP_K:
			if val, err := strconv.Atoi(valStr); err == nil {
				switch key {
				case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_NUM_CTX:
					p.NumCtx = &val
				case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_LAST_N:
					p.RepeatLastN = &val
				case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_SEED:
					p.Seed = &val
				case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_NUM_PREDICT:
					p.NumPredict = &val
				case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_TOP_K:
					p.TopK = &val
				}
			}
		case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_PENALTY, llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_TEMPERATURE, llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_TOP_P, llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_MIN_P:
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				switch key {
				case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_REPEAT_PENALTY:
					p.RepeatPenalty = &val
				case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_TEMPERATURE:
					p.Temperature = &val
				case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_TOP_P:
					p.TopP = &val
				case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_MIN_P:
					p.MinP = &val
				}
			}
		case llmapi.LLM_OLLAMA_MODELFILE_PARAMETER_STOP:
			p.Stop = &valStr
		default:
			return nil, errors.Errorf("unsupported parameter key: %s", key)
		}
	}

	return p, nil
}

func parseMessages(msgStr string) []*llmapi.OllamaModelFileMessage {
	if msgStr == "" {
		return nil
	}

	var messages []*llmapi.OllamaModelFileMessage
	pairs := strings.Split(msgStr, "|")

	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]

		switch key {
		case llmapi.LLM_OLLAMA_GGUF_MESSAGE_ROLE_USER,
			llmapi.LLM_OLLAMA_GGUF_MESSAGE_ROLE_ASSISTANT,
			llmapi.LLM_OLLAMA_GGUF_MESSAGE_ROLE_SYSTEM:
			messages = append(messages, &llmapi.OllamaModelFileMessage{
				Role:    key,
				Content: val,
			})
		}
	}

	return messages
}
