package regiondrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SKVMRegionDriver struct {
	SBaseRegionDriver
}

func init() {
	driver := SKVMRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SKVMRegionDriver) GetProvider() string {
	return models.CLOUD_PROVIDER_KVM
}

func (self *SKVMRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SKVMRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SKVMRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SKVMRegionDriver) ValidateUpdateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SKVMRegionDriver) ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backends []cloudprovider.SLoadbalancerBackend) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_, err := models.LoadbalancerManager.TableSpec().Update(lb, func() error {
			if lb.AddressType == models.LB_ADDR_TYPE_INTRANET {
				// TODO support use reserved ip address
				// TODO prefer ip address from server_type loadbalancer?
				req := &models.SLoadbalancerNetworkRequestData{
					Loadbalancer: lb,
					NetworkId:    lb.NetworkId,
					Address:      lb.Address,
				}
				// NOTE the small window when agents can see the ephemeral address
				ln, err := models.LoadbalancernetworkManager.NewLoadbalancerNetwork(ctx, userCred, req)
				if err != nil {
					log.Errorf("allocating loadbalancer network failed: %v, req: %#v", err, req)
					lb.Address = ""
				} else {
					lb.Address = ln.IpAddr
				}
			}
			return nil
		})
		return nil, err
	})
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		for _, backend := range backends {
			loadbalancerBackend := models.SLoadbalancerBackend{
				BackendGroupId: lbbg.Id,
				BackendId:      backend.ID,
				BackendType:    backend.BackendType,
				BackendRole:    backend.BackendRole,
				Weight:         backend.Weight,
				Address:        backend.Address,
				Port:           backend.Port,
			}
			loadbalancerBackend.Status = models.LB_STATUS_ENABLED
			loadbalancerBackend.ProjectId = userCred.GetProjectId()
			loadbalancerBackend.Name = fmt.Sprintf("%s-%s-%s", lbbg.Name, backend.BackendType, backend.Name)
			if err := models.LoadbalancerBackendManager.TableSpec().Insert(&loadbalancerBackend); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SKVMRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}
