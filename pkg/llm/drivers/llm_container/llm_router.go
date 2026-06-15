package llm_container

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterLLMContainerDriver(newLLMRouter())
}

type llmRouter struct {
	baseDriver
}

func newLLMRouter() models.ILLMContainerDriver {
	return &llmRouter{baseDriver: newBaseDriver(api.LLM_CONTAINER_LLM_ROUTER)}
}

func (r *llmRouter) GetSpec(sku *models.SLLMSku) interface{} {
	if sku == nil || sku.LLMType != string(api.LLM_CONTAINER_LLM_ROUTER) || sku.LLMSpec == nil || sku.LLMSpec.LLMRouter == nil {
		return nil
	}
	return sku.LLMSpec.LLMRouter
}

func copyLLMRouterSpec(in *api.LLMSpecLLMRouter) *api.LLMSpecLLMRouter {
	if in == nil {
		return nil
	}
	out := *in
	out.Runtime = strings.TrimSpace(out.Runtime)
	out.RouterMethod = strings.TrimSpace(out.RouterMethod)
	out.ConfigPath = strings.TrimSpace(out.ConfigPath)
	out.ModelDir = strings.TrimSpace(out.ModelDir)
	out.RoutePath = strings.TrimSpace(out.RoutePath)
	out.HealthPath = strings.TrimSpace(out.HealthPath)
	out.CandidateMappingPath = strings.TrimSpace(out.CandidateMappingPath)
	if in.CustomizedEnvs != nil {
		out.CustomizedEnvs = append([]*api.LLMRouterEnv(nil), in.CustomizedEnvs...)
	}
	if in.CustomizedArgs != nil {
		out.CustomizedArgs = append([]*api.LLMRouterArg(nil), in.CustomizedArgs...)
	}
	if in.Extra != nil {
		out.Extra = map[string]interface{}{}
		for k, v := range in.Extra {
			out.Extra[k] = v
		}
	}
	return &out
}

func mergeLLMRouterSpecs(base, override *api.LLMSpecLLMRouter) *api.LLMSpecLLMRouter {
	if base == nil && override == nil {
		return nil
	}
	out := copyLLMRouterSpec(base)
	if out == nil {
		out = &api.LLMSpecLLMRouter{}
	}
	ov := copyLLMRouterSpec(override)
	if ov == nil {
		return out
	}
	if ov.Runtime != "" {
		out.Runtime = ov.Runtime
	}
	if ov.RouterMethod != "" {
		out.RouterMethod = ov.RouterMethod
	}
	if ov.ConfigPath != "" {
		out.ConfigPath = ov.ConfigPath
	}
	if ov.ModelDir != "" {
		out.ModelDir = ov.ModelDir
	}
	if ov.RoutePath != "" {
		out.RoutePath = ov.RoutePath
	}
	if ov.HealthPath != "" {
		out.HealthPath = ov.HealthPath
	}
	if ov.CandidateMappingPath != "" {
		out.CandidateMappingPath = ov.CandidateMappingPath
	}
	if ov.CustomizedEnvs != nil {
		out.CustomizedEnvs = ov.CustomizedEnvs
	}
	if ov.CustomizedArgs != nil {
		out.CustomizedArgs = ov.CustomizedArgs
	}
	if ov.Extra != nil {
		if out.Extra == nil {
			out.Extra = map[string]interface{}{}
		}
		for k, v := range ov.Extra {
			out.Extra[k] = v
		}
	}
	return out
}

func (r *llmRouter) normalizeSpec(in *api.LLMSpecLLMRouter) (*api.LLMSpecLLMRouter, error) {
	out := copyLLMRouterSpec(in)
	if out == nil {
		out = &api.LLMSpecLLMRouter{}
	}
	out.Runtime = api.LLM_ROUTER_DEFAULT_RUNTIME
	if out.ModelDir == "" {
		out.ModelDir = api.LLM_ROUTER_DEFAULT_MODEL_DIR
	}
	if out.RoutePath == "" {
		out.RoutePath = api.LLM_ROUTER_DEFAULT_ROUTE_PATH
	}
	if out.HealthPath == "" {
		out.HealthPath = api.LLM_ROUTER_DEFAULT_HEALTH_PATH
	}
	if out.RouterMethod == "" {
		return out, errors.Wrap(httperrors.ErrMissingParameter, "llm_router.router_method is required")
	}
	return out, nil
}

func (r *llmRouter) GetEffectiveSpec(llm *models.SLLM, sku *models.SLLMSku) interface{} {
	var skuSpec *api.LLMSpecLLMRouter
	if spec := r.GetSpec(sku); spec != nil {
		skuSpec = spec.(*api.LLMSpecLLMRouter)
	}
	var llmSpec *api.LLMSpecLLMRouter
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.LLMRouter != nil {
		llmSpec = llm.LLMSpec.LLMRouter
	}
	out, err := r.normalizeSpec(mergeLLMRouterSpecs(skuSpec, llmSpec))
	if err != nil {
		return mergeLLMRouterSpecs(skuSpec, llmSpec)
	}
	return out
}

func (r *llmRouter) ValidateLLMSkuCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	var err error
	input, err = r.baseDriver.ValidateLLMSkuCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	spec, err := r.ValidateLLMCreateSpec(ctx, userCred, nil, input.LLMSpec)
	if err != nil {
		return nil, err
	}
	input.LLMSpec = spec
	return input, nil
}

func (r *llmRouter) ValidateLLMSkuUpdateData(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSkuUpdateInput) (*api.LLMSkuUpdateInput, error) {
	var err error
	input, err = r.baseDriver.ValidateLLMSkuUpdateData(ctx, userCred, sku, input)
	if err != nil {
		return nil, err
	}
	if input.LLMSpec != nil {
		input.LLMSpec, err = r.ValidateLLMUpdateSpec(ctx, userCred, nil, input.LLMSpec)
	}
	return input, err
}

func (r *llmRouter) ValidateLLMCreateSpec(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSpec) (*api.LLMSpec, error) {
	var skuSpec *api.LLMSpecLLMRouter
	if sku != nil && sku.LLMSpec != nil {
		skuSpec = sku.LLMSpec.LLMRouter
	}
	var inputSpec *api.LLMSpecLLMRouter
	if input != nil {
		inputSpec = input.LLMRouter
	}
	spec, err := r.normalizeSpec(mergeLLMRouterSpecs(skuSpec, inputSpec))
	if err != nil {
		return input, err
	}
	return &api.LLMSpec{LLMRouter: spec}, nil
}

func (r *llmRouter) ValidateLLMUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil || input.LLMRouter == nil {
		return input, nil
	}
	spec, err := r.normalizeSpec(input.LLMRouter)
	if err != nil {
		return input, err
	}
	return &api.LLMSpec{LLMRouter: spec}, nil
}

type llmRouterRuntimeArg struct {
	Key   string
	Value string
}

func (a llmRouterRuntimeArg) String() string {
	key := strings.TrimSpace(a.Key)
	if key == "" {
		return ""
	}
	key = strings.TrimPrefix(key, "--")
	if strings.TrimSpace(a.Value) == "" {
		return "--" + key
	}
	return fmt.Sprintf("--%s %s", key, strings.TrimSpace(a.Value))
}

func llmRouterCustomizedArgsToRuntime(args []*api.LLMRouterArg) []llmRouterRuntimeArg {
	out := make([]llmRouterRuntimeArg, 0, len(args))
	for _, arg := range args {
		if arg == nil || strings.TrimSpace(arg.Key) == "" {
			continue
		}
		out = append(out, llmRouterRuntimeArg{Key: arg.Key, Value: arg.Value})
	}
	return out
}

func buildLLMRouterEntrypointScript(spec *api.LLMSpecLLMRouter) string {
	parts := []string{
		api.LLM_ROUTER_EXEC_PATH,
		"--host 0.0.0.0",
		fmt.Sprintf("--port %d", api.LLM_ROUTER_DEFAULT_PORT),
	}
	return strings.Join(parts, " ")
}

func (r *llmRouter) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	effSpec := &api.LLMSpecLLMRouter{RouterMethod: "routerdc"}
	if eff := r.GetEffectiveSpec(llm, sku); eff != nil {
		effSpec = eff.(*api.LLMSpecLLMRouter)
	}
	envs := []*commonapi.ContainerKeyValue{
		models.NewEnv("LLM_ROUTER_RUNTIME", effSpec.Runtime),
		models.NewEnv("LLM_ROUTER_METHOD", effSpec.RouterMethod),
		models.NewEnv("LLM_ROUTER_MODEL_DIR", effSpec.ModelDir),
		models.NewEnv("LLM_ROUTER_ROUTE_PATH", effSpec.RoutePath),
		models.NewEnv("LLM_ROUTER_HEALTH_PATH", effSpec.HealthPath),
	}
	if effSpec.ConfigPath != "" {
		envs = append(envs, models.NewEnv("LLM_ROUTER_CONFIG", effSpec.ConfigPath))
	}
	if effSpec.CandidateMappingPath != "" {
		envs = append(envs, models.NewEnv("LLM_ROUTER_CANDIDATE_MAPPING", effSpec.CandidateMappingPath))
	}
	for _, env := range effSpec.CustomizedEnvs {
		if env != nil && strings.TrimSpace(env.Key) != "" {
			envs = append(envs, models.NewEnv(strings.TrimSpace(env.Key), env.Value))
		}
	}
	spec := computeapi.ContainerSpec{
		ContainerSpec: commonapi.ContainerSpec{
			Image:             image.ToContainerImage(),
			ImageCredentialId: image.CredentialId,
			Command:           []string{"/bin/sh", "-c"},
			Args:              []string{buildLLMRouterEntrypointScript(effSpec)},
			EnableLxcfs:       true,
			AlwaysRestart:     true,
			Envs:              envs,
		},
	}
	appendContainerIsolatedDevices(&spec, llm, sku, devices)
	diskIndex := 0
	spec.VolumeMounts = append(spec.VolumeMounts,
		&commonapi.ContainerVolumeMount{
			Disk: &commonapi.ContainerVolumeMountDisk{
				SubDirectory: api.LLM_ROUTER,
				Index:        &diskIndex,
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: effSpec.ModelDir,
			ReadOnly:  false,
		},
		&commonapi.ContainerVolumeMount{
			Disk: &commonapi.ContainerVolumeMountDisk{
				SubDirectory: "cache",
				Index:        &diskIndex,
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: api.LLM_ROUTER_CACHE_DIR,
			ReadOnly:  false,
		},
	)
	return &computeapi.PodContainerCreateInput{ContainerSpec: spec}
}

func (r *llmRouter) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	return []*computeapi.PodContainerCreateInput{
		r.GetContainerSpec(ctx, llm, image, sku, props, devices, diskId),
	}
}

func (r *llmRouter) GetLLMAccessUrlInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *models.LLMAccessInfoInput) (*api.LLMAccessUrlInfo, error) {
	return models.GetLLMAccessUrlInfo(ctx, userCred, llm, input, "http", api.LLM_ROUTER_DEFAULT_PORT)
}
