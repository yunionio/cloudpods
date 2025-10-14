package llm

import (
	"context"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
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

func (o *ollama) PullModelByInstall(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, modelName string, modelTag string) error {
	return nil
}

func (o *ollama) PullModelByGgufFile(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM, ggufFileUrl string, model string) error {
	return nil
}
