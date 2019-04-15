package regiondrivers

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SEsxiRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SEsxiRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SEsxiRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_VMWARE
}

func (self *SEsxiRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewUnsupportOperationError("%s does not support creating loadbalancer", self.GetProvider())
}

func (self *SEsxiRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not support creating loadbalancer acl", self.GetProvider())
}

func (self *SEsxiRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not support creating loadbalancer certificate", self.GetProvider())
}
