package llm_container

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
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
	models.RegisterLLMContainerDriver(newSGLang())
}

type sglang struct {
	baseDriver
}

func newSGLang() models.ILLMContainerDriver {
	return &sglang{baseDriver: newBaseDriver(api.LLM_CONTAINER_SGLANG)}
}

func (s *sglang) GetSpec(sku *models.SLLMSku) interface{} {
	if sku == nil || sku.LLMType != string(api.LLM_CONTAINER_SGLANG) || sku.LLMSpec == nil || sku.LLMSpec.SGLang == nil {
		return nil
	}
	return sku.LLMSpec.SGLang
}

func (s *sglang) GetEffectiveSpec(llm *models.SLLM, sku *models.SLLMSku) interface{} {
	var skuSpec *api.LLMSpecSGLang
	if spec := s.GetSpec(sku); spec != nil {
		skuSpec = spec.(*api.LLMSpecSGLang)
	}
	var llmSpec *api.LLMSpecSGLang
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.SGLang != nil {
		llmSpec = llm.LLMSpec.SGLang
	}
	if skuSpec == nil && llmSpec == nil {
		return nil
	}
	out := &api.LLMSpecSGLang{}
	if skuSpec != nil {
		out.PreferredModel = skuSpec.PreferredModel
		out.CustomizedArgs = skuSpec.CustomizedArgs
	}
	if llmSpec != nil {
		if llmSpec.PreferredModel != "" {
			out.PreferredModel = llmSpec.PreferredModel
		}
	}
	mergedArgs, err := normalizeSGLangCustomizedArgs(out.CustomizedArgs)
	if err != nil {
		log.Errorf("normalize sku sglang customized args: %v", err)
		out.CustomizedArgs = nil
	} else {
		out.CustomizedArgs = mergedArgs
	}
	if llmSpec != nil {
		mergedArgs, err = mergeSGLangCustomizedArgs(out.CustomizedArgs, llmSpec.CustomizedArgs)
		if err != nil {
			log.Errorf("merge sglang customized args: %v", err)
		} else {
			out.CustomizedArgs = mergedArgs
		}
	}
	return out
}

func (s *sglang) ValidateLLMCreateData(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMCreateInput) (*api.LLMCreateInput, error) {
	llmType := string(api.LLM_CONTAINER_SGLANG)
	if err := models.ValidateRequireDevices(llmType, input.Devices, nil, sku); err != nil {
		return input, err
	}
	if err := models.ValidateRequireMountedModels(llmType, input.MountedModels, nil, sku); err != nil {
		return input, err
	}
	return input, nil
}

func (s *sglang) ValidateLLMUpdateData(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, sku *models.SLLMSku, input *api.LLMUpdateInput) (*api.LLMUpdateInput, error) {
	llmType := string(api.LLM_CONTAINER_SGLANG)
	if err := models.ValidateRequireDevices(llmType, input.Devices, llm.Devices, sku); err != nil {
		return input, err
	}
	if err := models.ValidateRequireMountedModels(llmType, input.MountedModels, llm.MountedModels, sku); err != nil {
		return input, err
	}
	return input, nil
}

func (s *sglang) ValidateLLMSkuCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMSkuCreateInput) (*api.LLMSkuCreateInput, error) {
	input, err := s.baseDriver.ValidateLLMSkuCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}

	spec, err := s.ValidateLLMCreateSpec(ctx, userCred, nil, input.LLMSpec)
	if err != nil {
		return nil, err
	}
	if spec == nil {
		spec = &api.LLMSpec{SGLang: &api.LLMSpecSGLang{}}
	} else if spec.SGLang == nil {
		spec.SGLang = &api.LLMSpecSGLang{}
	}
	input.LLMSpec = spec
	return input, nil
}

func (s *sglang) ValidateLLMSkuUpdateData(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSkuUpdateInput) (*api.LLMSkuUpdateInput, error) {
	input, err := s.baseDriver.ValidateLLMSkuUpdateData(ctx, userCred, sku, input)
	if err != nil {
		return nil, err
	}
	if input.LLMSpec == nil {
		return input, nil
	}

	fakeLLM := &models.SLLM{LLMSpec: sku.LLMSpec}
	spec, err := s.ValidateLLMUpdateSpec(ctx, userCred, fakeLLM, input.LLMSpec)
	if err != nil {
		return nil, err
	}
	input.LLMSpec = spec
	if input.LLMSpec != nil && input.LLMSpec.SGLang == nil {
		input.LLMSpec.SGLang = &api.LLMSpecSGLang{}
	}
	return input, nil
}

func (s *sglang) ValidateLLMCreateSpec(ctx context.Context, userCred mcclient.TokenCredential, sku *models.SLLMSku, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil {
		return nil, nil
	}
	if input.SGLang == nil {
		input.SGLang = &api.LLMSpecSGLang{}
	}

	preferred := input.SGLang.PreferredModel
	if preferred == "" && sku != nil && sku.LLMSpec != nil && sku.LLMSpec.SGLang != nil {
		preferred = sku.LLMSpec.SGLang.PreferredModel
	}

	spec := &api.LLMSpecSGLang{}
	if sku != nil && sku.LLMSpec != nil && sku.LLMSpec.SGLang != nil {
		base := *sku.LLMSpec.SGLang
		spec = &base
	}
	if preferred != "" {
		spec.PreferredModel = preferred
	}
	mergedArgs, err := mergeSGLangCustomizedArgs(spec.CustomizedArgs, input.SGLang.CustomizedArgs)
	if err != nil {
		return nil, err
	}
	spec.CustomizedArgs = mergedArgs

	return &api.LLMSpec{SGLang: spec}, nil
}

func (s *sglang) ValidateLLMUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *api.LLMSpec) (*api.LLMSpec, error) {
	if input == nil || input.SGLang == nil {
		return input, nil
	}
	base := &api.LLMSpecSGLang{}
	if llm != nil && llm.LLMSpec != nil && llm.LLMSpec.SGLang != nil {
		b := *llm.LLMSpec.SGLang
		base = &b
	}

	if input.SGLang.PreferredModel != "" {
		base.PreferredModel = input.SGLang.PreferredModel
	}
	customizedArgs, err := normalizeSGLangCustomizedArgs(base.CustomizedArgs)
	if err != nil {
		return nil, err
	}
	base.CustomizedArgs = customizedArgs
	if input.SGLang.CustomizedArgs != nil {
		customizedArgs, err = normalizeSGLangCustomizedArgs(input.SGLang.CustomizedArgs)
		if err != nil {
			return nil, err
		}
		base.CustomizedArgs = customizedArgs
	}

	return &api.LLMSpec{SGLang: base}, nil
}

func (s *sglang) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	var postOverlays []*commonapi.ContainerVolumeMountDiskPostOverlay
	if llm != nil {
		var err error
		postOverlays, err = llm.GetMountedModelsPostOverlay()
		if err != nil {
			log.Errorf("GetMountedModelsPostOverlay failed %s", err)
		}
	}
	tensorParallelSize := 1
	if sku != nil && sku.Devices != nil && len(*sku.Devices) > 0 {
		tensorParallelSize = len(*sku.Devices)
	}
	effSpec := (*api.LLMSpecSGLang)(nil)
	if eff := s.GetEffectiveSpec(llm, sku); eff != nil {
		effSpec = eff.(*api.LLMSpecSGLang)
	}
	backendParameters := ""
	if sku != nil {
		backendParameters = sku.BackendParameters
	}
	startScript := buildSGLangEntrypointScript(len(postOverlays) > 0, tensorParallelSize, backendParameters, effSpec)
	envs := []*commonapi.ContainerKeyValue{
		{
			Key:   "HUGGING_FACE_HUB_CACHE",
			Value: api.LLM_SGLANG_CACHE_DIR,
		},
		{
			Key:   "HF_ENDPOINT",
			Value: api.LLM_SGLANG_HF_ENDPOINT,
		},
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

	effDevs := models.GetEffectiveDevices(llm, sku)
	if len(devices) == 0 && effDevs != nil && len(*effDevs) > 0 {
		for i := range *effDevs {
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

	diskIndex := 0
	ctrVols := []*commonapi.ContainerVolumeMount{
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				SubDirectory: api.LLM_SGLANG,
				Index:        &diskIndex,
				PostOverlay:  postOverlays,
			},
			Type:        commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath:   api.LLM_SGLANG_BASE_PATH,
			ReadOnly:    false,
			Propagation: commonapi.MOUNTPROPAGATION_PROPAGATION_HOST_TO_CONTAINER,
		},
		{
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

func (s *sglang) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	return []*computeapi.PodContainerCreateInput{
		s.GetContainerSpec(ctx, llm, image, sku, props, devices, diskId),
	}
}

func (s *sglang) GetLLMAccessUrlInfo(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, input *models.LLMAccessInfoInput) (*api.LLMAccessUrlInfo, error) {
	return models.GetLLMAccessUrlInfo(ctx, userCred, llm, input, "http", api.LLM_SGLANG_DEFAULT_PORT)
}

func (s *sglang) StartLLM(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) error {
	return nil
}

func (s *sglang) GetProbedInstantModelsExt(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, mdlIds ...string) (map[string]api.LLMInternalInstantMdlInfo, error) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "get llm container")
	}

	cmd := fmt.Sprintf("du -sk %s/*/", api.LLM_SGLANG_MODELS_PATH)
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
		fullPath := fields[1]
		name := path.Base(fullPath)
		if name == "" {
			continue
		}
		modelsMap[name] = api.LLMInternalInstantMdlInfo{
			Name:    name,
			ModelId: name,
			Tag:     "latest",
			Size:    sizeKB * 1024,
		}
	}
	return modelsMap, nil
}

func (s *sglang) DetectModelPaths(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, pkgInfo api.LLMInternalInstantMdlInfo) ([]string, error) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "get llm container")
	}

	modelPath := path.Join(api.LLM_SGLANG_MODELS_PATH, pkgInfo.Name)
	checkCmd := fmt.Sprintf("[ -d %s ] && echo 'EXIST' || echo 'MISSING'", shellQuoteSingle(modelPath))
	output, err := exec(ctx, lc.CmpId, checkCmd, 10)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check file existence")
	}
	if !strings.Contains(output, "EXIST") {
		return nil, errors.Errorf("model directory %s missing", modelPath)
	}
	return []string{modelPath}, nil
}

func (s *sglang) GetImageInternalPathMounts(sApp *models.SInstantModel) map[string]string {
	res := make(map[string]string)
	for _, mount := range sApp.Mounts {
		relPath := strings.TrimPrefix(mount, api.LLM_SGLANG_BASE_PATH)
		res[relPath] = path.Join(api.LLM_SGLANG, relPath)
	}
	return res
}

func (s *sglang) GetSaveDirectories(sApp *models.SInstantModel) (string, []string, error) {
	var filteredMounts []string
	for _, mount := range sApp.Mounts {
		if strings.HasPrefix(mount, api.LLM_SGLANG_BASE_PATH) {
			relPath := strings.TrimPrefix(mount, api.LLM_SGLANG_BASE_PATH)
			filteredMounts = append(filteredMounts, relPath)
		}
	}
	return "", filteredMounts, nil
}

func (s *sglang) ValidateMounts(mounts []string, mdlName string, mdlTag string) ([]string, error) {
	return mounts, nil
}

func (s *sglang) CheckDuplicateMounts(errStr string, dupIndex int) string {
	return "Duplicate mounts detected"
}

func (s *sglang) GetInstantModelIdByPostOverlay(postOverlay *commonapi.ContainerVolumeMountDiskPostOverlay, mdlNameToId map[string]string) string {
	return ""
}

func (s *sglang) GetDirPostOverlay(dir api.LLMMountDirInfo) *commonapi.ContainerVolumeMountDiskPostOverlay {
	uid := int64(1000)
	gid := int64(1000)
	ov := dir.ToOverlay()
	ov.FsUser = &uid
	ov.FsGroup = &gid
	return &ov
}

func (s *sglang) PreInstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return errors.Wrap(err, "get llm container")
	}
	cmd := fmt.Sprintf("mkdir -p %s", api.LLM_SGLANG_MODELS_PATH)
	_, err = exec(ctx, lc.CmpId, cmd, 10)
	return err
}

func (s *sglang) InstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, dirs []string, mdlIds []string) error {
	return nil
}

func (s *sglang) UninstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	return nil
}

func (s *sglang) DownloadModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, tmpDir string, modelName string, modelTag string, progress func(progress float32)) (string, []string, error) {
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

	rev := resolveHfdRevision(modelTag)
	apiURL := fmt.Sprintf("%s/api/models/%s?revision=%s", api.LLM_SGLANG_HF_ENDPOINT, escapeURLPathPreserveSlash(modelName), url.QueryEscape(rev))
	log.Infof("Downloading HF model via HF Mirror API for SGLang: %s", func() string {
		b, _ := json.Marshal(map[string]string{
			"model":    modelName,
			"revision": rev,
			"dir":      localDir,
			"endpoint": api.LLM_SGLANG_HF_ENDPOINT,
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
	totalSize := int64(0)
	completedSize := int64(0)
	for _, sibling := range meta.Siblings {
		if sibling.Size <= 0 {
			continue
		}
		rf := strings.TrimSpace(sibling.RFilename)
		if rf == "" {
			continue
		}
		totalSize += sibling.Size
		dst := filepath.Join(localDir, filepath.FromSlash(rf))
		if isCompleteFile(dst, sibling.Size) {
			completedSize += sibling.Size
		}
	}
	reportInstantModelDownloadProgress(progress, completedSize, totalSize)
	if isHuggingFaceImportComplete(localDir, meta.Siblings) {
		targetDir := path.Join(api.LLM_SGLANG_MODELS_PATH, modelBase)
		log.Infof("Model %s already exists in import dir %s", modelName, localDir)
		return modelName, []string{targetDir}, nil
	}

	for _, sibling := range meta.Siblings {
		rf := strings.TrimSpace(sibling.RFilename)
		if rf == "" {
			continue
		}
		dst := filepath.Join(localDir, filepath.FromSlash(rf))
		if isCompleteFile(dst, sibling.Size) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return "", nil, errors.Wrapf(err, "mkdir for %s", dst)
		}
		fileURL := fmt.Sprintf("%s/%s/resolve/%s/%s", api.LLM_SGLANG_HF_ENDPOINT, escapeURLPathPreserveSlash(modelName), url.PathEscape(rev), escapeURLPathPreserveSlash(rf))
		fileCompleted := completedSize
		if err := llm.HttpDownloadFileWithProgress(ctx, fileURL, dst, instantModelFileDownloadProgress(progress, fileCompleted, totalSize, sibling.Size)); err != nil {
			return "", nil, errors.Wrapf(err, "download file failed: %s -> %s", fileURL, dst)
		}
		if sibling.Size > 0 {
			completedSize += sibling.Size
			reportInstantModelDownloadProgress(progress, completedSize, totalSize)
		}
	}

	targetDir := path.Join(api.LLM_SGLANG_MODELS_PATH, modelBase)
	return modelName, []string{targetDir}, nil
}

var protectedSGLangArgKeys = map[string]struct{}{
	"model-path":        {},
	"served-model-name": {},
	"host":              {},
	"port":              {},
	"tp-size":           {},
}

func validateSGLangArgKey(key string) error {
	if key == "" {
		return errors.Error("sglang arg key is empty")
	}
	if strings.HasPrefix(key, "--") {
		return errors.Errorf("invalid sglang arg key %q: do not include leading --", key)
	}
	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return errors.Errorf("invalid sglang arg key %q", key)
	}
	if _, ok := protectedSGLangArgKeys[key]; ok {
		return errors.Errorf("sglang arg key %q is protected", key)
	}
	return nil
}

func normalizeSGLangCustomizedArgs(args []*api.SGLangCustomizedArg) ([]*api.SGLangCustomizedArg, error) {
	if len(args) == 0 {
		return nil, nil
	}
	out := make([]*api.SGLangCustomizedArg, 0, len(args))
	indexByKey := make(map[string]int, len(args))
	for _, arg := range args {
		if arg == nil {
			continue
		}
		key := strings.TrimSpace(arg.Key)
		if err := validateSGLangArgKey(key); err != nil {
			return nil, err
		}
		next := &api.SGLangCustomizedArg{
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

func mergeSGLangCustomizedArgs(base, overrides []*api.SGLangCustomizedArg) ([]*api.SGLangCustomizedArg, error) {
	out := make([]*api.SGLangCustomizedArg, 0, len(base)+len(overrides))
	indexByKey := make(map[string]int, len(base)+len(overrides))
	appendNormalized := func(items []*api.SGLangCustomizedArg) error {
		normalized, err := normalizeSGLangCustomizedArgs(items)
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

func appendSGLangCustomizedFlags(flags []string, effSpec *api.LLMSpecSGLang) []string {
	if effSpec == nil || len(effSpec.CustomizedArgs) == 0 {
		return flags
	}
	normalizedArgs, err := normalizeSGLangCustomizedArgs(effSpec.CustomizedArgs)
	if err != nil {
		log.Errorf("normalize sglang customized args: %v", err)
		return flags
	}
	for _, arg := range normalizedArgs {
		flagName := "--" + arg.Key
		if arg.Value == "" {
			flags = append(flags, flagName)
			continue
		}
		flags = append(flags, fmt.Sprintf("%s %s", flagName, shellQuoteSingle(arg.Value)))
	}
	return flags
}

func sglangCustomizedArgsToRuntime(args []*api.SGLangCustomizedArg) []runtimeArg {
	if len(args) == 0 {
		return nil
	}
	out := make([]runtimeArg, 0, len(args))
	for _, arg := range args {
		if arg == nil {
			continue
		}
		out = append(out, runtimeArg{Key: arg.Key, Value: arg.Value})
	}
	return out
}

func appendSGLangRuntimeFlags(flags []string, backendParameters string, effSpec *api.LLMSpecSGLang) []string {
	backendArgs, err := parseBackendParameterArgs(backendParameters, validateSGLangArgKey)
	if err != nil {
		log.Errorf("parse sglang backend parameters: %v", err)
	}
	var customizedArgs []runtimeArg
	if effSpec != nil {
		customizedArgs = sglangCustomizedArgsToRuntime(effSpec.CustomizedArgs)
	}
	mergedArgs, err := mergeRuntimeArgs(backendArgs, customizedArgs, validateSGLangArgKey)
	if err != nil {
		log.Errorf("merge sglang runtime args: %v", err)
		return flags
	}
	return appendRuntimeFlags(flags, mergedArgs)
}

func buildSGLangServeFlagsWithModelExpr(modelExpr string, servedModelNameExpr string, tensorParallelSize int, backendParameters string, effSpec *api.LLMSpecSGLang) []string {
	flags := []string{
		fmt.Sprintf("--model-path %s", modelExpr),
		fmt.Sprintf("--served-model-name %s", servedModelNameExpr),
		"--host 0.0.0.0",
		fmt.Sprintf("--port %d", api.LLM_SGLANG_DEFAULT_PORT),
		fmt.Sprintf("--tp-size %d", tensorParallelSize),
	}
	return appendSGLangRuntimeFlags(flags, backendParameters, effSpec)
}

func buildSGLangServeFlags(modelPath string, tensorParallelSize int, backendParameters string, effSpec *api.LLMSpecSGLang) []string {
	modelQuoted := shellQuoteSingle(modelPath)
	return buildSGLangServeFlagsWithModelExpr(
		modelQuoted,
		fmt.Sprintf(`"$(basename %s)"`, modelQuoted),
		tensorParallelSize,
		backendParameters,
		effSpec,
	)
}

func buildSGLangEntrypointScript(hasMountedModels bool, tensorParallelSize int, backendParameters string, effSpec *api.LLMSpecSGLang) string {
	modelsPath := shellQuoteSingle(api.LLM_SGLANG_MODELS_PATH)
	if !hasMountedModels {
		return fmt.Sprintf("mkdir -p %s && exec sleep infinity", modelsPath)
	}

	preferredPath := ""
	if effSpec != nil && strings.TrimSpace(effSpec.PreferredModel) != "" {
		preferredPath = path.Join(api.LLM_SGLANG_MODELS_PATH, strings.TrimSpace(effSpec.PreferredModel))
	}
	serveCmd := strings.Join(buildSGLangServeFlagsWithModelExpr(
		`"$model"`,
		`"$(basename "$model")"`,
		tensorParallelSize,
		backendParameters,
		effSpec,
	), " ")
	return strings.Join([]string{
		"set -e",
		fmt.Sprintf("mkdir -p %s", modelsPath),
		fmt.Sprintf("preferred=%s", shellQuoteSingle(preferredPath)),
		`if [ -n "$preferred" ] && [ -d "$preferred" ]; then`,
		`  model="$preferred"`,
		"else",
		fmt.Sprintf(`  model="$(find %s -mindepth 1 -maxdepth 1 -type d | sort | head -n 1)"`, modelsPath),
		"fi",
		`if [ -z "$model" ]; then`,
		`  echo "no mounted SGLang model found" >&2`,
		"  exit 1",
		"fi",
		fmt.Sprintf("exec %s %s", api.LLM_SGLANG_EXEC_PATH, serveCmd),
	}, "\n")
}
