package llm_container

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
	llmutil "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterLLMContainerDriver(newOllama())
	// log.Infoln("registed ollama")
}

type ollama struct{}

func newOllama() models.ILLMContainerDriver {
	return new(ollama)
}

func (o *ollama) GetType() api.LLMContainerType {
	return api.LLM_CONTAINER_OLLAMA
}

func (o *ollama) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	spec := computeapi.ContainerSpec{
		ContainerSpec: commonapi.ContainerSpec{
			Image:             image.ToContainerImage(),
			ImageCredentialId: image.CredentialId,
			EnableLxcfs:       true,
			AlwaysRestart:     true,
		},
	}

	if len(devices) == 0 && len(*sku.Devices) > 0 {
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

	// // process rootfs limit
	// var diskIndex *int
	// if len(diskId) == 0 {
	// 	diskIndex0 := 0
	// 	diskIndex = &diskIndex0
	// }
	// if options.Options.EnableRootfsLimit {
	// 	if !d.RootfsUnlimit {
	// 		spec.RootFs = &apis.ContainerRootfs{
	// 			Type: apis.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
	// 			Disk: &apis.ContainerVolumeMountDisk{
	// 				Index:        diskIndex,
	// 				Id:           diskId,
	// 				SubDirectory: "/rootfs-steam",
	// 			},
	// 			Persistent: false,
	// 		}
	// 	}
	// }

	// process volume mounts
	postOverlays, err := llm.GetMountedModelsPostOverlay()
	if err != nil {
		log.Errorf("GetMountedModelsPostOverlay failed %s", err)
	}
	vols := spec.VolumeMounts

	// udevPath := filepath.Join(GetTmpSocketsHostPath(d.GetName()), "udev")
	diskIndex := 0
	ctrVols := []*commonapi.ContainerVolumeMount{
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				SubDirectory: api.LLM_OLLAMA,
				Overlay: &commonapi.ContainerVolumeMountDiskOverlay{
					LowerDir: []string{api.LLM_OLLAMA_HOST_PATH},
				},
				PostOverlay: postOverlays,
				Index:       &diskIndex,
			},
			Type:        commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath:   api.LLM_OLLAMA_BASE_PATH,
			ReadOnly:    false,
			Propagation: commonapi.MOUNTPROPAGATION_PROPAGATION_HOST_TO_CONTAINER,
		},
	}
	vols = append(vols, ctrVols...)

	spec.VolumeMounts = vols

	return &computeapi.PodContainerCreateInput{
		ContainerSpec: spec,
	}
}

// func (o *ollama) PullModelByInstall(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, modelName string, modelTag string) error {
// 	return nil
// }

// func (o *ollama) PullModelByGgufFile(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, ggufFileUrl string, model string) error {
// 	return nil
// }

// func (o *ollama) GetManifests(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, taskId string) error {
// 	modelName, modelTag, _ := llm.GetLargeLanguageModelName()
// 	suffix := fmt.Sprintf("%s/manifests/%s", modelName, modelTag)
// 	url := fmt.Sprintf(api.LLM_OLLAMA_LIBRARY_BASE_URL, suffix)
// 	ctr, _ := llm.GetLLMContainer()

// 	return download(ctx, userCred, ctr.CmpId, taskId, url, getManifestsPath(modelName, modelTag))
// }

// func (o *ollama) AccessBlobsCache(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, taskId string) error {
// 	ctr, _ := llm.GetLLMContainer()
// 	modelName, modelTag, _ := llm.GetLargeLanguageModelName()
// 	blobs, err := fetchBlobs(ctx, userCred, ctr.CmpId, modelName, modelTag)
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to fetch blobs for model %s:%s", modelName, modelTag)
// 	}

// 	input := &api.OllamaAccessCacheInput{
// 		Blobs:     blobs,
// 		ModelName: modelName,
// 	}
// 	_, err = ollama_pod.RequestOllamaBlobsCache(ctx, userCred, ctr.CmpId, taskId, input)

// 	return err
// }

func (o *ollama) DownloadModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, tmpDir string, modelName string, modelTag string) (string, []string, error) {
	// 1. download manifest from registry
	manifestsUrl := fmt.Sprintf(api.LLM_OLLAMA_LIBRARY_BASE_URL, fmt.Sprintf("%s/manifests/%s", modelName, modelTag))
	log.Infof("Downloading manifest from %s", manifestsUrl)

	manifestContent, err := llm.HttpGet(ctx, manifestsUrl)
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to download manifest from %s", manifestsUrl)
	}

	// 2. calculate model_id (sha256 of manifest content, take first 12 chars)
	hash := sha256.Sum256(manifestContent)
	modelId := hex.EncodeToString(hash[:])[:12]
	log.Infof("Model %s:%s has model_id: %s", modelName, modelTag, modelId)

	// 3. parse manifest to get blobs
	manifest := &Manifest{}
	if err := json.Unmarshal(manifestContent, manifest); err != nil {
		return "", nil, errors.Wrapf(err, "failed to parse manifest")
	}

	// collect all blob digests (config + layers)
	var blobs []string
	blobs = append(blobs, manifest.Config.Digest)
	for _, layer := range manifest.Layers {
		blobs = append(blobs, layer.Digest)
	}

	// 4. create directory structure
	// tmpDir/blobs/
	// tmpDir/manifests/registry.ollama.ai/library/<modelName>/<modelTag>
	blobsDir := path.Join(tmpDir, "blobs")
	manifestsDir := path.Join(tmpDir, api.LLM_OLLAMA_MANIFESTS_BASE_PATH, modelName)
	if err := os.MkdirAll(blobsDir, 0755); err != nil {
		return "", nil, errors.Wrapf(err, "failed to create blobs directory %s", blobsDir)
	}
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return "", nil, errors.Wrapf(err, "failed to create manifests directory %s", manifestsDir)
	}

	// 5. download each blob
	for _, digest := range blobs {
		// convert sha256:xxx to sha256-xxx for filename
		blobFileName := strings.Replace(digest, ":", "-", 1)
		blobPath := path.Join(blobsDir, blobFileName)

		// check if blob already exists
		if _, err := os.Stat(blobPath); err == nil {
			log.Infof("Blob %s already exists, skipping", blobFileName)
			continue
		}

		// download blob from registry
		// URL format: https://registry.ollama.ai/v2/library/<modelName>/blobs/<digest>
		blobUrl := fmt.Sprintf(api.LLM_OLLAMA_LIBRARY_BASE_URL, fmt.Sprintf("%s/blobs/%s", modelName, digest))
		log.Infof("Downloading blob %s from %s", blobFileName, blobUrl)

		if err := llm.HttpDownloadFile(ctx, blobUrl, blobPath); err != nil {
			return "", nil, errors.Wrapf(err, "failed to download blob %s", digest)
		}
	}

	// 6. save manifest file
	manifestPath := path.Join(manifestsDir, modelTag)
	if err := os.WriteFile(manifestPath, manifestContent, 0644); err != nil {
		return "", nil, errors.Wrapf(err, "failed to save manifest to %s", manifestPath)
	}
	log.Infof("Model %s:%s downloaded successfully to %s", modelName, modelTag, tmpDir)

	// 7. collect mount paths
	mounts := make([]string, len(blobs))
	for i, blob := range blobs {
		blobFileName := strings.Replace(blob, ":", "-", 1)
		mounts[i] = path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_BLOBS_DIR, blobFileName)
	}
	mounts = append(mounts, path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_MANIFESTS_BASE_PATH, modelName, modelTag))

	return modelId, mounts, nil
}

func (o *ollama) PreInstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	// before mount, make sure the manifests dir is ready
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return errors.Wrap(err, "get llm container")
	}

	// mkdir llm-registry-base-path / modelname
	mkdirReigtryBasePaht := fmt.Sprintf("mkdir -p %s", path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_MANIFESTS_BASE_PATH, instMdl.ModelName))
	_, err = exec(ctx, lc.CmpId, mkdirReigtryBasePaht, 10)
	if err != nil {
		return errors.Wrap(err, "failed to mkdir llm-registry-base-path / modelname")
	}

	return nil
}

func (o *ollama) InstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, dirs []string, mdlIds []string) error {
	///* TODO
	return nil
}

func (o *ollama) GetModelMountPaths(ctx context.Context, userCred mcclient.TokenCredential, llmInstMdl *models.SLLMInstantModel) ([]string, error) {
	instMdl, _ := llmInstMdl.FindInstantModel(false)
	return instMdl.Mounts, nil
}

func (o *ollama) GetDirPostOverlay(dir api.LLMMountDirInfo) *commonapi.ContainerVolumeMountDiskPostOverlay {
	uid := int64(1000)
	gid := int64(1000)
	ov := dir.ToOverlay()
	ov.FsUser = &uid
	ov.FsGroup = &gid
	return &ov
}

func (o *ollama) UninstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, llmInstMdl *models.SLLMInstantModel) error {
	// don't rm file

	// mounts, err := o.GetModelMountPaths(ctx, userCred, llmInstMdl)
	// if err != nil {
	// 	return errors.Wrap(err, "GetModelMountPaths")
	// }
	// ctr, err := llm.GetLLMSContainer(ctx)
	// if err != nil {
	// 	return errors.Wrap(err, "GetSContainer")
	// }
	// _, err = exec(ctx, ctr.Id, fmt.Sprintf("rm -rf %s", strings.Join(mounts, " ")), 10)
	// if err != nil {
	// 	return errors.Wrapf(err, "run cmd to remove model mounts of model %s", jsonutils.Marshal(llmInstMdl))
	// }
	return nil
}

func (o *ollama) GetInstantModelIdByPostOverlay(postOverlay *commonapi.ContainerVolumeMountDiskPostOverlay, mdlNameToId map[string]string) string {
	if postOverlay.Image != nil {
		for k := range postOverlay.Image.PathMap {
			idx := strings.Index(k, api.LLM_OLLAMA_MANIFESTS_BASE_PATH)
			if idx != -1 {
				suffix := k[idx+len(api.LLM_OLLAMA_MANIFESTS_BASE_PATH):]
				parts := strings.Split(strings.Trim(suffix, "/"), "/")
				if len(parts) >= 2 {
					modelName := parts[len(parts)-2]
					modelTag := parts[len(parts)-1]
					log.Infof("In GetInstantModelIdByPostOverlay, Extracted modelName: %s, modelTag: %s, Got modelId: %s", modelName, modelTag, mdlNameToId[modelName+":"+modelTag])
					return mdlNameToId[modelName+":"+modelTag]
				}
			}
		}
	}
	return ""
}

func (o *ollama) DetectModelPaths(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, pkgInfo api.LLMInternalInstantMdlInfo) ([]string, error) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "get llm container")
	}
	// check file exists
	originBlobs := make([]string, len(pkgInfo.Blobs))
	for idx, blob := range pkgInfo.Blobs {
		originBlobs[idx] = path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_BLOBS_DIR, blob)
	}
	originManifests := path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_MANIFESTS_BASE_PATH, pkgInfo.Name, pkgInfo.Tag)

	var checks []string
	for _, blob := range originBlobs {
		checks = append(checks, fmt.Sprintf("[ -f '%s' ]", blob))
	}
	checks = append(checks, fmt.Sprintf("[ -f '%s' ]", originManifests))

	checkCmd := strings.Join(checks, " && ") + " && echo 'ALL_EXIST' || echo 'SOME_MISSING'"
	output, err := exec(ctx, lc.CmpId, checkCmd, 10)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check file existence")
	}

	if !strings.Contains(output, "ALL_EXIST") {
		log.Infof("Some files are missing for model %s:%s, blobs: %v, manifest: %s, checkCmd: %s",
			pkgInfo.Name, pkgInfo.Tag, originBlobs, originManifests, checkCmd)
		return nil, errors.Errorf("required model files are missing")
	}

	// // mkdir
	// savePath := fmt.Sprintf(api.LLM_OLLAMA_SAVE_DIR, pkgInfo.Name+"-"+pkgInfo.Tag+"-"+pkgInfo.ModelId)
	// mkSaveDir := fmt.Sprintf("mkdir -p %s %s %s", savePath, path.Join(savePath, api.LLM_OLLAMA_BLOBS_DIR), path.Join(savePath, api.LLM_OLLAMA_HOST_MANIFESTS_DIR))
	// _, err = exec(ctx, ctr.CmpId, mkSaveDir, 10)
	// if err != nil {
	// 	return a, filtenil, errors.Wrap(err, "mkdir savedir")
	// }

	// // cp file
	// for _, blob := range originBlobs {
	// 	cpBlob := fmt.Sprintf("cp %s %s", blob, path.Join(savePath, api.LLM_OLLAMA_BLOBS_DIR))
	// 	_, err = exec(ctx, ctr.CmpId, cpBlob, 60)
	// 	if err != nil {
	// 		return nil, errors.Wrap(err, "copy files")
	// 	}
	// }
	// cpManifest := "cp " + originManifests + " " + path.Join(savePath, api.LLM_OLLAMA_HOST_MANIFESTS_DIR)
	// _, err = exec(ctx, ctr.CmpId, cpManifest, 20)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "copy files")
	// }

	return append(originBlobs, originManifests), nil
}

func (o *ollama) GetImageInternalPathMounts(sMdl *models.SInstantModel) map[string]string {
	imageToContainer := make(map[string]string)

	for _, mount := range sMdl.Mounts {
		imgPath := strings.TrimPrefix(mount, api.LLM_OLLAMA_BASE_PATH)
		imageToContainer[imgPath] = path.Join(api.LLM_OLLAMA, imgPath)
	}

	return imageToContainer
}

func (o *ollama) GetSaveDirectories(sApp *models.SInstantModel) (string, []string, error) {
	var filteredMounts []string

	for _, mount := range sApp.Mounts {
		if strings.HasPrefix(mount, api.LLM_OLLAMA_BASE_PATH) {
			relPath := strings.TrimPrefix(mount, api.LLM_OLLAMA_BASE_PATH)
			filteredMounts = append(filteredMounts, relPath)
		}
	}

	return "", filteredMounts, nil
}

func (o *ollama) GetProbedInstantModelsExt(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, mdlIds ...string) (map[string]api.LLMInternalInstantMdlInfo, error) {
	lc, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "get llm container")
	}

	// get all effective models
	getModels := "ollama list" // NAME ID SIZE MODIFIED
	modelsOutput, err := exec(ctx, lc.CmpId, getModels, 10)
	if err != nil {
		return nil, errors.Wrap(err, "get models")
	}
	lines := strings.Split(strings.TrimSpace(modelsOutput), "\n")

	models := make(map[string]api.LLMInternalInstantMdlInfo, len(lines)-1)
	for i := 1; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) > 2 {
			if len(mdlIds) > 0 && !utils.IsInStringArray(fields[1], mdlIds) {
				continue
			}
			modelName, modelTag, _ := llm.GetLargeLanguageModelName(fields[0])
			models[fields[1]] = api.LLMInternalInstantMdlInfo{
				Name:    modelName,
				Tag:     modelTag,
				ModelId: fields[1],
				// Modified: fields[3],
			}
		}
	}

	// for each model, get manifests file, find blobs, calculate size
	for modelId, model := range models {
		manifests, err := getManifests(ctx, lc.CmpId, model.Name, model.Tag)
		if err != nil {
			return nil, errors.Wrap(err, "get manifests")
		}
		model.Size = manifests.Config.Size
		model.Blobs = append(model.Blobs, manifests.Config.Digest)
		for _, layer := range manifests.Layers {
			model.Size += layer.Size
			model.Blobs = append(model.Blobs, layer.Digest)
		}
		models[modelId] = model
	}

	return models, nil
}

func (o *ollama) ValidateMounts(mounts []string, mdlName string, mdlTag string) ([]string, error) {
	return mounts, nil
}

func (o *ollama) CheckDuplicateMounts(errStr string, dupIndex int) string {
	// Find the first model path before "duplicated container target dirs"
	firstPath := extractModelPath(errStr[:dupIndex], api.LLM_OLLAMA_MANIFESTS_BASE_PATH, true)
	firstModel := parseModelName(firstPath)

	// Find the second model path after "duplicated container target dirs"
	secondPath := extractModelPath(errStr[dupIndex:], api.LLM_OLLAMA_MANIFESTS_BASE_PATH, false)
	secondModel := parseModelName(secondPath)

	return fmt.Sprintf("Model %s and %s have duplicated container target dirs", firstModel, secondModel)
}

// func download(ctx context.Context, userCred mcclient.TokenCredential, containerId string, taskId string, webUrl string, path string) error {
// 	input := &computeapi.ContainerDownloadFileInput{
// 		WebUrl: webUrl,
// 		Path:   path,
// 	}

// 	_, err := ollama_pod.RequestDownloadFileIntoContainer(ctx, userCred, containerId, taskId, input)
// 	return err
// }

// func (o *ollama) CopyBlobs(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) error {
// 	ctr, _ := llm.GetLLMContainer()
// 	modelName, modelTag, _ := llm.GetLargeLanguageModelName("")
// 	manifests, _ := getManifests(ctx, ctr.CmpId, modelName, modelTag)

// 	blobsTargetDir := path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_BLOBS_DIR)
// 	blobsSrcDir := path.Join(api.LLM_OLLAMA_CACHE_MOUNT_PATH, api.LLM_OLLAMA_CACHE_DIR)

// 	var commands []string
// 	commands = append(commands, fmt.Sprintf("mkdir -p %s", blobsTargetDir))
// 	blob := manifests.Config.Digest
// 	commands = append(commands, fmt.Sprintf("cp %s %s", path.Join(blobsSrcDir, blob), path.Join(blobsTargetDir, blob)))
// 	for _, layer := range manifests.Layers {
// 		blob = layer.Digest
// 		src := path.Join(blobsSrcDir, blob)
// 		target := path.Join(blobsTargetDir, blob)
// 		commands = append(commands, fmt.Sprintf("cp %s %s", src, target))
// 	}

// 	cmd := strings.Join(commands, " && ")
// 	if _, err := exec(ctx, ctr.CmpId, cmd, 120); err != nil {
// 		return errors.Wrapf(err, "failed to copy blobs to container")
// 	}
// 	return nil
// }

func exec(ctx context.Context, containerId string, cmd string, timeoutSec int64) (string, error) {
	// exec command
	input := &computeapi.ContainerExecSyncInput{
		Command: []string{"sh", "-c", cmd},
		Timeout: timeoutSec,
	}
	resp, err := llmutil.ExecSyncContainer(ctx, containerId, input)

	// check error and return result
	var rst string
	if nil != err || resp == nil {
		return "", errors.Wrapf(err, "LLM exec error")
	}
	rst, _ = resp.GetString("stdout")
	log.Infoln("llm container exec result: ", resp)
	return rst, nil
}

func getManifestsPath(modelName, modelTag string) string {
	return path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_MANIFESTS_BASE_PATH, modelName, modelTag)
}

type Layer struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

type Manifest struct {
	SchemaVersion int     `json:"schemaVersion"`
	MediaType     string  `json:"mediaType"`
	Config        Layer   `json:"config"`
	Layers        []Layer `json:"layers"`
}

func getManifests(ctx context.Context, containerId string, modelName string, modelTag string) (*Manifest, error) {
	manifestContent, err := exec(ctx, containerId, "cat "+getManifestsPath(modelName, modelTag), 10)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read manifests from container")
	}

	// find all blobs
	manifests := &Manifest{}
	if err = json.Unmarshal([]byte(manifestContent), manifests); err != nil {
		return nil, errors.Wrapf(err, "failed to parse manifests")
	}
	// log.Infof("manifests: %v", manifests)

	manifests.Config.Digest = strings.Replace(manifests.Config.Digest, ":", "-", 1)
	for idx, layer := range manifests.Layers {
		manifests.Layers[idx].Digest = strings.Replace(layer.Digest, ":", "-", 1)
	}

	return manifests, nil
}

func extractModelPath(str, startMarker string, findLast bool) string {
	var idx int
	if findLast {
		idx = strings.LastIndex(str, startMarker)
	} else {
		idx = strings.Index(str, startMarker)
	}

	if idx == -1 {
		return ""
	}

	pathStart := idx
	pathEnd := -1
	for i := pathStart; i < len(str); i++ {
		if str[i] == '\\' && i+4 < len(str) && str[i+1] == '\\' && str[i+2] == '\\' && str[i+3] == '\\' && str[i+4] == '"' { // for \\\\\"
			pathEnd = i
			break
		}
		if str[i] == '"' || str[i] == ',' || str[i] == ':' || str[i] == '}' {
			pathEnd = i
			break
		}
	}
	var extracted string
	if pathEnd != -1 {
		extracted = str[pathStart:pathEnd]
	} else {
		extracted = str[pathStart:]
	}

	return extracted
}

func parseModelName(path string) string {
	if !strings.HasPrefix(path, api.LLM_OLLAMA_MANIFESTS_BASE_PATH) {
		return ""
	}
	model := strings.TrimPrefix(path, api.LLM_OLLAMA_MANIFESTS_BASE_PATH)
	model = strings.TrimPrefix(model, "/")
	lastSlash := strings.LastIndex(model, "/")
	if lastSlash != -1 {
		name := model[:lastSlash]
		tag := model[lastSlash+1:]
		tag = strings.TrimRight(tag, `\`)
		return name + ":" + tag
	}
	return strings.TrimRight(model, `\`)
}
