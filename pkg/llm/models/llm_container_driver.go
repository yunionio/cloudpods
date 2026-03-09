package models

import (
	"context"
	"fmt"
	"sync"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type drivers struct {
	drivers *sync.Map
}

func newDrivers() *drivers {
	return &drivers{
		drivers: &sync.Map{},
	}
}

func (d *drivers) GetWithError(typ string) (interface{}, error) {
	drv, ok := d.drivers.Load(typ)
	if !ok {
		return drv, httperrors.NewNotFoundError("app container driver %s not found", typ)
	}
	return drv, nil
}

func (d *drivers) Get(typ string) interface{} {
	drv, err := d.GetWithError(typ)
	if err != nil {
		panic(err.Error())
	}
	return drv
}

func (d *drivers) Register(typ string, drv interface{}) {
	d.drivers.Store(typ, drv)
}

func registerDriver[K ~string, D any](drvs *drivers, typ K, drv D) {
	drvs.Register(string(typ), drv)
}

func getDriver[K ~string, D any](drvs *drivers, typ K) D {
	return drvs.Get(string(typ)).(D)
}

func getDriverWithError[K ~string, D any](drvs *drivers, typ K) (D, error) {
	drv, err := drvs.GetWithError(string(typ))
	if err != nil {
		return drv.(D), err
	}
	return drv.(D), nil
}

type ILLMContainerInstantModel interface {
	GetMountedModels(sku *SLLMSku) []string

	GetProbedInstantModelsExt(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, mdlIds ...string) (map[string]llm.LLMInternalInstantMdlInfo, error)
	DetectModelPaths(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, pkgInfo llm.LLMInternalInstantMdlInfo) ([]string, error)

	GetImageInternalPathMounts(sApp *SInstantModel) map[string]string
	GetSaveDirectories(sApp *SInstantModel) (string, []string, error)

	ValidateMounts(mounts []string, mdlName string, mdlTag string) ([]string, error)
	CheckDuplicateMounts(errStr string, dupIndex int) string
	GetInstantModelIdByPostOverlay(postOverlay *commonapi.ContainerVolumeMountDiskPostOverlay, mdlNameToId map[string]string) string
	GetDirPostOverlay(dir llm.LLMMountDirInfo) *commonapi.ContainerVolumeMountDiskPostOverlay

	PreInstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, instMdl *SLLMInstantModel) error
	InstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, dirs []string, mdlIds []string) error
	UninstallModel(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, instMdl *SLLMInstantModel) error
	DownloadModel(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, tmpDir string, modelName string, modelTag string) (string, []string, error)
}

// ILLMContainerDriverMultiContainer is an optional interface for drivers that create a pod with multiple containers (e.g. Dify). If not implemented, the driver is assumed to provide a single container via GetContainerSpec.
type ILLMContainerDriverMultiContainer interface {
	GetContainerSpecs(ctx context.Context, llm *SLLM, image *SLLMImage, sku *SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput
}

type ILLMContainerDriver interface {
	GetType() llm.LLMContainerType
	GetContainerSpec(ctx context.Context, llm *SLLM, image *SLLMImage, sku *SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput

	// StartLLM is called after the pod is running. For drivers that need to start the model process inside the container (e.g. vLLM), it runs the start command via exec and waits for health; on failure returns an error. For drivers that need no extra step (e.g. Ollama), it returns nil.
	StartLLM(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM) error

	// GetSpec returns the type-specific spec from the SKU (e.g. *LLMSpecOllama, *LLMSpecDify). Returns nil if not applicable or missing.
	GetSpec(sku *SLLMSku) interface{}
	// GetPrimaryImageId returns the primary image id for this SKU type (e.g. LLMImageId for ollama/vllm, DifyApiImageId for dify).
	GetPrimaryImageId(sku *SLLMSku) string
	// ValidateCreateSpec validates create input and returns the LLMSpec to store. Called by SKU manager after base validation.
	ValidateCreateSpec(ctx context.Context, userCred mcclient.TokenCredential, input *llm.LLMSkuCreateInput) (*llm.LLMSpec, error)
	// ValidateUpdateSpec validates update input, merges with current spec, and returns the LLMSpec to store. Called by SKU when LLMSpec is not nil.
	ValidateUpdateSpec(ctx context.Context, userCred mcclient.TokenCredential, sku *SLLMSku, input *llm.LLMSkuUpdateInput) (*llm.LLMSpec, error)

	ILLMContainerMCPAgent
}

// ILLMContainerInstantModelDriver is for drivers that support instant models (e.g. Ollama, vLLM). GetMountedModels is only required here so that drivers without models (e.g. Dify) need not implement it.
type ILLMContainerInstantModelDriver interface {
	ILLMContainerDriver
	ILLMContainerInstantModel
}

type ILLMContainerMCPAgent interface {
	GetLLMUrl(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM) (string, error)
}

var (
	llmContainerDrivers = newDrivers()
)

func RegisterLLMContainerDriver(drv ILLMContainerDriver) {
	registerDriver(llmContainerDrivers, drv.GetType(), drv)
}

func GetLLMContainerDriver(typ llm.LLMContainerType) ILLMContainerDriver {
	return getDriver[llm.LLMContainerType, ILLMContainerDriver](llmContainerDrivers, typ)
}

func GetLLMContainerDriverWithError(typ llm.LLMContainerType) (ILLMContainerDriver, error) {
	return getDriverWithError[llm.LLMContainerType, ILLMContainerDriver](llmContainerDrivers, typ)
}

func GetLLMContainerInstantModelDriver(typ llm.LLMContainerType) (ILLMContainerInstantModelDriver, error) {
	drv, err := GetLLMContainerDriverWithError(typ)
	if err != nil {
		return nil, err
	}
	if instantDrv, ok := drv.(ILLMContainerInstantModelDriver); ok {
		return instantDrv, nil
	}
	return nil, fmt.Errorf("driver %s does not support instant model operations", typ)
}

// GetDriverPodContainers returns the container(s) for the given driver. If the driver implements ILLMContainerDriverMultiContainer, GetContainerSpecs is used; otherwise a single-element slice from GetContainerSpec is returned.
func GetDriverPodContainers(ctx context.Context, drv ILLMContainerDriver, llm *SLLM, image *SLLMImage, sku *SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	if multi, ok := drv.(ILLMContainerDriverMultiContainer); ok {
		return multi.GetContainerSpecs(ctx, llm, image, sku, props, devices, diskId)
	}
	spec := drv.GetContainerSpec(ctx, llm, image, sku, props, devices, diskId)
	if spec == nil {
		return nil
	}
	return []*computeapi.PodContainerCreateInput{spec}
}
