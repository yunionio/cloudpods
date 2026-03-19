package llm_container

import (
	"context"

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
				SubDirectory: "storage",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: "/root",
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "storage-models/models",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: "/root/ComfyUI/models",
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
			MountPath: "/root/ComfyUI/input",
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "storage-user/output",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: "/root/ComfyUI/output",
		},
		{
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: "storage-user/workflows",
			},
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: "/root/ComfyUI/user/default/workflows",
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
	return nil, nil
}

func (c *comfyui) DetectModelPaths(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, pkgInfo api.LLMInternalInstantMdlInfo) ([]string, error) {
	return nil, nil
}

func (c *comfyui) GetImageInternalPathMounts(sApp *models.SInstantModel) map[string]string {
	return nil
}

func (c *comfyui) GetSaveDirectories(sApp *models.SInstantModel) (string, []string, error) {
	return "", nil, nil
}

func (c *comfyui) ValidateMounts(mounts []string, mdlName string, mdlTag string) ([]string, error) {
	return nil, nil
}

func (c *comfyui) CheckDuplicateMounts(errStr string, dupIndex int) string {
	return "Duplicate mounts detected"
}

func (c *comfyui) GetInstantModelIdByPostOverlay(postOverlay *commonapi.ContainerVolumeMountDiskPostOverlay, mdlNameToId map[string]string) string {
	return ""
}

func (c *comfyui) GetDirPostOverlay(dir api.LLMMountDirInfo) *commonapi.ContainerVolumeMountDiskPostOverlay {
	return nil
}

func (c *comfyui) PreInstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	return nil
}

func (c *comfyui) InstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, dirs []string, mdlIds []string) error {
	return nil
}

func (c *comfyui) UninstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, instMdl *models.SLLMInstantModel) error {
	return nil
}

func (c *comfyui) DownloadModel(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, tmpDir string, modelName string, modelTag string) (string, []string, error) {
	return "", nil, nil
}
