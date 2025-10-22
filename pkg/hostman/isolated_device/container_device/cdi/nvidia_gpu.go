package cdi

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
)

func init() {
	isolated_device.RegisterContainerCDIManaer(apis.CONTAINER_CDI_KIND_NVIDIA_GPU, func() (isolated_device.IContainerCDIManager, error) {
		return newNvidiaGPU(), nil
	})
}

type nvidiaGPU struct{}

func newNvidiaGPU() isolated_device.IContainerCDIManager {
	return &nvidiaGPU{}
}

func (n *nvidiaGPU) GetKind() apis.ContainerCDIKind {
	return apis.CONTAINER_CDI_KIND_NVIDIA_GPU
}

func (n *nvidiaGPU) GetSpecFilePath() string {
	return "/etc/cdi/nvidia.yaml"
}

// TODO: 检查 cdi 是否存在 /etc/cdi/nvidia.yaml
func (n *nvidiaGPU) GetDeviceName(dev *hostapi.ContainerDevice) (string, error) {
	if dev.IsolatedDevice == nil {
		return "", errors.Errorf("isolated_device is nil: %s", jsonutils.Marshal(dev))
	}
	gpuId := dev.IsolatedDevice.Path
	if gpuId == "" {
		return "", errors.Wrapf(errors.ErrNotEmpty, "gpu_id from %s", jsonutils.Marshal(dev))
	}
	return gpuId, nil
}
