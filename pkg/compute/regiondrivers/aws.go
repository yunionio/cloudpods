package regiondrivers

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAwsRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SAwsRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SAwsRegionDriver) GetProvider() string {
	return models.CLOUD_PROVIDER_AWS
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer", self.GetProvider())
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer acl", self.GetProvider())
}

func (self *SAwsRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotImplementedError("%s does not currently support creating loadbalancer certificate", self.GetProvider())
}
