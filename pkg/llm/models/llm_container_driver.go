package models

import (
	"context"
	"sync"

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

type ILLMContainerPullModel interface {
	// PullModelByInstall(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, modelName string, modelTag string) error
	// PullModelByGgufFile(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, ggufFileUrl string, model string) error
	// DownloadGgufFile(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, ggufFileUrl string, ggufFilePath string) error
	// InstallGgufModel(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, ggufFilePath string) error
	GetManifests(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, taskId string) error
	AccessBlobsCache(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, taskId string) error
	CopyBlobs(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM) error
}

type ILLMContainerInstantApp interface {
	GetProbedPackagesExt(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, pkgNames ...string) (map[string]llm.LLMInternalPkgInfo, error)
	// DetectAppPaths(ctx context.Context, userCred mcclient.TokenCredential, llm *SLLM, pkgInfo llm.LLMInternalPkgInfo) ([]string, error)

	// GetImageInternalPathMounts(iApp *SInstantApp) map[string]string
	// ConvertMountToSavePath(mount string) (string, error)
	// GetSaveVolumeMountToImageInputDirPrefix(iApp *SInstantApp) string

	// ValidateMounts(mounts []string, pkg string) ([]string, error)
	// GetPackageAppIdByPostOverlay(postOverlay *commonapi.ContainerVolumeMountDiskPostOverlay) string
	// GetDirPostOverlay(dir cdk.DesktopMountDirInfo) *commonapi.ContainerVolumeMountDiskPostOverlay

	// PreInstallApp(ctx context.Context, userCred mcclient.TokenCredential, d *SDesktop, app *SDesktopApp, iconBase64 string) error
	// InstallApp(ctx context.Context, userCred mcclient.TokenCredential, d *SDesktop, dirs []string, appIds []string) error
	// UninstallApp(ctx context.Context, userCred mcclient.TokenCredential, d *SDesktop, app *SDesktopApp) error
	// CleanAppTempIcon(ctx context.Context, userCred mcclient.TokenCredential, d *SDesktop, pkgName string) error
}

type ILLMContainerDriver interface {
	GetType() llm.LLMContainerType
	GetContainerSpec(ctx context.Context, llm *SLLM, image *SLLMImage, sku *SLLMModel, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput

	// ILLMContainerPullModel

	ILLMContainerInstantApp
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
