package regiondrivers

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/utils"
)

type SAliyunRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SAliyunRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SAliyunRegionDriver) GetProvider() string {
	return models.CLOUD_PROVIDER_ALIYUN
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	loadbalancerSpec, _ := data.GetString("loadbalancer_spec")
	if len(loadbalancerSpec) != 0 && !utils.IsInStringArray(loadbalancerSpec, []string{"slb.s1.small", "slb.s2.small", "slb.s2.mediu", "slb.s3.small", "slb.s3.mediu", "slb.s3.large"}) {
		return nil, httperrors.NewInputParameterError("Unsupport loadbalancer_spec %s, support slb.s1.small、slb.s2.small、slb.s2.medium、slb.s3.small、slb.s3.medium、slb.s3.large", loadbalancerSpec)
	}
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateUpdateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("certificate") || data.Contains("private_key") {
		return nil, httperrors.NewUnsupportOperationError("Aliyun not allow to change certificate")
	}
	return data, nil
}

func (self *SAliyunRegionDriver) ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backends []cloudprovider.SLoadbalancerBackend) (*jsonutils.JSONDict, error) {
	groupType, _ := data.GetString("type")
	switch groupType {
	case "", models.LB_BACKENDGROUP_TYPE_NORMAL:
		break
	case models.LB_BACKENDGROUP_TYPE_MASTER_SLAVE:
		if len(backends) != 2 {
			return nil, httperrors.NewInputParameterError("master slave backendgorup must contain two backend")
		}
	default:
		return nil, httperrors.NewInputParameterError("Unsupport backendgorup type %s", groupType)
	}
	for _, backend := range backends {
		if len(backend.ExternalID) == 0 {
			return nil, httperrors.NewInputParameterError("invalid guest %s", backend.Name)
		}
	}
	return data, nil
}
