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
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.Vllm != nil {
		if llm.LLMSpec.Vllm.PreferredModel != "" {
			out := *llm.LLMSpec.Vllm
			return &out
		}
		// llm explicitly present but empty -> fall back to sku default
	}
	if skuSpec != nil {
		out := *skuSpec
		return &out
	}
	return nil
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
	preferred := ""
	if input.Vllm != nil {
		preferred = input.Vllm.PreferredModel
	}
	if preferred == "" && sku != nil && sku.LLMSpec != nil && sku.LLMSpec.Vllm != nil {
		preferred = sku.LLMSpec.Vllm.PreferredModel
	}
	return &api.LLMSpec{Vllm: &api.LLMSpecVllm{PreferredModel: preferred}}, nil
}

// ValidateLLMUpdateSpec implements ILLMContainerDriver. Merges preferred_model with current LLM spec; only overwrite when non-empty.
func (v *vllm) ValidateLLMUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil || input.Vllm == nil {
		return input, nil
	}
	current := ""
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.Vllm != nil {
		current = llm.LLMSpec.Vllm.PreferredModel
	}
	preferred := input.Vllm.PreferredModel
	if preferred == "" {
		preferred = current
	}
	return &api.LLMSpec{Vllm: &api.LLMSpecVllm{PreferredModel: preferred}}, nil
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

func (v *vllm) GetLLMUrl(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) (string, error) {
	// Similar logic to Ollama to determine URL
	server, err := llm.GetServer(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get server")
	}

	networkType := llm.NetworkType
	if networkType == string(computeapi.NETWORK_TYPE_GUEST) {
		if len(llm.LLMIp) == 0 {
			return "", errors.Error("LLM IP is empty for guest network")
		}
		return fmt.Sprintf("http://%s:%d", llm.LLMIp, api.LLM_VLLM_DEFAULT_PORT), nil
	} else {
		// hostlocal
		if len(server.HostAccessIp) == 0 {
			return "", errors.Error("host access IP is empty")
		}
		// Assuming we might map ports or just use the default if host networking isn't strictly port-mapped per instance
		// For simplicity, returning default port on host IP, assuming bridge/direct access or specific port mapping logic exists elsewhere.
		// NOTE: In ollama.go, it queries AccessInfo. Here we simplify.
		return fmt.Sprintf("http://%s:%d", server.HostAccessIp, api.LLM_VLLM_DEFAULT_PORT), nil
	}
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

	preferredPath := ""
	if eff := v.GetEffectiveSpec(llm, sku); eff != nil {
		if preferred := eff.(*api.LLMSpecVllm).PreferredModel; preferred != "" {
			preferredPath = path.Join(api.LLM_VLLM_MODELS_PATH, preferred)
		}
	}
	resolved, err := v.resolveModelAndParams(ctx, lc.CmpId, preferredPath, tensorParallelSize)
	if err != nil {
		return err
	}
	if resolved == nil {
		return nil // no model
	}

	modelEscaped := escapeShellSingleQuoted(resolved.ModelPath)
	startCmd := fmt.Sprintf(
		`nohup %s --model '%s' --served-model-name "$(basename '%s')" --port %d \
		--tensor-parallel-size %d --swap-space %d --enable-prefix-caching \
		--gpu-memory-utilization %s --max-model-len %d --max-num-seqs %d \
		> /tmp/vllm.log 2>&1 &`,
		api.LLM_VLLM_EXEC_PATH,
		modelEscaped,
		modelEscaped,
		api.LLM_VLLM_DEFAULT_PORT,
		tensorParallelSize,
		swapSpaceGiB,
		resolved.GpuUtil,
		resolved.MaxModelLen,
		resolved.MaxNumSeqs,
	)
	_, err = exec(ctx, lc.CmpId, startCmd, 30)
	if err != nil {
		log.Errorf("vLLM start failed, exec command: %s", startCmd)
		return errors.Wrapf(err, "exec start vLLM, command: %s", startCmd)
	}
	cmd := startCmd
	// Wait for health endpoint
	baseURL, err := v.GetLLMUrl(ctx, userCred, llm)
	if err != nil {
		return errors.Wrap(err, "get llm url for health check")
	}
	healthURL := strings.TrimSuffix(baseURL, "/") + "/health"
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
	// We place files under tmpDir/huggingface/<org>/<repo> so that the archive contains relative paths.
	if strings.TrimSpace(tmpDir) == "" {
		return "", nil, errors.Error("tmpDir is empty")
	}
	if strings.TrimSpace(modelName) == "" {
		return "", nil, errors.Error("modelName is empty")
	}

	localDir := filepath.Join(tmpDir, "huggingface", filepath.FromSlash(modelName))
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", nil, errors.Wrap(err, "mkdir local model dir")
	}
	// If already downloaded, short-circuit (directory exists and non-empty).
	if entries, err := os.ReadDir(localDir); err == nil && len(entries) > 0 {
		targetDir := path.Join(api.LLM_VLLM_MODELS_PATH, modelName)
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

	targetDir := path.Join(api.LLM_VLLM_MODELS_PATH, modelName)
	return modelName, []string{targetDir}, nil
}

// vllmResolveResult is the result of resolving model path and estimating vLLM memory params in the container.
type vllmResolveResult struct {
	ModelPath   string
	GpuUtil     string
	MaxModelLen int
	MaxNumSeqs  int
}

// resolveModelAndParams runs one exec in the container to resolve the model path and estimate
// --gpu-memory-utilization, --max-model-len, --max-num-seqs. Returns (nil, nil) when no model is found.
func (v *vllm) resolveModelAndParams(ctx context.Context, containerId string, preferredPath string, tensorParallelSize int) (*vllmResolveResult, error) {
	preferredEscaped := escapeShellSingleQuoted(preferredPath)
	escapedScript := escapeShellSingleQuoted(strings.TrimSpace(api.LLM_VLLM_ESTIMATE_PARAMS_SCRIPT))
	defaultGpuUtil := strconv.FormatFloat(float64(api.LLM_VLLM_DEFAULT_GPU_MEMORY_UTIL), 'f', -1, 64)
	cmd := fmt.Sprintf(
		`mkdir -p %s;
		preferred='%s';
		if [ -n "$preferred" ] && [ -d "$preferred" ]; then model="$preferred"; else model=$(ls -d %s/* 2>/dev/null | head -n 1); fi;
		if [ -z "$model" ]; then echo "NO_MODEL"; exit 0; fi;
		tp=%d;
		vllm_out=$(python3 -c '%s' "$model" "$tp" 2>/dev/null) || true;
		GPU_MEMORY_UTIL=%s; MAX_MODEL_LEN=%d; MAX_NUM_SEQS=%d;
		[ -n "$vllm_out" ] && eval "$vllm_out";
		printf '%%s\n' "$model";
		printf 'GPU_MEMORY_UTIL=%%s MAX_MODEL_LEN=%%s MAX_NUM_SEQS=%%s\n' "$GPU_MEMORY_UTIL" "$MAX_MODEL_LEN" "$MAX_NUM_SEQS"`,
		api.LLM_VLLM_MODELS_PATH,
		preferredEscaped,
		api.LLM_VLLM_MODELS_PATH,
		tensorParallelSize,
		escapedScript,
		defaultGpuUtil,
		api.LLM_VLLM_DEFAULT_MAX_MODEL_LEN,
		api.LLM_VLLM_DEFAULT_MAX_NUM_SEQS,
	)
	out, err := exec(ctx, containerId, cmd, 30)
	if err != nil {
		return nil, errors.Wrapf(err, "exec resolve model and params")
	}
	out = strings.TrimSpace(out)
	if out == "NO_MODEL" {
		return nil, nil
	}
	lines := strings.SplitN(out, "\n", 2)
	if len(lines) < 2 {
		return nil, errors.Errorf("vLLM resolve output missing params line: %s", out)
	}
	res := &vllmResolveResult{
		ModelPath:   strings.TrimSpace(lines[0]),
		GpuUtil:     defaultGpuUtil,
		MaxModelLen: api.LLM_VLLM_DEFAULT_MAX_MODEL_LEN,
		MaxNumSeqs:  api.LLM_VLLM_DEFAULT_MAX_NUM_SEQS,
	}
	for _, f := range strings.Fields(lines[1]) {
		if val, ok := strings.CutPrefix(f, api.LLM_VLLM_RESOLVE_OUTPUT_PREFIX_GPU_UTIL); ok {
			res.GpuUtil = val
		} else if val, ok := strings.CutPrefix(f, api.LLM_VLLM_RESOLVE_OUTPUT_PREFIX_MAX_LEN); ok {
			if n, e := strconv.Atoi(val); e == nil && n > 0 {
				res.MaxModelLen = n
			}
		} else if val, ok := strings.CutPrefix(f, api.LLM_VLLM_RESOLVE_OUTPUT_PREFIX_MAX_NUM_SEQ); ok {
			if n, e := strconv.Atoi(val); e == nil && n > 0 {
				res.MaxNumSeqs = n
			}
		}
	}
	return res, nil
}
