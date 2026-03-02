package llm_container

import (
	"context"
	"fmt"
	"net/http"
	"path"
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

type vllm struct{}

func newVLLM() models.ILLMContainerDriver {
	return new(vllm)
}

func (v *vllm) GetType() api.LLMContainerType {
	return api.LLM_CONTAINER_VLLM
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
		// Fix Error 803
		{
			Key:   "LD_LIBRARY_PATH",
			Value: "/lib64:/usr/local/cuda/lib64:/lib/x86_64-linux-gnu:${LD_LIBRARY_PATH}",
		},
		// Fix Error 803
		{
			Key:   "LD_PRELOAD",
			Value: "/lib/libcuda.so.1 /lib/libnvidia-ptxjitcompiler.so.1 /lib/libnvidia-gpucomp.so",
		},
	}
	if llm.PreferredModel != "" {
		envs = append(envs, &commonapi.ContainerKeyValue{
			Key:   "PREFERRED_MODEL",
			Value: path.Join(api.LLM_VLLM_MODELS_PATH, llm.PreferredModel),
		})
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
	ctrVols := []*commonapi.ContainerVolumeMount{
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				SubDirectory: api.LLM_VLLM,
				Index:        &diskIndex,
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
	// Build command: resolve model path (PREFERRED_MODEL or first dir), then nohup vllm ... &
	// Env PREFERRED_MODEL is already set in container from GetContainerSpec.
	cmd := fmt.Sprintf(
		`mkdir -p %s; if [ -n "$PREFERRED_MODEL" ] && [ -d "$PREFERRED_MODEL" ]; then model="$PREFERRED_MODEL"; else model=$(ls -d %s/* 2>/dev/null | head -n 1); fi; if [ -z "$model" ]; then echo "NO_MODEL"; exit 0; fi; nohup %s --model "$model" --served-model-name "$(basename "$model")" --port %d --tensor-parallel-size %d --trust-remote-code --swap-space %d > /tmp/vllm.log 2>&1 &`,
		api.LLM_VLLM_MODELS_PATH,
		api.LLM_VLLM_MODELS_PATH,
		api.LLM_VLLM_EXEC_PATH,
		api.LLM_VLLM_DEFAULT_PORT,
		tensorParallelSize,
		swapSpaceGiB,
	)
	output, err := exec(ctx, lc.CmpId, cmd, 30)
	if err != nil {
		return errors.Wrap(err, "exec start vLLM")
	}
	if strings.Contains(output, "NO_MODEL") {
		return nil
	}
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
		return errors.Errorf("vLLM health check timeout after %v, last log: %s", api.LLM_VLLM_HEALTH_CHECK_TIMEOUT, strings.TrimSpace(logTail))
	}
	return errors.Errorf("vLLM health check timeout after %v", api.LLM_VLLM_HEALTH_CHECK_TIMEOUT)
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
		res[relPath] = path.Join(api.LLM_VLLM_BASE_PATH, relPath)
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
	uid := int64(0) // root
	gid := int64(0)
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

func (v *vllm) DownloadModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, tmpDir string, modelName string, modelTag string) (string, []string, error) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return "", nil, errors.Wrap(err, "get llm container")
	}

	// Logic to download model inside the container
	// modelName is expected to be like "facebook/opt-125m"
	targetDir := path.Join(api.LLM_VLLM_MODELS_PATH, modelName)

	// Check if already exists
	checkCmd := fmt.Sprintf("[ -d '%s' ] && echo 'EXIST'", targetDir)
	out, _ := exec(ctx, lc.CmpId, checkCmd, 10)
	if strings.Contains(out, "EXIST") {
		log.Infof("Model %s already exists at %s", modelName, targetDir)
		return modelName, []string{targetDir}, nil
	}

	// Try to use huggingface-cli
	// Assuming container has internet access and tools
	downloadCmd := fmt.Sprintf("mkdir -p %s && huggingface-cli download %s --local-dir %s --local-dir-use-symlinks False", targetDir, modelName, targetDir)

	// If huggingface-cli is missing, try installing it (if pip available)
	// fallback to pip install
	fullCmd := fmt.Sprintf("if ! command -v huggingface-cli &> /dev/null; then pip install -U huggingface_hub; fi; %s", downloadCmd)

	log.Infof("Downloading model %s with cmd: %s", modelName, fullCmd)
	_, err = exec(ctx, lc.CmpId, fullCmd, 3600) // 1 hour timeout for large models
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to download model %s", modelName)
	}

	return modelName, []string{targetDir}, nil
}
