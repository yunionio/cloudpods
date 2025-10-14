package llm

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/golang-plus/errors"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/drivers/pod"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func init() {
	models.RegisterLLMContainerDriver(newOllama())
	// log.Infoln("registed ollama")
}

type ollama struct{}

func newOllama() models.ILLMContainerDriver {
	return new(ollama)
}

func (o *ollama) GetType() llm.LLMContainerType {
	return llm.LLM_CONTAINER_OLLAMA
}

func (o *ollama) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMModel, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	spec := computeapi.ContainerSpec{
		ContainerSpec: apis.ContainerSpec{
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
				Type: apis.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
				IsolatedDevice: &computeapi.ContainerIsolatedDevice{
					Index: &index,
				},
			})
		}
	} else if len(devices) > 0 {
		for i := range devices {
			spec.Devices = append(spec.Devices, &computeapi.ContainerDevice{
				Type: apis.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
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

	// // process volume mounts
	// vols := make([]*apis.ContainerVolumeMount, 0)
	// appVolIndex := 2
	// postOverlays, err := d.GetMountedAppsPostOverlay()
	// if err != nil {
	// 	log.Errorf("GetMountedAppsPostOverlay failed %s", err)
	// }
	// vols = append(spec.VolumeMounts, GetDiskVolumeMounts(sku.Volumes, appVolIndex, postOverlays)...)

	// udevPath := filepath.Join(GetTmpSocketsHostPath(d.GetName()), "udev")
	// ctrVols := []*apis.ContainerVolumeMount{
	// 	{
	// 		UniqueName: "fake-udev",
	// 		Type:       apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
	// 		MountPath:  "/usr/bin/fake-udev",
	// 		HostPath: &apis.ContainerVolumeMountHostPath{
	// 			Type: apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
	// 			Path: filepath.Join(cdk.HOST_DIR_OPT_CDK, "etc/wolf/fake-udev"),
	// 		},
	// 		ReadOnly: true,
	// 	},
	// 	{
	// 		UniqueName: "steam-tmp-sockets",
	// 		Type:       apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
	// 		MountPath:  "/tmp/sockets",
	// 		HostPath: &apis.ContainerVolumeMountHostPath{
	// 			Type:       apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
	// 			Path:       GetTmpSocketsHostPath(d.GetName()),
	// 			AutoCreate: true,
	// 		},
	// 	},
	// 	{
	// 		UniqueName: "steam-run-udev",
	// 		Type:       apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
	// 		MountPath:  VOL_RUN_UDEV,
	// 		HostPath: &apis.ContainerVolumeMountHostPath{
	// 			AutoCreate: true,
	// 			Type:       apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
	// 			Path:       udevPath,
	// 		},
	// 	},
	// 	{
	// 		UniqueName: "steam-launcher-script",
	// 		Type:       apis.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
	// 		MountPath:  STEAM_LAUNCHER_SCRIPT_PATH,
	// 		HostPath: &apis.ContainerVolumeMountHostPath{
	// 			Type: apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
	// 			Path: filepath.Join(cdk.HOST_DIR_OPT_CDK, STEAM_LAUNCHER_SCRIPT_PATH),
	// 		},
	// 	},
	// }

	// set ollama
	var ollama bool = true
	spec.OllamaContainer = &ollama

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

var ollama_pod pod.SPodDriver

func (o *ollama) GetManifests(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, taskId string) error {
	modelName, modelTag, _ := llm.GetLargeLanguageModelName()
	suffix := fmt.Sprintf("%s/manifests/%s", modelName, modelTag)
	url := fmt.Sprintf(api.LLM_OLLAMA_LIBRARY_BASE_URL, suffix)
	ctr, _ := llm.GetLLMContainer()

	return download(ctx, userCred, ctr.CmpId, taskId, url, getManifestsPath(modelName, modelTag))
}

func (o *ollama) AccessBlobsCache(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, taskId string) error {
	ctr, _ := llm.GetLLMContainer()
	modelName, modelTag, _ := llm.GetLargeLanguageModelName()
	blobs, err := fetchBlobs(ctx, userCred, ctr.CmpId, modelName, modelTag)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch blobs for model %s:%s", modelName, modelTag)
	}

	input := &api.OllamaAccessCacheInput{
		Blobs:     blobs,
		ModelName: modelName,
	}
	_, err = ollama_pod.RequestOllamaBlobsCache(ctx, userCred, ctr.CmpId, taskId, input)

	return err
}

func (o *ollama) CopyBlobs(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) error {
	ctr, _ := llm.GetLLMContainer()
	modelName, modelTag, _ := llm.GetLargeLanguageModelName()
	blobs, _ := fetchBlobs(ctx, userCred, ctr.CmpId, modelName, modelTag)

	blobsTargetDir := path.Join(api.LLM_OLLAMA_BASE_PATH, api.LLM_OLLAMA_BLOBS_DIR)
	blobsSrcDir := path.Join(api.LLM_OLLAMA_CACHE_MOUNT_PATH, api.LLM_OLLAMA_CACHE_DIR)

	var commands []string
	commands = append(commands, fmt.Sprintf("mkdir -p %s", blobsTargetDir))
	for _, blob := range blobs {
		src := path.Join(blobsSrcDir, blob)
		target := path.Join(blobsTargetDir, blob)
		commands = append(commands, fmt.Sprintf("cp %s %s", src, target))
	}

	cmd := strings.Join(commands, " && ")
	if _, err := exec(ctx, userCred, ctr.CmpId, "/bin/sh", "-c", cmd); err != nil {
		return errors.Wrapf(err, "failed to copy blobs to container")
	}
	return nil
}

func download(ctx context.Context, userCred mcclient.TokenCredential, containerId string, taskId string, webUrl string, path string) error {
	input := &computeapi.ContainerDownloadFileInput{
		WebUrl: webUrl,
		Path:   path,
	}

	_, err := ollama_pod.RequestDownloadFileIntoContainer(ctx, userCred, containerId, taskId, input)
	return err
}

func exec(ctx context.Context, userCred mcclient.TokenCredential, containerId string, command ...string) (string, error) {
	// exec command

	input := &computeapi.ContainerExecSyncInput{
		Command: command,
	}
	resp, err := ollama_pod.RequestExecSyncContainer(ctx, userCred, containerId, input)

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

func fetchBlobs(ctx context.Context, userCred mcclient.TokenCredential, containerId string, modelName string, modelTag string) ([]string, error) {
	manifestContent, err := exec(ctx, userCred, containerId, "cat", getManifestsPath(modelName, modelTag))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read manifests from container")
	}

	// find all blobs
	var results []string
	re := regexp.MustCompile(`"digest":"(sha256:[^"]*)"`)
	matches := re.FindAllStringSubmatch(manifestContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			digest := match[1]
			processedDigest := strings.Replace(digest, "sha256:", "sha256-", 1)
			results = append(results, processedDigest)
		}
	}

	return results, nil
}
