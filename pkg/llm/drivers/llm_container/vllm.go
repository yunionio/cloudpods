package llm_container

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterLLMContainerDriver(newVLLM())
}

type vllm struct {
	baseDriver
}

func newVLLM() models.ILLMContainerDriver {
	return &vllm{baseDriver: newBaseDriver(api.LLM_CONTAINER_VLLM)}
}

// escapeShellSingleQuoted escapes s for use inside a single-quoted shell string (each ' becomes '\”).
func escapeShellSingleQuoted(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

func shellQuoteSingle(s string) string {
	return "'" + escapeShellSingleQuoted(s) + "'"
}

var protectedVLLMArgKeys = map[string]struct{}{
	"model":                {},
	"served-model-name":    {},
	"port":                 {},
	"tensor-parallel-size": {},
}

func validateVLLMArgKey(key string) error {
	if key == "" {
		return errors.Error("vllm arg key is empty")
	}
	if strings.HasPrefix(key, "--") {
		return errors.Errorf("invalid vllm arg key %q: do not include leading --", key)
	}
	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return errors.Errorf("invalid vllm arg key %q", key)
	}
	if _, ok := protectedVLLMArgKeys[key]; ok {
		return errors.Errorf("vllm arg key %q is protected", key)
	}
	return nil
}

func normalizeVLLMCustomizedArgs(args []*api.VllmCustomizedArg) ([]*api.VllmCustomizedArg, error) {
	if len(args) == 0 {
		return nil, nil
	}
	out := make([]*api.VllmCustomizedArg, 0, len(args))
	indexByKey := make(map[string]int, len(args))
	for _, arg := range args {
		if arg == nil {
			continue
		}
		key := strings.TrimSpace(arg.Key)
		if err := validateVLLMArgKey(key); err != nil {
			return nil, err
		}
		next := &api.VllmCustomizedArg{
			Key:   key,
			Value: arg.Value,
		}
		if idx, ok := indexByKey[key]; ok {
			out[idx] = next
			continue
		}
		indexByKey[key] = len(out)
		out = append(out, next)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func mergeVLLMCustomizedArgs(base, overrides []*api.VllmCustomizedArg) ([]*api.VllmCustomizedArg, error) {
	out := make([]*api.VllmCustomizedArg, 0, len(base)+len(overrides))
	indexByKey := make(map[string]int, len(base)+len(overrides))
	appendNormalized := func(items []*api.VllmCustomizedArg) error {
		normalized, err := normalizeVLLMCustomizedArgs(items)
		if err != nil {
			return err
		}
		for _, arg := range normalized {
			if idx, ok := indexByKey[arg.Key]; ok {
				out[idx] = arg
				continue
			}
			indexByKey[arg.Key] = len(out)
			out = append(out, arg)
		}
		return nil
	}
	if err := appendNormalized(base); err != nil {
		return nil, err
	}
	if err := appendNormalized(overrides); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func buildVLLMServeFlags(modelPath string, tensorParallelSize, defaultSwapSpaceGiB int, effSpec *api.LLMSpecVllm) []string {
	modelQuoted := shellQuoteSingle(modelPath)
	flags := []string{
		fmt.Sprintf("--model %s", modelQuoted),
		fmt.Sprintf(`--served-model-name "$(basename %s)"`, modelQuoted),
		fmt.Sprintf("--port %d", api.LLM_VLLM_DEFAULT_PORT),
		fmt.Sprintf("--tensor-parallel-size %d", tensorParallelSize),
		fmt.Sprintf("--swap-space %d", defaultSwapSpaceGiB),
	}
	if effSpec == nil || len(effSpec.CustomizedArgs) == 0 {
		return flags
	}

	normalizedArgs, err := normalizeVLLMCustomizedArgs(effSpec.CustomizedArgs)
	if err != nil {
		log.Errorf("normalize vllm customized args: %v", err)
		return flags
	}
	for _, arg := range normalizedArgs {
		flagName := "--" + arg.Key
		if arg.Key == "swap-space" {
			if arg.Value == "" {
				flags[4] = flagName
			} else {
				flags[4] = fmt.Sprintf("%s %s", flagName, shellQuoteSingle(arg.Value))
			}
			continue
		}
		if arg.Value == "" {
			flags = append(flags, flagName)
			continue
		}
		flags = append(flags, fmt.Sprintf("%s %s", flagName, shellQuoteSingle(arg.Value)))
	}
	return flags
}

func (v *vllm) GetSpec(sku *models.SLLMSku) interface{} {
	if sku == nil || sku.LLMType != string(api.LLM_CONTAINER_VLLM) || sku.LLMSpec == nil || sku.LLMSpec.Vllm == nil {
		return nil
	}
	return sku.LLMSpec.Vllm
}

func (v *vllm) GetEffectiveSpec(llm *models.SLLM, sku *models.SLLMSku) interface{} {
	var skuSpec *api.LLMSpecVllm
	if s := v.GetSpec(sku); s != nil {
		skuSpec = s.(*api.LLMSpecVllm)
	}
	var llmSpec *api.LLMSpecVllm
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.Vllm != nil {
		llmSpec = llm.LLMSpec.Vllm
	}
	if skuSpec == nil && llmSpec == nil {
		return nil
	}
	out := &api.LLMSpecVllm{}
	if skuSpec != nil {
		out.PreferredModel = skuSpec.PreferredModel
		out.CustomizedArgs = skuSpec.CustomizedArgs
	}
	if llmSpec != nil {
		if llmSpec.PreferredModel != "" {
			out.PreferredModel = llmSpec.PreferredModel
		}
	}
	mergedArgs, err := mergeVLLMCustomizedArgs(out.CustomizedArgs, nil)
	if err != nil {
		log.Errorf("normalize sku vllm customized args: %v", err)
		out.CustomizedArgs = nil
	} else {
		out.CustomizedArgs = mergedArgs
	}
	if llmSpec != nil {
		mergedArgs, err = mergeVLLMCustomizedArgs(out.CustomizedArgs, llmSpec.CustomizedArgs)
		if err != nil {
			log.Errorf("merge vllm customized args: %v", err)
		} else {
			out.CustomizedArgs = mergedArgs
		}
	}
	return out
}

func (v *vllm) ValidateLLMSkuCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	input, err := v.baseDriver.ValidateLLMSkuCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}

	// Reuse ValidateLLMCreateSpec; ensure llm_spec.vllm always exists for vLLM SKU.
	spec, err := v.ValidateLLMCreateSpec(ctx, userCred, nil, input.LLMSpec)
	if err != nil {
		return nil, err
	}
	if spec == nil {
		spec = &api.LLMSpec{Vllm: &api.LLMSpecVllm{}}
	} else if spec.Vllm == nil {
		spec.Vllm = &api.LLMSpecVllm{}
	}
	input.LLMSpec = spec
	return input, nil
}

func (v *vllm) ValidateLLMSkuUpdateData(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSkuUpdateInput) (*api.LLMSkuUpdateInput, error) {
	input, err := v.baseDriver.ValidateLLMSkuUpdateData(ctx, userCred, sku, input)
	if err != nil {
		return nil, err
	}
	if input.LLMSpec == nil {
		return input, nil
	}

	// Reuse ValidateLLMUpdateSpec by treating current SKU spec as the "current llm spec".
	fakeLLM := &models.SLLM{LLMSpec: sku.LLMSpec}
	spec, err := v.ValidateLLMUpdateSpec(ctx, userCred, fakeLLM, input.LLMSpec)
	if err != nil {
		return nil, err
	}
	input.LLMSpec = spec
	if input.LLMSpec != nil && input.LLMSpec.Vllm == nil {
		input.LLMSpec.Vllm = &api.LLMSpecVllm{}
	}
	return input, nil
}

// ValidateLLMCreateSpec implements ILLMContainerDriver. Merges preferred_model from SKU when input's is empty.
func (v *vllm) ValidateLLMCreateSpec(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil {
		return nil, nil
	}
	if input.Vllm == nil {
		input.Vllm = &api.LLMSpecVllm{}
	}

	preferred := input.Vllm.PreferredModel
	if preferred == "" && sku != nil && sku.LLMSpec != nil && sku.LLMSpec.Vllm != nil {
		preferred = sku.LLMSpec.Vllm.PreferredModel
	}

	spec := &api.LLMSpecVllm{}
	if sku != nil && sku.LLMSpec != nil && sku.LLMSpec.Vllm != nil {
		base := *sku.LLMSpec.Vllm
		spec = &base
	}
	// Apply create overrides
	if preferred != "" {
		spec.PreferredModel = preferred
	}
	mergedArgs, err := mergeVLLMCustomizedArgs(spec.CustomizedArgs, input.Vllm.CustomizedArgs)
	if err != nil {
		return nil, err
	}
	spec.CustomizedArgs = mergedArgs

	return &api.LLMSpec{Vllm: spec}, nil
}

// ValidateLLMUpdateSpec implements ILLMContainerDriver. Merges preferred_model with current LLM spec; only overwrite when non-empty.
func (v *vllm) ValidateLLMUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil || input.Vllm == nil {
		return input, nil
	}
	base := &api.LLMSpecVllm{}
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.Vllm != nil {
		b := *llm.LLMSpec.Vllm
		base = &b
	}

	// preferred_model: only overwrite when non-empty
	if input.Vllm.PreferredModel != "" {
		base.PreferredModel = input.Vllm.PreferredModel
	}
	mergedArgs, err := mergeVLLMCustomizedArgs(base.CustomizedArgs, input.Vllm.CustomizedArgs)
	if err != nil {
		return nil, err
	}
	base.CustomizedArgs = mergedArgs

	return &api.LLMSpec{Vllm: base}, nil
}

func (v *vllm) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	// Container entrypoint only keeps the container alive; vLLM is started by StartLLM via exec.
	startScript := `mkdir -p ` + api.LLM_VLLM_MODELS_PATH + ` && exec sleep infinity`
	envs := []*commonapi.ContainerKeyValue{
		{
			Key:   "HUGGING_FACE_HUB_CACHE",
			Value: api.LLM_VLLM_CACHE_DIR,
		},
		{
			Key:   "HF_ENDPOINT",
			Value: api.LLM_VLLM_HF_ENDPOINT,
		},
		// // Fix Error 803
		// {
		// 	Key:   "LD_LIBRARY_PATH",
		// 	Value: "/lib64:/usr/local/cuda/lib64:/lib/x86_64-linux-gnu:${LD_LIBRARY_PATH}",
		// },
		// // Fix Error 803
		// {
		// 	Key:   "LD_PRELOAD",
		// 	Value: "/lib/libcuda.so.1 /lib/libnvidia-ptxjitcompiler.so.1 /lib/libnvidia-gpucomp.so",
		// },
	}
	spec := computeapi.ContainerSpec{
		ContainerSpec: commonapi.ContainerSpec{
			Image:             image.ToContainerImage(),
			ImageCredentialId: image.CredentialId,
			Command:           []string{"/bin/sh", "-c"},
			Args:              []string{startScript},
			EnableLxcfs:       true,
			AlwaysRestart:     true,
			Envs:              envs,
		},
	}

	// GPU Devices
	if len(devices) == 0 && (sku.Devices != nil && len(*sku.Devices) > 0) {
		for i := range *sku.Devices {
			index := i
			spec.Devices = append(spec.Devices, &computeapi.ContainerDevice{
				Type: commonapi.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
				IsolatedDevice: &computeapi.ContainerIsolatedDevice{
					Index: &index,
				},
			})
		}
	} else if len(devices) > 0 {
		for i := range devices {
			spec.Devices = append(spec.Devices, &computeapi.ContainerDevice{
				Type: commonapi.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
				IsolatedDevice: &computeapi.ContainerIsolatedDevice{
					Id: devices[i].Id,
				},
			})
		}
	}

	// Volume Mounts
	diskIndex := 0
	postOverlays, err := llm.GetMountedModelsPostOverlay()
	if err != nil {
		log.Errorf("GetMountedModelsPostOverlay failed %s", err)
	}
	ctrVols := []*commonapi.ContainerVolumeMount{
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				SubDirectory: api.LLM_VLLM,
				Index:        &diskIndex,
				PostOverlay:  postOverlays,
			},
			Type:        commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath:   api.LLM_VLLM_BASE_PATH,
			ReadOnly:    false,
			Propagation: commonapi.MOUNTPROPAGATION_PROPAGATION_HOST_TO_CONTAINER,
		},
		{
			// Mount cache dir to save HF cache
			Disk: &commonapi.ContainerVolumeMountDisk{
				SubDirectory: "cache",
				Index:        &diskIndex,
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: "/root/.cache",
			ReadOnly:  false,
		},
	}
	spec.VolumeMounts = append(spec.VolumeMounts, ctrVols...)

	return &computeapi.PodContainerCreateInput{
		ContainerSpec: spec,
	}
}

func (v *vllm) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	return []*computeapi.PodContainerCreateInput{
		v.GetContainerSpec(ctx, llm, image, sku, props, devices, diskId),
	}
}

func (v *vllm) GetLLMAccessUrlInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *models.LLMAccessInfoInput) (*api.LLMAccessUrlInfo, error) {
	return models.GetLLMAccessUrlInfo(ctx, userCred, llm, input, "http", api.LLM_VLLM_DEFAULT_PORT)
}

func buildVLLMHealthCheckURL(networkType, llmIP, hostAccessIP string, accessInfo *models.SAccessInfo) (string, error) {
	if networkType == string(computeapi.NETWORK_TYPE_GUEST) {
		if len(llmIP) == 0 {
			return "", errors.Error("LLM IP is empty for guest network")
		}
		return fmt.Sprintf("http://%s:%d/ping", llmIP, api.LLM_VLLM_DEFAULT_PORT), nil
	}
	if accessInfo != nil && accessInfo.AccessPort > 0 {
		if len(hostAccessIP) == 0 {
			return "", errors.Error("host access IP is empty")
		}
		return fmt.Sprintf("http://%s:%d/ping", hostAccessIP, accessInfo.AccessPort), nil
	}
	if len(llmIP) > 0 {
		return fmt.Sprintf("http://%s:%d/ping", llmIP, api.LLM_VLLM_DEFAULT_PORT), nil
	}
	if len(hostAccessIP) == 0 {
		return "", errors.Error("host access IP is empty")
	}
	return fmt.Sprintf("http://%s:%d/ping", hostAccessIP, api.LLM_VLLM_DEFAULT_PORT), nil
}

// resolveModelPath resolves the model directory inside the container.
// It prefers preferredPath when it exists; otherwise it picks the first directory under models path.
// Returns (empty, nil) when no model is found.
func (v *vllm) resolveModelPath(ctx context.Context, containerId string, preferredPath string) (string, error) {
	preferredQuoted := shellQuoteSingle(preferredPath)
	cmd := fmt.Sprintf(
		`mkdir -p %s;
		preferred=%s;
		if [ -n "$preferred" ] && [ -d "$preferred" ]; then model="$preferred"; else model=$(ls -d %s/* 2>/dev/null | head -n 1); fi;
		if [ -z "$model" ]; then echo "NO_MODEL"; exit 0; fi;
		printf '%%s\n' "$model"`,
		api.LLM_VLLM_MODELS_PATH,
		preferredQuoted,
		api.LLM_VLLM_MODELS_PATH,
	)
	out, err := exec(ctx, containerId, cmd, 30)
	if err != nil {
		return "", errors.Wrap(err, "exec resolve model path")
	}
	out = strings.TrimSpace(out)
	if out == "NO_MODEL" || out == "" {
		return "", nil
	}
	return out, nil
}

// StartLLM starts the vLLM server inside the container via exec, then waits for the health endpoint to be ready.
func (v *vllm) StartLLM(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) error {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return errors.Wrap(err, "get llm container")
	}
	sku, err := llm.GetLLMSku(llm.LLMSkuId)
	if err != nil {
		return errors.Wrap(err, "get llm sku")
	}
	tensorParallelSize := 1
	if sku.Devices != nil && len(*sku.Devices) > 0 {
		tensorParallelSize = len(*sku.Devices)
	}
	swapSpaceGiB := (sku.Memory * 1) / (2 * 1024)
	if swapSpaceGiB < 1 {
		swapSpaceGiB = 1
	}

	effSpec := (*api.LLMSpecVllm)(nil)
	preferredPath := ""
	if eff := v.GetEffectiveSpec(llm, sku); eff != nil {
		effSpec = eff.(*api.LLMSpecVllm)
		if preferred := effSpec.PreferredModel; preferred != "" {
			preferredPath = path.Join(api.LLM_VLLM_MODELS_PATH, preferred)
		}
	}
	modelPath, err := v.resolveModelPath(ctx, lc.CmpId, preferredPath)
	if err != nil {
		return err
	}
	if modelPath == "" {
		return nil // no model
	}

	startCmd := fmt.Sprintf(
		"nohup %s %s > /tmp/vllm.log 2>&1 &",
		api.LLM_VLLM_EXEC_PATH,
		strings.Join(buildVLLMServeFlags(modelPath, tensorParallelSize, swapSpaceGiB, effSpec), " "),
	)
	_, err = exec(ctx, lc.CmpId, startCmd, 30)
	if err != nil {
		log.Errorf("vLLM start failed, exec command: %s", startCmd)
		return errors.Wrapf(err, "exec start vLLM, command: %s", startCmd)
	}
	cmd := startCmd
	// Wait for health endpoint

	input, err := llm.GetLLMAccessInfoInput(ctx, userCred)
	if err != nil {
		return errors.Wrap(err, "get llm url for health check")
	}
	var accessInfo *models.SAccessInfo
	for i := range input.AccessInfos {
		if input.AccessInfos[i].ListenPort == api.LLM_VLLM_DEFAULT_PORT {
			accessInfo = &input.AccessInfos[i]
			break
		}
	}
	if accessInfo == nil && len(input.AccessInfos) > 0 {
		accessInfo = &input.AccessInfos[0]
	}
	healthURL, err := buildVLLMHealthCheckURL(llm.NetworkType, llm.LLMIp, input.HostInternalIp, accessInfo)
	if err != nil {
		return errors.Wrap(err, "build health check url")
	}
	deadline := time.Now().Add(api.LLM_VLLM_HEALTH_CHECK_TIMEOUT)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return errors.Wrap(err, "new health check request")
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "context done while waiting for vLLM")
		case <-time.After(api.LLM_VLLM_HEALTH_CHECK_INTERVAL):
			// continue
		}
	}
	// Optionally read last lines of /tmp/vllm.log for better error message
	logTail, _ := exec(ctx, lc.CmpId, "tail -n 20 /tmp/vllm.log 2>/dev/null || true", 5)
	if logTail != "" {
		return errors.Errorf("vLLM health check timeout after %v, exec command: %s, last log: %s", api.LLM_VLLM_HEALTH_CHECK_TIMEOUT, cmd, strings.TrimSpace(logTail))
	}
	return errors.Errorf("vLLM health check timeout after %v, exec command: %s", api.LLM_VLLM_HEALTH_CHECK_TIMEOUT, cmd)
}

// ILLMContainerInstantApp implementation

func (v *vllm) GetProbedInstantModelsExt(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, mdlIds ...string) (map[string]api.LLMInternalInstantMdlInfo, error) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "get llm container")
	}

	// List directories in models path
	cmd := fmt.Sprintf("du -sk %s/*/", api.LLM_VLLM_MODELS_PATH)
	output, err := exec(ctx, lc.CmpId, cmd, 10)
	if err != nil {
		// If ls fails, maybe no directory yet, return empty
		return make(map[string]api.LLMInternalInstantMdlInfo), nil
	}

	modelsMap := make(map[string]api.LLMInternalInstantMdlInfo)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Size is in KB
		sizeKB, _ := strconv.ParseInt(fields[0], 10, 64)
		fullPath := fields[1]
		name := path.Base(fullPath)
		if name == "" {
			continue
		}
		// We treat the directory name as the model name
		// For vLLM, name usually implies "organization/model" if downloaded from HF, but here we just list local dirs
		modelsMap[name] = api.LLMInternalInstantMdlInfo{
			Name:    name,
			ModelId: name,
			Tag:     "latest",
			Size:    sizeKB * 1024,
		}
	}
	return modelsMap, nil
}

func (v *vllm) DetectModelPaths(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, pkgInfo api.LLMInternalInstantMdlInfo) ([]string, error) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "get llm container")
	}

	modelPath := path.Join(api.LLM_VLLM_MODELS_PATH, pkgInfo.Name)
	checkCmd := fmt.Sprintf("[ -d '%s' ] && echo 'EXIST' || echo 'MISSING'", modelPath)
	output, err := exec(ctx, lc.CmpId, checkCmd, 10)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check file existence")
	}

	if !strings.Contains(output, "EXIST") {
		return nil, errors.Errorf("model directory %s missing", modelPath)
	}

	return []string{modelPath}, nil
}

func (v *vllm) GetImageInternalPathMounts(sApp *models.SInstantModel) map[string]string {
	// Map host paths to container paths
	// For vLLM simple volume mount, this might be 1:1 or based on the base path
	res := make(map[string]string)
	for _, mount := range sApp.Mounts {
		relPath := strings.TrimPrefix(mount, api.LLM_VLLM_BASE_PATH)
		res[relPath] = path.Join(api.LLM_VLLM, relPath)
	}
	return res
}

func (v *vllm) GetSaveDirectories(sApp *models.SInstantModel) (string, []string, error) {
	var filteredMounts []string
	for _, mount := range sApp.Mounts {
		if strings.HasPrefix(mount, api.LLM_VLLM_BASE_PATH) {
			relPath := strings.TrimPrefix(mount, api.LLM_VLLM_BASE_PATH)
			filteredMounts = append(filteredMounts, relPath)
		}
	}
	return "", filteredMounts, nil
}

func (v *vllm) ValidateMounts(mounts []string, mdlName string, mdlTag string) ([]string, error) {
	return mounts, nil
}

func (v *vllm) CheckDuplicateMounts(errStr string, dupIndex int) string {
	return "Duplicate mounts detected"
}

func (v *vllm) GetInstantModelIdByPostOverlay(postOverlay *commonapi.ContainerVolumeMountDiskPostOverlay, mdlNameToId map[string]string) string {
	return ""
}

func (v *vllm) GetDirPostOverlay(dir api.LLMMountDirInfo) *commonapi.ContainerVolumeMountDiskPostOverlay {
	uid := int64(1000)
	gid := int64(1000)
	ov := dir.ToOverlay()
	ov.FsUser = &uid
	ov.FsGroup = &gid
	return &ov
}

func (v *vllm) PreInstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return errors.Wrap(err, "get llm container")
	}
	// Create base directory
	cmd := fmt.Sprintf("mkdir -p %s", api.LLM_VLLM_MODELS_PATH)
	_, err = exec(ctx, lc.CmpId, cmd, 10)
	return err
}

func (v *vllm) InstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, dirs []string, mdlIds []string) error {
	return nil
}

func (v *vllm) UninstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	// Optionally remove the model directory
	// For safety, we might just log or leave it
	return nil
}

func resolveHfdRevision(modelTag string) string {
	if strings.TrimSpace(modelTag) == "" {
		return "main"
	}
	return strings.TrimSpace(modelTag)
}

type hfModelAPIResponse struct {
	Siblings []struct {
		RFilename string `json:"rfilename"`
	} `json:"siblings"`
}

func escapeURLPathPreserveSlash(p string) string {
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func isNonEmptyFile(p string) bool {
	st, err := os.Stat(p)
	if err != nil {
		return false
	}
	return !st.IsDir() && st.Size() > 0
}

func (v *vllm) DownloadModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, tmpDir string, modelName string, modelTag string) (string, []string, error) {
	// Download HF model on host into tmpDir for instant-model import.
	// We place files under tmpDir/huggingface/<repo> so that the archive contains relative paths.
	if strings.TrimSpace(tmpDir) == "" {
		return "", nil, errors.Error("tmpDir is empty")
	}
	if strings.TrimSpace(modelName) == "" {
		return "", nil, errors.Error("modelName is empty")
	}

	modelBase := filepath.Base(modelName)
	localDir := filepath.Join(tmpDir, "huggingface", modelBase)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", nil, errors.Wrap(err, "mkdir local model dir")
	}
	// If already downloaded, short-circuit (directory exists and non-empty).
	if entries, err := os.ReadDir(localDir); err == nil && len(entries) > 0 {
		targetDir := path.Join(api.LLM_VLLM_MODELS_PATH, modelBase)
		log.Infof("Model %s already exists in import dir %s", modelName, localDir)
		return modelName, []string{targetDir}, nil
	}

	rev := resolveHfdRevision(modelTag)
	apiURL := fmt.Sprintf("%s/api/models/%s?revision=%s", api.LLM_VLLM_HF_ENDPOINT, escapeURLPathPreserveSlash(modelName), url.QueryEscape(rev))
	log.Infof("Downloading HF model via HF Mirror API: %s", func() string {
		b, _ := json.Marshal(map[string]string{
			"model":    modelName,
			"revision": rev,
			"dir":      localDir,
			"endpoint": api.LLM_VLLM_HF_ENDPOINT,
			"api":      apiURL,
		})
		return string(b)
	}())
	metaBody, err := llm.HttpGet(ctx, apiURL)
	if err != nil {
		return "", nil, errors.Wrapf(err, "fetch hf model metadata failed: %s", apiURL)
	}
	meta := hfModelAPIResponse{}
	if err := json.Unmarshal(metaBody, &meta); err != nil {
		return "", nil, errors.Wrap(err, "unmarshal hf model metadata")
	}
	if len(meta.Siblings) == 0 {
		return "", nil, errors.Errorf("hf model metadata has no siblings: %s", apiURL)
	}

	for _, s := range meta.Siblings {
		rf := strings.TrimSpace(s.RFilename)
		if rf == "" {
			continue
		}
		dst := filepath.Join(localDir, filepath.FromSlash(rf))
		if isNonEmptyFile(dst) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return "", nil, errors.Wrapf(err, "mkdir for %s", dst)
		}
		fileURL := fmt.Sprintf("%s/%s/resolve/%s/%s", api.LLM_VLLM_HF_ENDPOINT, escapeURLPathPreserveSlash(modelName), url.PathEscape(rev), escapeURLPathPreserveSlash(rf))
		if err := llm.HttpDownloadFile(ctx, fileURL, dst); err != nil {
			return "", nil, errors.Wrapf(err, "download file failed: %s -> %s", fileURL, dst)
		}
	}

	targetDir := path.Join(api.LLM_VLLM_MODELS_PATH, modelBase)
	return modelName, []string{targetDir}, nil
}
