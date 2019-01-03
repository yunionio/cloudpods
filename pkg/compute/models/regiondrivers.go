package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IRegionDriver interface {
	GetProvider() string

	ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error
	RequestDeleteLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error

	ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *SLoadbalancerAcl, task taskman.ITask) error
	RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *SLoadbalancerAcl, task taskman.ITask) error

	ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateUpdateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *SLoadbalancerCertificate, task taskman.ITask) error
	RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *SLoadbalancerCertificate, task taskman.ITask) error

	ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backends []cloudprovider.SLoadbalancerBackend) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error
	RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *SLoadbalancerBackendGroup, task taskman.ITask) error

	RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *SLoadbalancerBackend, task taskman.ITask) error
	RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *SLoadbalancerBackend, task taskman.ITask) error

	RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, task taskman.ITask) error
	RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, task taskman.ITask) error
}

var regionDrivers map[string]IRegionDriver

func init() {
	regionDrivers = make(map[string]IRegionDriver)
}

func RegisterRegionDriver(driver IRegionDriver) {
	regionDrivers[driver.GetProvider()] = driver
}

func GetRegionDriver(provider string) IRegionDriver {
	driver, ok := regionDrivers[provider]
	if ok {
		return driver
	}
	log.Fatalf("Unsupported provider %s", provider)
	return nil
}
