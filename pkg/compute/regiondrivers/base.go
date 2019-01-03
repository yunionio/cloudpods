package regiondrivers

import (
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaseRegionDriver struct {
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancer")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancer")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerAcl")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerAcl")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerCertificate")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerCertificate")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerBackendGroup")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerBackendGroup")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerBackend")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerBackend")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerListener")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerListener")
}
