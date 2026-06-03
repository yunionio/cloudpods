package llm_container

import (
	"context"
	"fmt"
	"path"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	hermesAgentDataDir = "/opt/data"
)

func init() {
	models.RegisterLLMContainerDriver(newHermesAgent())
}

type hermesAgent struct {
	baseDriver
}

func newHermesAgent() models.ILLMContainerDriver {
	return &hermesAgent{
		baseDriver: newBaseDriver(api.LLM_CONTAINER_HERMES_AGENT),
	}
}

func (h *hermesAgent) GetSpec(sku *models.SLLMSku) interface{} {
	if sku == nil || sku.LLMType != string(api.LLM_CONTAINER_HERMES_AGENT) || sku.LLMSpec == nil || sku.LLMSpec.HermesAgent == nil {
		return nil
	}
	return sku.LLMSpec.HermesAgent
}

func (h *hermesAgent) GetEffectiveSpec(llm *models.SLLM, sku *models.SLLMSku) interface{} {
	var skuSpec *api.LLMSpecHermesAgent
	if spec := h.GetSpec(sku); spec != nil {
		skuSpec = spec.(*api.LLMSpecHermesAgent)
	}
	var llmSpec *api.LLMSpecHermesAgent
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.HermesAgent != nil {
		llmSpec = llm.LLMSpec.HermesAgent
	}
	return mergeHermesAgentSpecs(skuSpec, llmSpec)
}

func copyHermesAgentSpec(spec *api.LLMSpecHermesAgent) *api.LLMSpecHermesAgent {
	if spec == nil {
		return nil
	}
	out := *spec
	out.LLMId = strings.TrimSpace(out.LLMId)
	out.LLMUrl = strings.TrimSpace(out.LLMUrl)
	out.Model = strings.TrimSpace(out.Model)
	out.ApiKey = strings.TrimSpace(out.ApiKey)
	return &out
}

func mergeHermesAgentSpecs(base, override *api.LLMSpecHermesAgent) *api.LLMSpecHermesAgent {
	if base == nil && override == nil {
		return nil
	}
	out := copyHermesAgentSpec(base)
	if out == nil {
		out = &api.LLMSpecHermesAgent{}
	}
	if override == nil {
		return out
	}
	ov := copyHermesAgentSpec(override)
	if ov.LLMId != "" {
		out.LLMId = ov.LLMId
	}
	if ov.LLMUrl != "" {
		out.LLMUrl = ov.LLMUrl
	}
	if ov.Model != "" {
		out.Model = ov.Model
	}
	if ov.ApiKey != "" {
		out.ApiKey = ov.ApiKey
	}
	if ov.ContextLength > 0 {
		out.ContextLength = ov.ContextLength
	}
	return out
}

func ensureOpenAICompatibleBaseURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		return ""
	}
	if strings.HasSuffix(base, "/v1") {
		return base
	}
	return base + "/v1"
}

func getVLLMServedModelName(llm *models.SLLM, sku *models.SLLMSku) string {
	preferred := ""
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.Vllm != nil {
		preferred = strings.TrimSpace(llm.LLMSpec.Vllm.PreferredModel)
	}
	if preferred == "" && sku != nil && sku.LLMSpec != nil && sku.LLMSpec.Vllm != nil {
		preferred = strings.TrimSpace(sku.LLMSpec.Vllm.PreferredModel)
	}
	if preferred == "" {
		return ""
	}
	return path.Base(preferred)
}

func resolveHermesAgentSpec(ctx context.Context, userCred mcclient.TokenCredential, spec *api.LLMSpecHermesAgent) (*api.LLMSpecHermesAgent, error) {
	out := copyHermesAgentSpec(spec)
	if out == nil {
		return nil, nil
	}
	if out.LLMId == "" {
		return out, nil
	}

	llmObj, err := models.GetLLMManager().FetchByIdOrName(ctx, userCred, out.LLMId)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch target LLM %s", out.LLMId)
	}
	targetLLM := llmObj.(*models.SLLM)
	targetSku, err := targetLLM.GetLLMSku(targetLLM.LLMSkuId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch target LLM SKU")
	}
	if targetSku.LLMType != string(api.LLM_CONTAINER_VLLM) {
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "Hermes llm_id target must be vllm, got %s", targetSku.LLMType)
	}
	out.LLMId = targetLLM.Id
	if out.LLMUrl == "" {
		accessURL, err := targetLLM.GetLLMAccessUrlInfo(ctx, userCred, jsonutils.NewDict())
		if err != nil {
			return nil, errors.Wrap(err, "get target LLM access URL")
		}
		if accessURL == nil || accessURL.LoginUrl == "" {
			return nil, errors.Wrap(httperrors.ErrInvalidStatus, "target LLM access URL is empty")
		}
		out.LLMUrl = ensureOpenAICompatibleBaseURL(accessURL.LoginUrl)
	}
	if out.Model == "" {
		out.Model = getVLLMServedModelName(targetLLM, targetSku)
	}
	return out, nil
}

func (h *hermesAgent) ValidateLLMSkuCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	input, err := h.baseDriver.ValidateLLMSkuCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	spec, err := h.ValidateLLMCreateSpec(ctx, userCred, nil, input.LLMSpec)
	if err != nil {
		return nil, err
	}
	if spec == nil {
		spec = &api.LLMSpec{HermesAgent: &api.LLMSpecHermesAgent{}}
	} else if spec.HermesAgent == nil {
		spec.HermesAgent = &api.LLMSpecHermesAgent{}
	}
	input.LLMSpec = spec
	return input, nil
}

func (h *hermesAgent) ValidateLLMSkuUpdateData(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSkuUpdateInput) (*api.LLMSkuUpdateInput, error) {
	input, err := h.baseDriver.ValidateLLMSkuUpdateData(ctx, userCred, sku, input)
	if err != nil {
		return nil, err
	}
	if input.LLMSpec == nil {
		return input, nil
	}
	fakeLLM := &models.SLLM{LLMSpec: sku.LLMSpec}
	spec, err := h.ValidateLLMUpdateSpec(ctx, userCred, fakeLLM, input.LLMSpec)
	if err != nil {
		return nil, err
	}
	input.LLMSpec = spec
	if input.LLMSpec != nil && input.LLMSpec.HermesAgent == nil {
		input.LLMSpec.HermesAgent = &api.LLMSpecHermesAgent{}
	}
	return input, nil
}

func (h *hermesAgent) ValidateLLMCreateSpec(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil {
		return nil, nil
	}
	var skuSpec *api.LLMSpecHermesAgent
	if sku != nil && sku.LLMSpec != nil && sku.LLMSpec.HermesAgent != nil {
		skuSpec = sku.LLMSpec.HermesAgent
	}
	spec := mergeHermesAgentSpecs(skuSpec, input.HermesAgent)
	if spec == nil {
		spec = &api.LLMSpecHermesAgent{}
	}
	spec, err := resolveHermesAgentSpec(ctx, userCred, spec)
	if err != nil {
		return nil, err
	}
	return &api.LLMSpec{HermesAgent: spec}, nil
}

func (h *hermesAgent) ValidateLLMUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil || input.HermesAgent == nil {
		return input, nil
	}
	var base *api.LLMSpecHermesAgent
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.HermesAgent != nil {
		base = llm.LLMSpec.HermesAgent
	}
	spec := mergeHermesAgentSpecs(base, input.HermesAgent)
	spec, err := resolveHermesAgentSpec(ctx, userCred, spec)
	if err != nil {
		return nil, err
	}
	return &api.LLMSpec{HermesAgent: spec}, nil
}

func hasHermesAgentModelConfig(spec *api.LLMSpecHermesAgent) bool {
	return spec != nil && (spec.LLMUrl != "" || spec.Model != "" || spec.ApiKey != "" || spec.ContextLength > 0)
}

func buildHermesAgentEntrypointScript(spec *api.LLMSpecHermesAgent) string {
	lines := []string{"set -e"}
	if hasHermesAgentModelConfig(spec) {
		lines = append(lines,
			"HERMES_CMD='/opt/hermes/.venv/bin/hermes'",
			`if [ ! -x "$HERMES_CMD" ]; then`,
			`  HERMES_CMD="$(command -v hermes || true)"`,
			"fi",
			`if [ -z "$HERMES_CMD" ] || [ ! -x "$HERMES_CMD" ]; then`,
			`  echo "hermes command not found at /opt/hermes/.venv/bin/hermes or PATH" >&2`,
			"  exit 127",
			"fi",
		)
		lines = append(lines, `"$HERMES_CMD" config set model.provider custom`)
		if spec.ContextLength > 0 {
			lines = append(lines, fmt.Sprintf(`"$HERMES_CMD" config set model.context_length %d`, spec.ContextLength))
		}
		if spec.LLMUrl != "" {
			lines = append(lines, fmt.Sprintf(`"$HERMES_CMD" config set model.base_url %s`, shellQuoteSingle(spec.LLMUrl)))
		}
		if spec.Model != "" {
			lines = append(lines, fmt.Sprintf(`"$HERMES_CMD" config set model.default %s`, shellQuoteSingle(spec.Model)))
		}
		apiKey := spec.ApiKey
		if apiKey == "" {
			apiKey = "EMPTY"
		}
		lines = append(lines, fmt.Sprintf(`"$HERMES_CMD" config set model.api_key %s`, shellQuoteSingle(apiKey)))
		lines = append(lines, `"$HERMES_CMD" config migrate`)
	}
	lines = append(lines, "exec /init")
	return strings.Join(lines, "\n")
}

func (h *hermesAgent) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	diskIndex := 0

	hermesVols := []*commonapi.ContainerVolumeMount{
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "config",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: desktopConfigDir,
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "home",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: homeDir,
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "data",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: hermesAgentDataDir,
		},
	}
	hermesInner := desktopWebtopImageBaseContainerSpec(image)
	hermesInner.Envs = append(desktopWebtopCommonEnvs(llm.GetId(), "Cloudpods Desktop"),
		models.NewEnv("HERMES_HOME", hermesAgentDataDir),
		&commonapi.ContainerKeyValue{Key: "HERMES_WEB_DIST", Value: "/opt/hermes/hermes_cli/web_dist"},
	)
	effSpec := (*api.LLMSpecHermesAgent)(nil)
	if eff := h.GetEffectiveSpec(llm, sku); eff != nil {
		effSpec = eff.(*api.LLMSpecHermesAgent)
	}
	if hasHermesAgentModelConfig(effSpec) {
		hermesInner.Command = []string{"/bin/sh", "-c"}
		hermesInner.Args = []string{buildHermesAgentEntrypointScript(effSpec)}
	} else if effSpec != nil && effSpec.LLMId != "" {
		log.Infof("hermes-agent %s has llm_id %s without resolved llm_url/model", llm.GetId(), effSpec.LLMId)
	}
	hermesSpec := computeapi.ContainerSpec{
		ContainerSpec: hermesInner,
		VolumeMounts:  hermesVols,
		RootFs:        desktopContainerRootFs(&diskIndex),
	}
	return &computeapi.PodContainerCreateInput{
		Name:          fmt.Sprintf("%s-%d", llm.GetName(), 0),
		ContainerSpec: hermesSpec,
	}
}

func (h *hermesAgent) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	return []*computeapi.PodContainerCreateInput{
		h.GetContainerSpec(ctx, llm, image, sku, props, devices, diskId),
	}
}

func (h *hermesAgent) GetLLMAccessUrlInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *models.LLMAccessInfoInput) (*api.LLMAccessUrlInfo, error) {
	return models.GetLLMAccessUrlInfo(ctx, userCred, llm, input, "https", api.LLM_DESKTOP_DEFAULT_PORT)
}

// GetLoginInfo returns desktop web UI login credentials (same defaults as container env).
func (h *hermesAgent) GetLoginInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) (*api.LLMAccessInfo, error) {
	return getDesktopWebUILoginInfo(ctx, llm)
}
