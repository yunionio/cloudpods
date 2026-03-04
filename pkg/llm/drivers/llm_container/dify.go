package llm_container

import (
	"context"
	"fmt"
	"strconv"

	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	models.RegisterLLMContainerDriver(newDify())
}

type dify struct{}

func newDify() models.ILLMContainerDriver {
	return new(dify)
}

func (d *dify) GetType() api.LLMContainerType {
	return api.LLM_CONTAINER_DIFY
}

// GetContainerSpec is required by ILLMContainerDriver but not used for Dify; pod creation uses GetContainerSpecs. Return the first container so the interface is satisfied.
func (d *dify) GetContainerSpec(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) *computeapi.PodContainerCreateInput {
	specs := d.GetContainerSpecs(ctx, llm, image, sku, props, devices, diskId)
	if len(specs) == 0 {
		return nil
	}
	return specs[0]
}

// GetContainerSpecs returns all Dify pod containers (postgres, redis, api, worker, nginx, etc.). Uses llm.GetDifyCustomizedEnvs() when set so UserCustomizedEnvs take effect without DifyContainersManager.
func (d *dify) GetContainerSpecs(ctx context.Context, llm *models.SLLM, image *models.SLLMImage, sku *models.SLLMSku, props []string, devices []computeapi.SIsolatedDevice, diskId string) []*computeapi.PodContainerCreateInput {
	return models.GetDifyContainersByNameAndSku(llm.GetName(), sku, llm.GetDifyCustomizedEnvs())
}

// StartLLM is a no-op for Dify; all services are started by their container entrypoints.
func (d *dify) StartLLM(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) error {
	return nil
}

// GetLLMUrl returns the Dify access URL (nginx port 80). Same pattern as vLLM/Ollama: guest network uses LLMIp, hostlocal uses host IP.
func (d *dify) GetLLMUrl(ctx context.Context, userCred mcclient.TokenCredential, llm *models.SLLM) (string, error) {
	server, err := llm.GetServer(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get server")
	}
	port := 80
	if p, err := strconv.Atoi(api.DIFY_NGINX_PORT); err == nil {
		port = p
	}
	networkType := llm.NetworkType
	if networkType == string(computeapi.NETWORK_TYPE_GUEST) {
		if len(llm.LLMIp) == 0 {
			return "", errors.Error("LLM IP is empty for guest network")
		}
		return fmt.Sprintf("http://%s:%d", llm.LLMIp, port), nil
	}
	if len(server.HostAccessIp) == 0 {
		return "", errors.Error("host access IP is empty")
	}
	return fmt.Sprintf("http://%s:%d", server.HostAccessIp, port), nil
}
