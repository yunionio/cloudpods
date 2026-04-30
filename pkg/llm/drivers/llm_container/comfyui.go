package llm_container

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterLLMContainerDriver(newComfyUI())
}

type comfyui struct {
	baseDriver
}

func newComfyUI() models.ILLMContainerDriver {
	return &comfyui{baseDriver: newBaseDriver(api.LLM_CONTAINER_COMFYUI)}
}

func (c *comfyui) GetSpec(sku *models.SLLMSku) interface{} {
	if sku.LLMSpec == nil {
		return nil
	}
	return sku.LLMSpec.ComfyUI
}

func (c *comfyui) GetEffectiveSpec(llm *models.SLLM, sku *models.SLLMSku) interface{} {
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.ComfyUI != nil {
		return llm.LLMSpec.ComfyUI
	}
	return c.GetSpec(sku)
}

func (c *comfyui) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	var postOverlays []*commonapi.ContainerVolumeMountDiskPostOverlay
	if llm != nil {
		var err error
		postOverlays, err = llm.GetMountedModelsPostOverlay()
		if err != nil {
			log.Errorf("GetMountedModelsPostOverlay failed %s", err)
		}
	}

	spec := computeapi.ContainerSpec{
		ContainerSpec: commonapi.ContainerSpec{
			Image:             image.ToContainerImage(),
			ImageCredentialId: image.CredentialId,
			EnableLxcfs:       true,
			AlwaysRestart:     true,
			Envs: []*commonapi.ContainerKeyValue{
				{
					Key:   "CLI_ARGS",
					Value: "--disable-xformers",
				},
			},
		},
	}

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

	// Volume Mounts, see: https://github.com/YanWenKun/ComfyUI-Docker?tab=readme-ov-file#quick-start---nvidia-gpu
	diskIndex := 0
	ctrVols := []*commonapi.ContainerVolumeMount{
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: api.LLM_COMFYUI_STORAGE_VOLUME_SUBDIR,
				PostOverlay:  postOverlays,
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: "/root",
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: api.LLM_COMFYUI_MODELS_VOLUME_SUBDIR,
			},
			Type:        commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath:   api.LLM_COMFYUI_MODELS_PATH,
			Propagation: commonapi.MOUNTPROPAGATION_PROPAGATION_HOST_TO_CONTAINER,
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "storage-models/hf-hub",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: "/root/.cache/huggingface/hub",
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "storage-models/torch-hub",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: "/root/.cache/torch/hub",
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "storage-user/input",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: path.Join(api.LLM_COMFYUI_BASE_PATH, "input"),
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "storage-user/output",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: path.Join(api.LLM_COMFYUI_BASE_PATH, "output"),
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "storage-user/workflows",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: path.Join(api.LLM_COMFYUI_BASE_PATH, "user/default/workflows"),
		},
	}
	spec.VolumeMounts = append(spec.VolumeMounts, ctrVols...)

	return &computeapi.PodContainerCreateInput{
		ContainerSpec: spec,
	}
}

func (c *comfyui) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	return []*computeapi.PodContainerCreateInput{
		c.GetContainerSpec(ctx, llm, image, sku, props, devices, diskId),
	}
}

func (c *comfyui) GetLLMAccessUrlInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *models.LLMAccessInfoInput) (*api.LLMAccessUrlInfo, error) {
	return models.GetLLMAccessUrlInfo(ctx, userCred, llm, input, "http", 8188)
}

func (c *comfyui) GetProbedInstantModelsExt(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, mdlIds ...string) (map[string]api.LLMInternalInstantMdlInfo, error) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "get llm container")
	}

	cmd := buildComfyUIProbeCommand()
	output, err := exec(ctx, lc.CmpId, cmd, 10)
	if err != nil {
		return make(map[string]api.LLMInternalInstantMdlInfo), nil
	}

	modelsMap := make(map[string]api.LLMInternalInstantMdlInfo)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sizeKB, _ := strconv.ParseInt(fields[0], 10, 64)
		fullPath := path.Clean(fields[1])
		key, info, ok := buildComfyUIProbedModelInfo(fullPath, sizeKB*1024)
		if !ok {
			continue
		}
		modelType := getComfyUILLMTypeByModelPath(fullPath)
		instMdl, _ := models.GetInstantModelManager().FindInstantModelByMountAndLLMType(fullPath, modelType, true)
		if instMdl == nil {
			instMdl, _ = models.GetInstantModelManager().FindInstantModelByLLMType(info.ModelId, info.Tag, modelType, true)
		}
		if instMdl == nil && info.Tag == resolveHfdRevision("") {
			instMdl, _ = models.GetInstantModelManager().FindInstantModelByLLMType(info.ModelId, "", modelType, true)
		}
		if instMdl != nil {
			key = instMdl.Id
			info.ModelId = instMdl.ModelId
			info.Name = instMdl.ModelName
			info.Tag = instMdl.ModelTag
			if len(mdlIds) > 0 && !containsString(mdlIds, key) {
				continue
			}
		} else if len(mdlIds) > 0 {
			continue
		}
		modelsMap[key] = info
	}
	return modelsMap, nil
}

func (c *comfyui) DetectModelPaths(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, pkgInfo api.LLMInternalInstantMdlInfo) ([]string, error) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "get llm container")
	}

	instMdl, _ := models.GetInstantModelManager().FindInstantModelByLLMType(pkgInfo.ModelId, pkgInfo.Tag, api.LLM_CONTAINER_COMFYUI, true)
	if instMdl == nil && pkgInfo.Tag == resolveHfdRevision("") {
		instMdl, _ = models.GetInstantModelManager().FindInstantModelByLLMType(pkgInfo.ModelId, "", api.LLM_CONTAINER_COMFYUI, true)
	}
	if instMdl != nil && len(instMdl.Mounts) > 0 {
		paths := make([]string, 0, len(instMdl.Mounts))
		for _, modelPath := range instMdl.Mounts {
			checkCmd := fmt.Sprintf("[ -e %s ] && echo 'EXIST' || echo 'MISSING'", shellQuoteSingle(modelPath))
			output, err := exec(ctx, lc.CmpId, checkCmd, 10)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check file existence")
			}
			if strings.Contains(output, "EXIST") {
				paths = append(paths, modelPath)
			}
		}
		if len(paths) > 0 {
			return paths, nil
		}
	}

	candidates := []string{}
	for _, modelTypeDir := range getComfyUIModelTypeDirs() {
		candidates = append(candidates, path.Join(api.LLM_COMFYUI_MODELS_PATH, modelTypeDir, buildComfyUIModelDirName(pkgInfo.ModelId, pkgInfo.Tag)))
		if pkgInfo.Name != "" {
			candidates = append(candidates, path.Join(api.LLM_COMFYUI_MODELS_PATH, modelTypeDir, filepath.Base(pkgInfo.Name)))
		}
	}
	for _, modelPath := range candidates {
		checkCmd := fmt.Sprintf("[ -e %s ] && echo 'EXIST' || echo 'MISSING'", shellQuoteSingle(modelPath))
		output, err := exec(ctx, lc.CmpId, checkCmd, 10)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check file existence")
		}
		if strings.Contains(output, "EXIST") {
			return []string{modelPath}, nil
		}
	}
	return nil, errors.Errorf("model directory missing for %s:%s", pkgInfo.ModelId, pkgInfo.Tag)
}

func (c *comfyui) GetImageInternalPathMounts(sApp *models.SInstantModel) map[string]string {
	res := make(map[string]string)
	for _, mount := range sApp.Mounts {
		relPath, ok := getComfyUIModelMountRelPath(mount)
		if !ok {
			continue
		}
		res[relPath] = path.Join(api.LLM_COMFYUI_MODELS_VOLUME_SUBDIR, relPath)
	}
	return res
}

func (c *comfyui) GetSaveDirectories(sApp *models.SInstantModel) (string, []string, error) {
	var filteredMounts []string
	for _, mount := range sApp.Mounts {
		relPath, ok := getComfyUIModelMountRelPath(mount)
		if !ok {
			continue
		}
		filteredMounts = append(filteredMounts, relPath)
	}
	return "", filteredMounts, nil
}

func (c *comfyui) ValidateMounts(mounts []string, mdlName string, mdlTag string) ([]string, error) {
	out := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		cleanMount := path.Clean(strings.TrimSpace(mount))
		if cleanMount == "." || cleanMount == "/" {
			continue
		}
		if _, ok := getComfyUIModelMountRelPath(cleanMount); !ok {
			return nil, errors.Errorf("invalid comfyui model mount %q: must be under %s", mount, api.LLM_COMFYUI_MODELS_PATH)
		}
		out = append(out, cleanMount)
	}
	return out, nil
}

func (c *comfyui) CheckDuplicateMounts(errStr string, dupIndex int) string {
	return "Duplicate mounts detected"
}

func (c *comfyui) GetInstantModelIdByPostOverlay(postOverlay *commonapi.ContainerVolumeMountDiskPostOverlay, mdlNameToId map[string]string) string {
	if postOverlay == nil {
		return ""
	}
	findByPath := func(p string) string {
		modelName, modelTag, ok := parseComfyUIModelDirName(p)
		if !ok {
			return ""
		}
		return mdlNameToId[modelName+":"+modelTag]
	}
	if postOverlay.Image != nil {
		for k, v := range postOverlay.Image.PathMap {
			if mdlId := findByPath(k); mdlId != "" {
				return mdlId
			}
			if mdlId := findByPath(v); mdlId != "" {
				return mdlId
			}
		}
	}
	for _, hostLowerDir := range postOverlay.HostLowerDir {
		if mdlId := findByPath(hostLowerDir); mdlId != "" {
			return mdlId
		}
	}
	return ""
}

func (c *comfyui) GetDirPostOverlay(dir api.LLMMountDirInfo) *commonapi.ContainerVolumeMountDiskPostOverlay {
	uid := int64(1000)
	gid := int64(1000)
	ov := dir.ToOverlay()
	ov.FsUser = &uid
	ov.FsGroup = &gid
	return &ov
}

func (c *comfyui) PreInstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return errors.Wrap(err, "get llm container")
	}
	dirs := make([]string, 0, len(getComfyUIModelTypeDirs()))
	for _, modelTypeDir := range getComfyUIModelTypeDirs() {
		dirs = append(dirs, shellQuoteSingle(path.Join(api.LLM_COMFYUI_MODELS_PATH, modelTypeDir)))
	}
	cmd := fmt.Sprintf("mkdir -p %s", strings.Join(dirs, " "))
	_, err = exec(ctx, lc.CmpId, cmd, 10)
	return err
}

func (c *comfyui) InstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, dirs []string, mdlIds []string) error {
	return nil
}

func (c *comfyui) UninstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	return nil
}

func (c *comfyui) DownloadModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, tmpDir string, modelName string, modelTag string) (string, []string, error) {
	if strings.TrimSpace(tmpDir) == "" {
		return "", nil, errors.Error("tmpDir is empty")
	}
	if strings.TrimSpace(modelName) == "" {
		return "", nil, errors.Error("modelName is empty")
	}

	rev := resolveHfdRevision(modelTag)
	apiURL := fmt.Sprintf("%s/api/models/%s?revision=%s", api.LLM_COMFYUI_HF_ENDPOINT, escapeURLPathPreserveSlash(modelName), url.QueryEscape(rev))
	log.Infof("Downloading HF model for ComfyUI via HF Mirror API: %s", func() string {
		b, _ := json.Marshal(map[string]string{
			"model":    modelName,
			"revision": rev,
			"dir":      tmpDir,
			"endpoint": api.LLM_COMFYUI_HF_ENDPOINT,
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

	filenames := make([]string, 0, len(meta.Siblings))
	for _, s := range meta.Siblings {
		filenames = append(filenames, s.RFilename)
	}
	targets, mounts, err := buildComfyUIDownloadTargets(tmpDir, modelName, filenames)
	if err != nil {
		return "", nil, err
	}

	for _, target := range targets {
		if isNonEmptyFile(target.LocalPath) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target.LocalPath), 0755); err != nil {
			return "", nil, errors.Wrapf(err, "mkdir for %s", target.LocalPath)
		}
		fileURL := fmt.Sprintf("%s/%s/resolve/%s/%s", api.LLM_COMFYUI_HF_ENDPOINT, escapeURLPathPreserveSlash(modelName), url.PathEscape(rev), escapeURLPathPreserveSlash(target.Source))
		if err := llm.HttpDownloadFile(ctx, fileURL, target.LocalPath); err != nil {
			return "", nil, errors.Wrapf(err, "download file failed: %s -> %s", fileURL, target.LocalPath)
		}
	}

	return modelName, mounts, nil
}

func getComfyUIModelMountRelPath(mount string) (string, bool) {
	cleanMount := path.Clean(strings.TrimSpace(mount))
	basePath := path.Clean(api.LLM_COMFYUI_MODELS_PATH)
	if cleanMount == basePath || !strings.HasPrefix(cleanMount, basePath+"/") {
		return "", false
	}
	relPath := strings.TrimPrefix(cleanMount, basePath+"/")
	if relPath == "" || relPath == "." {
		return "", false
	}
	return relPath, true
}

func buildComfyUIModelDirName(modelName string, modelTag string) string {
	modelDir, err := normalizeComfyUIRepoIDPath(modelName)
	if err != nil {
		return strings.TrimSpace(modelName)
	}
	return modelDir
}

func buildComfyUIModelPath(modelName string, modelTag string, llmType api.LLMContainerType) (string, error) {
	modelTypeDir, err := getComfyUIModelDirByLLMType(llmType)
	if err != nil {
		return "", err
	}
	modelDir, err := normalizeComfyUIRepoIDPath(modelName)
	if err != nil {
		return "", err
	}
	return path.Join(api.LLM_COMFYUI_MODELS_PATH, modelTypeDir, modelDir), nil
}

func parseComfyUIModelDirName(dirName string) (string, string, bool) {
	baseName := path.Base(strings.TrimSpace(dirName))
	if modelName, modelTag, ok := parseComfyUILegacyHFModelDirName(baseName); ok {
		return modelName, modelTag, true
	}
	if modelName, modelTag, ok := parseVLLMModelDirName(baseName); ok {
		return modelName, modelTag, true
	}
	repoID, ok := extractComfyUIRepoIDPath(dirName)
	if !ok {
		return "", "", false
	}
	return repoID, resolveHfdRevision(""), true
}

func buildComfyUIProbedModelInfo(modelPath string, sizeBytes int64) (string, api.LLMInternalInstantMdlInfo, bool) {
	modelName, modelTag, ok := parseComfyUIModelDirName(modelPath)
	if !ok {
		return "", api.LLMInternalInstantMdlInfo{}, false
	}
	return modelName + ":" + modelTag, api.LLMInternalInstantMdlInfo{
		Name:    modelName,
		Tag:     modelTag,
		ModelId: modelName,
		Size:    sizeBytes,
	}, true
}

type comfyUIDownloadTarget struct {
	Source    string
	LocalPath string
	Mount     string
}

func buildComfyUIDownloadTargets(tmpDir string, repoID string, filenames []string) ([]comfyUIDownloadTarget, []string, error) {
	splitRepo := hasComfyUISplitModelFiles(filenames)
	candidates := make([]comfyUIDownloadCandidate, 0, len(filenames))
	hasTopLevelByDir := make(map[string]bool)
	for _, filename := range filenames {
		src, ok := cleanComfyUIHFFilename(filename)
		if !ok {
			continue
		}
		modelDir, ok := classifyComfyUIModelFile(src, splitRepo)
		if !ok {
			continue
		}
		candidate := comfyUIDownloadCandidate{
			Source:    src,
			ModelDir:  modelDir,
			Target:    path.Base(src),
			TopLevel:  path.Dir(src) == ".",
			IsGeneric: isComfyUIGenericDiffusersFile(src),
		}
		candidates = append(candidates, candidate)
		if candidate.TopLevel {
			hasTopLevelByDir[modelDir] = true
		}
	}

	targets := make([]comfyUIDownloadTarget, 0, len(candidates))
	mountSet := make(map[string]struct{})
	usedTargets := make(map[string]struct{})
	for _, candidate := range candidates {
		if candidate.IsGeneric && !candidate.TopLevel && hasTopLevelByDir[candidate.ModelDir] {
			continue
		}
		targetName := disambiguateComfyUIDownloadTarget(candidate, usedTargets)
		relPath := path.Join(candidate.ModelDir, targetName)
		mount := path.Join(api.LLM_COMFYUI_MODELS_PATH, relPath)
		targets = append(targets, comfyUIDownloadTarget{
			Source:    candidate.Source,
			LocalPath: filepath.Join(tmpDir, filepath.FromSlash(relPath)),
			Mount:     mount,
		})
		mountSet[mount] = struct{}{}
	}
	if len(targets) == 0 {
		return nil, nil, errors.Errorf("no supported comfyui model files found in %s", repoID)
	}
	mounts := make([]string, 0, len(mountSet))
	for mount := range mountSet {
		mounts = append(mounts, mount)
	}
	sort.Strings(mounts)
	return targets, mounts, nil
}

type comfyUIDownloadCandidate struct {
	Source    string
	ModelDir  string
	Target    string
	TopLevel  bool
	IsGeneric bool
}

func isComfyUIGenericDiffusersFile(filename string) bool {
	base := strings.ToLower(path.Base(filename))
	switch base {
	case "diffusion_pytorch_model.safetensors", "diffusion_pytorch_model.bin", "model.safetensors", "pytorch_model.bin":
		return true
	default:
		return strings.HasPrefix(base, "model-") && (strings.HasSuffix(base, ".safetensors") || strings.HasSuffix(base, ".bin"))
	}
}

func disambiguateComfyUIDownloadTarget(candidate comfyUIDownloadCandidate, used map[string]struct{}) string {
	target := candidate.Target
	key := path.Join(candidate.ModelDir, target)
	if _, ok := used[key]; !ok {
		used[key] = struct{}{}
		return target
	}

	parent := path.Base(path.Dir(candidate.Source))
	if parent != "." && parent != "/" && parent != "" {
		target = sanitizeComfyUIDownloadTargetPart(parent) + "-" + candidate.Target
		key = path.Join(candidate.ModelDir, target)
		if _, ok := used[key]; !ok {
			used[key] = struct{}{}
			return target
		}
	}

	ext := path.Ext(candidate.Target)
	name := strings.TrimSuffix(candidate.Target, ext)
	for i := 2; ; i++ {
		target = fmt.Sprintf("%s-%d%s", name, i, ext)
		key = path.Join(candidate.ModelDir, target)
		if _, ok := used[key]; !ok {
			used[key] = struct{}{}
			return target
		}
	}
}

func sanitizeComfyUIDownloadTargetPart(part string) string {
	part = strings.TrimSpace(part)
	var b strings.Builder
	for _, r := range part {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "model"
	}
	return b.String()
}

func cleanComfyUIHFFilename(filename string) (string, bool) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "", false
	}
	cleanName := path.Clean(filename)
	if cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, "../") || strings.HasPrefix(cleanName, "/") {
		return "", false
	}
	return cleanName, true
}

func hasComfyUISplitModelFiles(filenames []string) bool {
	for _, filename := range filenames {
		cleanName, ok := cleanComfyUIHFFilename(filename)
		if !ok || !isComfyUIModelFile(cleanName) {
			continue
		}
		lower := strings.ToLower(cleanName)
		base := path.Base(lower)
		if hasComfyUIPathPart(lower, "clip") ||
			hasComfyUIPathPart(lower, api.LLM_COMFYUI_TEXT_ENCODERS_DIR) ||
			hasComfyUIPathPartPrefix(lower, "text_encoder") ||
			hasComfyUIPathPart(lower, "vae") ||
			hasComfyUIPathPart(lower, "unet") ||
			hasComfyUIPathPart(lower, "transformer") ||
			hasComfyUIPathPart(lower, api.LLM_COMFYUI_DIFFUSION_MODELS_DIR) ||
			strings.Contains(base, "t5xxl") ||
			strings.HasPrefix(base, "qwen") ||
			strings.HasPrefix(base, "clip_") ||
			base == "ae.safetensors" ||
			strings.Contains(base, "vae") {
			return true
		}
	}
	return false
}

func classifyComfyUIModelFile(filename string, splitRepo bool) (string, bool) {
	cleanName, ok := cleanComfyUIHFFilename(filename)
	if !ok || !isComfyUIModelFile(cleanName) {
		return "", false
	}
	lower := strings.ToLower(cleanName)
	base := path.Base(lower)
	switch {
	case strings.Contains(lower, "ip-adapter") || strings.Contains(lower, "ipadapter"):
		return api.LLM_COMFYUI_IPADAPTER_DIR, true
	case strings.Contains(lower, "controlnet") || strings.Contains(lower, "control-net"):
		return api.LLM_COMFYUI_CONTROLNET_DIR, true
	case strings.Contains(lower, "lora") || strings.Contains(lower, "lycoris"):
		return api.LLM_COMFYUI_LORAS_DIR, true
	case strings.Contains(lower, "clip_vision") || strings.Contains(lower, "clip-vision"):
		return api.LLM_COMFYUI_CLIP_VISION_DIR, true
	case hasComfyUIPathPart(lower, "clip") ||
		hasComfyUIPathPart(lower, api.LLM_COMFYUI_TEXT_ENCODERS_DIR) ||
		hasComfyUIPathPartPrefix(lower, "text_encoder") ||
		strings.Contains(base, "t5xxl") ||
		strings.HasPrefix(base, "qwen") ||
		strings.HasPrefix(base, "umt5") ||
		strings.HasPrefix(base, "clip_"):
		return api.LLM_COMFYUI_TEXT_ENCODERS_DIR, true
	case hasComfyUIPathPart(lower, "vae") ||
		strings.Contains(lower, "autoencoder") ||
		base == "ae.safetensors" ||
		strings.Contains(base, "vae"):
		return api.LLM_COMFYUI_VAE_DIR, true
	case strings.Contains(lower, "upscale") ||
		strings.Contains(lower, "realesrgan") ||
		strings.Contains(lower, "ultrasharp") ||
		strings.Contains(lower, "esrgan"):
		return api.LLM_COMFYUI_UPSCALE_MODELS_DIR, true
	case hasComfyUIPathPart(lower, api.LLM_COMFYUI_EMBEDDINGS_DIR) || strings.Contains(lower, "embedding"):
		return api.LLM_COMFYUI_EMBEDDINGS_DIR, true
	case hasComfyUIPathPart(lower, api.LLM_COMFYUI_STYLE_MODELS_DIR) || strings.Contains(lower, "style_model"):
		return api.LLM_COMFYUI_STYLE_MODELS_DIR, true
	case hasComfyUIPathPart(lower, "unet") ||
		hasComfyUIPathPart(lower, "transformer") ||
		hasComfyUIPathPart(lower, api.LLM_COMFYUI_DIFFUSION_MODELS_DIR) ||
		strings.Contains(base, "unet") ||
		strings.Contains(base, "diffusion") ||
		(splitRepo && strings.Contains(base, "flux")):
		return api.LLM_COMFYUI_DIFFUSION_MODELS_DIR, true
	default:
		return api.LLM_COMFYUI_CHECKPOINTS_DIR, true
	}
}

func isComfyUIModelFile(filename string) bool {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".safetensors", ".ckpt", ".pt", ".pth", ".bin", ".gguf":
		return true
	default:
		return false
	}
}

func hasComfyUIPathPart(filename string, part string) bool {
	for _, item := range strings.Split(filename, "/") {
		if item == part {
			return true
		}
	}
	return false
}

func hasComfyUIPathPartPrefix(filename string, prefix string) bool {
	for _, item := range strings.Split(filename, "/") {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}
	return false
}

func parseComfyUILegacyHFModelDirName(dirName string) (string, string, bool) {
	encoded, ok := strings.CutPrefix(strings.TrimSpace(dirName), "hf-")
	if !ok {
		return "", "", false
	}
	repoPart, tagPart, ok := strings.Cut(encoded, "--")
	if !ok || repoPart == "" || tagPart == "" {
		return "", "", false
	}
	repoID, err := base64.RawURLEncoding.DecodeString(repoPart)
	if err != nil {
		return "", "", false
	}
	tag, err := base64.RawURLEncoding.DecodeString(tagPart)
	if err != nil {
		return "", "", false
	}
	repoIDStr := strings.TrimSpace(string(repoID))
	tagStr := strings.TrimSpace(string(tag))
	if repoIDStr == "" || tagStr == "" {
		return "", "", false
	}
	return repoIDStr, tagStr, true
}

func normalizeComfyUIRepoIDPath(repoID string) (string, error) {
	repoID = strings.TrimSpace(repoID)
	if repoID == "" {
		return "", errors.Error("repo_id is empty")
	}
	repoPath := path.Clean(repoID)
	if repoPath == "." || repoPath == ".." || strings.HasPrefix(repoPath, "../") || strings.HasPrefix(repoPath, "/") {
		return "", errors.Errorf("invalid repo_id path %q", repoID)
	}
	for _, part := range strings.Split(repoPath, "/") {
		if part == "" || part == "." || part == ".." {
			return "", errors.Errorf("invalid repo_id path %q", repoID)
		}
	}
	return repoPath, nil
}

func extractComfyUIRepoIDPath(modelPath string) (string, bool) {
	cleanPath := path.Clean(strings.TrimSpace(modelPath))
	if cleanPath == "." || cleanPath == "/" {
		return "", false
	}
	for _, modelTypeDir := range getComfyUIModelTypeDirs() {
		prefixes := []string{
			path.Join(api.LLM_COMFYUI_MODELS_PATH, modelTypeDir),
			path.Join("/", modelTypeDir),
			path.Join(api.LLM_COMFYUI_MODELS_VOLUME_SUBDIR, modelTypeDir),
		}
		for _, prefix := range prefixes {
			if cleanPath == prefix || !strings.HasPrefix(cleanPath, prefix+"/") {
				continue
			}
			repoPath := strings.TrimPrefix(cleanPath, prefix+"/")
			if normalized, err := normalizeComfyUIRepoIDPath(repoPath); err == nil {
				return normalized, true
			}
			return "", false
		}
	}
	normalized, err := normalizeComfyUIRepoIDPath(cleanPath)
	if err != nil {
		return "", false
	}
	return normalized, true
}

func getComfyUIModelDirByLLMType(llmType api.LLMContainerType) (string, error) {
	switch llmType {
	case api.LLM_CONTAINER_COMFYUI:
		return api.LLM_COMFYUI_CHECKPOINTS_DIR, nil
	default:
		return "", errors.Errorf("unsupported comfyui llm_type %q", llmType)
	}
}

func getComfyUILLMTypeByModelDir(modelDir string) api.LLMContainerType {
	return api.LLM_CONTAINER_COMFYUI
}

func getComfyUILLMTypeByModelPath(modelPath string) api.LLMContainerType {
	cleanPath := path.Clean(strings.TrimSpace(modelPath))
	for _, modelTypeDir := range getComfyUIModelTypeDirs() {
		prefixes := []string{
			path.Join(api.LLM_COMFYUI_MODELS_PATH, modelTypeDir),
			path.Join("/", modelTypeDir),
			path.Join(api.LLM_COMFYUI_MODELS_VOLUME_SUBDIR, modelTypeDir),
		}
		for _, prefix := range prefixes {
			if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+"/") {
				return getComfyUILLMTypeByModelDir(modelTypeDir)
			}
		}
	}
	return getComfyUILLMTypeByModelDir(path.Base(path.Dir(cleanPath)))
}

func getComfyUIModelTypeDirs() []string {
	return []string{
		api.LLM_COMFYUI_CHECKPOINTS_DIR,
		api.LLM_COMFYUI_LORAS_DIR,
		api.LLM_COMFYUI_VAE_DIR,
		api.LLM_COMFYUI_CONTROLNET_DIR,
		api.LLM_COMFYUI_EMBEDDINGS_DIR,
		api.LLM_COMFYUI_UPSCALE_MODELS_DIR,
		api.LLM_COMFYUI_TEXT_ENCODERS_DIR,
		api.LLM_COMFYUI_CLIP_DIR,
		api.LLM_COMFYUI_CLIP_VISION_DIR,
		api.LLM_COMFYUI_UNET_DIR,
		api.LLM_COMFYUI_DIFFUSION_MODELS_DIR,
		api.LLM_COMFYUI_IPADAPTER_DIR,
		"diffusers",
		api.LLM_COMFYUI_STYLE_MODELS_DIR,
	}
}

func buildComfyUIProbeCommand() string {
	filePatterns := make([]string, 0, len(getComfyUIModelTypeDirs()))
	dirPatterns := make([]string, 0, len(getComfyUIModelTypeDirs()))
	for _, modelTypeDir := range getComfyUIModelTypeDirs() {
		modelDir := shellQuoteSingle(path.Join(api.LLM_COMFYUI_MODELS_PATH, modelTypeDir))
		filePatterns = append(filePatterns, modelDir+"/*")
		dirPatterns = append(dirPatterns, modelDir+"/*@*/", modelDir+"/hf-*/", modelDir+"/*/*/")
	}
	return fmt.Sprintf(
		"for f in %s; do [ -f \"$f\" ] && case \"$f\" in *.safetensors|*.ckpt|*.pt|*.pth|*.bin|*.gguf) du -sk \"$f\";; esac; done; for d in %s; do [ -d \"$d\" ] && du -sk \"$d\"; done",
		strings.Join(filePatterns, " "),
		strings.Join(dirPatterns, " "),
	)
}
