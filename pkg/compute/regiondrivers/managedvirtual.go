// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package regiondrivers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SManagedVirtualizationRegionDriver struct {
	SVirtualizationRegionDriver
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.ValidateManagerId(ctx, userCred, data)
}

func (self *SManagedVirtualizationRegionDriver) ValidateManagerId(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if managerId, _ := data.GetString("manager_id"); len(managerId) == 0 {
		return nil, httperrors.NewMissingParameterError("manager")
	}
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.ValidateManagerId(ctx, userCred, data)
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.ValidateManagerId(ctx, userCred, data)
}

func (self *SManagedVirtualizationRegionDriver) ValidateUpdateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendType string, lb *models.SLoadbalancer, backendGroup *models.SLoadbalancerBackendGroup, backend db.IModel) (*jsonutils.JSONDict, error) {
	if backendType != api.LB_BACKEND_GUEST {
		return nil, httperrors.NewUnsupportOperationError("internal error: unexpected backend type %s", backendType)
	}
	guest := backend.(*models.SGuest)
	host := guest.GetHost()
	if host == nil {
		return nil, fmt.Errorf("error getting host of guest %s", guest.GetId())
	}
	if lb == nil {
		return nil, fmt.Errorf("error loadbalancer of backend group %s", backendGroup.GetId())
	}
	hostRegion := host.GetRegion()
	lbRegion := lb.GetRegion()
	if hostRegion.Id != lbRegion.Id {
		return nil, httperrors.NewInputParameterError("region of host %q (%s) != region of loadbalancer %q (%s))",
			host.Name, host.ZoneId, lb.Name, lb.ZoneId)
	}
	address, err := models.LoadbalancerBackendManager.GetGuestAddress(guest)
	if err != nil {
		return nil, err
	}
	data.Set("address", jsonutils.NewString(address))
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lbbg *models.SLoadbalancerBackendGroup) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backends []cloudprovider.SLoadbalancerBackend) (*jsonutils.JSONDict, error) {
	for _, backend := range backends {
		if len(backend.ExternalID) == 0 {
			return nil, httperrors.NewInputParameterError("invalid guest %s", backend.Name)
		}
	}
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	_, err := self.ValidateManagerId(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	loadbalancerId, _ := data.GetString("loadbalancer_id")
	_, err = models.LoadbalancerManager.FetchById(loadbalancerId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("failed to find loadbalancer %s", loadbalancerId)
		}
		return nil, httperrors.NewGeneralError(err)
	}

	if aclStatus, _ := data.GetString("acl_status"); aclStatus == api.LB_BOOL_ON {
		aclId, _ := data.GetString("acl_id")
		if len(aclId) == 0 {
			return nil, httperrors.NewMissingParameterError("acl")
		}
		_, err = models.LoadbalancerAclManager.FetchById(aclId)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("failed to find acl %s", aclId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
	}
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lblis *models.SLoadbalancerListener, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	if listenerType, _ := data.GetString("listener_type"); len(listenerType) > 0 && lblis.ListenerType != listenerType {
		return nil, httperrors.NewInputParameterError("cannot change loadbalancer listener listener_type")
	}
	if listenerPort, _ := data.Int("listener_port"); listenerPort != 0 && listenerPort != int64(lblis.ListenerPort) {
		return nil, httperrors.NewInputParameterError("cannot change loadbalancer listener listener_port")
	}
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateDeleteLoadbalancerCondition(ctx context.Context, lb *models.SLoadbalancer) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateDeleteLoadbalancerBackendCondition(ctx context.Context, lbb *models.SLoadbalancerBackend) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateDeleteLoadbalancerBackendGroupCondition(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) GetBackendStatusForAdd() []string {
	return []string{api.VM_RUNNING}
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		params, err := lb.GetCreateLoadbalancerParams(iRegion)
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.CreateILoadBalancer(params)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lb, userCred, iLoadbalancer.GetGlobalId()); err != nil {
			return nil, err
		}
		if err := lb.SyncWithCloudLoadbalancer(ctx, userCred, iLoadbalancer, nil); err != nil {
			return nil, err
		}
		lbbgs, err := iLoadbalancer.GetILoadBalancerBackendGroups()
		if err != nil {
			return nil, err
		}
		if len(lbbgs) > 0 {
			provider := lb.GetCloudprovider()
			if provider == nil {
				return nil, fmt.Errorf("failed to find cloudprovider for lb %s", lb.Name)
			}
			models.LoadbalancerBackendGroupManager.SyncLoadbalancerBackendgroups(ctx, userCred, provider, lb, lbbgs, &models.SSyncRange{})
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestStartLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, err
		}
		return nil, iLoadbalancer.Start()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestStopLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, err
		}
		return nil, iLoadbalancer.Stop()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncstatusLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, err
		}
		status := iLoadbalancer.GetStatus()
		if utils.IsInStringArray(status, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_DISABLED}) {
			return nil, lb.SetStatus(userCred, status, "")
		}
		return nil, fmt.Errorf("Unknown loadbalancer status %s", status)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}

		if len(lb.ExternalId) == 0 {
			return nil, nil
		}

		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iLoadbalancer.Delete()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) createLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl) (jsonutils.JSONObject, error) {
	iRegion, err := lbacl.GetIRegion()
	if err != nil {
		return nil, err
	}

	lblis, err := lbacl.GetListener()
	if err != nil {
		return nil, err
	}

	acl := &cloudprovider.SLoadbalancerAccessControlList{
		Name:                lbacl.Name,
		Entrys:              []cloudprovider.SLoadbalancerAccessControlListEntry{},
		AccessControlEnable: lblis.AclStatus == api.LB_BOOL_ON,
	}
	if lbacl.AclEntries != nil {
		for _, entry := range *lbacl.AclEntries {
			acl.Entrys = append(acl.Entrys, cloudprovider.SLoadbalancerAccessControlListEntry{CIDR: entry.Cidr, Comment: entry.Comment})
		}
	}
	iLoadbalancerAcl, err := iRegion.CreateILoadBalancerAcl(acl)
	if err != nil {
		return nil, err
	}
	if err := db.SetExternalId(lbacl, userCred, iLoadbalancerAcl.GetGlobalId()); err != nil {
		return nil, err
	}
	return nil, lbacl.SyncWithCloudLoadbalancerAcl(ctx, userCred, iLoadbalancerAcl, lblis.GetOwnerId())
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.createLoadbalancerAcl(ctx, userCred, lbacl)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) syncLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl) (jsonutils.JSONObject, error) {
	iRegion, err := lbacl.GetIRegion()
	if err != nil {
		return nil, err
	}

	lblis, err := lbacl.GetListener()
	if err != nil {
		return nil, err
	}

	acl := &cloudprovider.SLoadbalancerAccessControlList{
		Name:                lbacl.Name,
		Entrys:              []cloudprovider.SLoadbalancerAccessControlListEntry{},
		AccessControlEnable: lblis.AclStatus == api.LB_BOOL_ON,
	}
	if lbacl.AclEntries != nil {
		for _, entry := range *lbacl.AclEntries {
			acl.Entrys = append(acl.Entrys, cloudprovider.SLoadbalancerAccessControlListEntry{CIDR: entry.Cidr, Comment: entry.Comment})
		}
	}
	lockman.LockRawObject(ctx, "acl", lbacl.Id)
	defer lockman.ReleaseRawObject(ctx, "acl", lbacl.Id)

	iLoadbalancerAcl, err := iRegion.GetILoadBalancerAclById(lbacl.ExternalId)
	if err != nil {
		return nil, err
	}
	return nil, iLoadbalancerAcl.Sync(acl)
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.syncLoadbalancerAcl(ctx, userCred, lbacl)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) deleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) (jsonutils.JSONObject, error) {
	if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
		return nil, nil
	}
	iRegion, err := lbacl.GetIRegion()
	if err != nil {
		return nil, err
	}

	if len(lbacl.ExternalId) == 0 {
		return nil, nil
	}

	iLoadbalancerAcl, err := iRegion.GetILoadBalancerAclById(lbacl.ExternalId)
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return nil, iLoadbalancerAcl.Delete()
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.deleteLoadbalancerAcl(ctx, userCred, lbacl, task)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) createLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SCachedLoadbalancerCertificate) (jsonutils.JSONObject, error) {
	iRegion, err := lbcert.GetIRegion()
	if err != nil {
		return nil, err
	}
	certificate := &cloudprovider.SLoadbalancerCertificate{
		Name:        lbcert.Name,
		PrivateKey:  lbcert.PrivateKey,
		Certificate: lbcert.Certificate,
	}
	iLoadbalancerCert, err := iRegion.CreateILoadBalancerCertificate(certificate)
	if err != nil {
		return nil, err
	}
	lbcert.SetModelManager(models.CachedLoadbalancerCertificateManager, lbcert)
	if err := db.SetExternalId(lbcert, userCred, iLoadbalancerCert.GetGlobalId()); err != nil {
		return nil, err
	}
	return nil, lbcert.SyncWithCloudLoadbalancerCertificate(ctx, userCred, iLoadbalancerCert, lbcert.GetOwnerId())
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SCachedLoadbalancerCertificate, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return self.createLoadbalancerCertificate(ctx, userCred, lbcert)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SCachedLoadbalancerCertificate, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		iRegion, err := lbcert.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancerCert, err := iRegion.GetILoadBalancerCertificateById(lbcert.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iLoadbalancerCert.Delete()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lbbg.GetIRegion()
		if err != nil {
			return nil, err
		}
		loadbalancer := lbbg.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		group := &cloudprovider.SLoadbalancerBackendGroup{
			Name:      lbbg.Name,
			GroupType: lbbg.Type,
			Backends:  backends,
		}
		iLoadbalancerBackendGroup, err := iLoadbalancer.CreateILoadBalancerBackendGroup(group)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lbbg, userCred, iLoadbalancerBackendGroup.GetGlobalId()); err != nil {
			return nil, err
		}
		iBackends, err := iLoadbalancerBackendGroup.GetILoadbalancerBackends()
		if err != nil {
			return nil, err
		}
		if len(iBackends) > 0 {
			provider := loadbalancer.GetCloudprovider()
			if provider == nil {
				return nil, fmt.Errorf("failed to find cloudprovider for lb %s", loadbalancer.Name)
			}
			models.LoadbalancerBackendManager.SyncLoadbalancerBackends(ctx, userCred, provider, lbbg, iBackends, &models.SSyncRange{})
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		iRegion, err := lbbg.GetIRegion()
		if err != nil {
			return nil, err
		}
		loadbalancer := lbbg.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}

		if len(lbbg.ExternalId) == 0 {
			return nil, nil
		}

		iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iLoadbalancerBackendGroup.Delete()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg := lbb.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			return nil, fmt.Errorf("failed to find lbbg for backend %s", lbb.Name)
		}
		lb := lbbg.GetLoadbalancer()
		if lb == nil {
			return nil, fmt.Errorf("failed to find lb for backendgroup %s", lbbg.Name)
		}
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, err
		}
		iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
		if err != nil {
			return nil, err
		}
		guest := lbb.GetGuest()
		if guest == nil {
			return nil, fmt.Errorf("failed to find guest for lbb %s", lbb.Name)
		}
		iLoadbalancerBackend, err := iLoadbalancerBackendGroup.AddBackendServer(guest.ExternalId, lbb.Weight, lbb.Port)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lbb, userCred, iLoadbalancerBackend.GetGlobalId()); err != nil {
			return nil, err
		}
		return nil, lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iLoadbalancerBackend, nil)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		lbbg := lbb.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			return nil, fmt.Errorf("failed to find lbbg for backend %s", lbb.Name)
		}
		lb := lbbg.GetLoadbalancer()
		if lb == nil {
			return nil, fmt.Errorf("failed to find lb for backendgroup %s", lbbg.Name)
		}
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, err
		}
		iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
		if err != nil {
			return nil, err
		}
		guest := lbb.GetGuest()
		if guest == nil {
			log.Warningf("failed to find guest for lbb %s", lbb.Name)
			return nil, nil
		}
		_, err = guest.GetIVM()
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iLoadbalancerBackendGroup.RemoveBackendServer(guest.ExternalId, lbb.Weight, lbb.Port)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg := lbb.GetLoadbalancerBackendGroup()
		if lbbg == nil {
			return nil, fmt.Errorf("failed to find lbbg for backend %s", lbb.Name)
		}
		lb := lbbg.GetLoadbalancer()
		if lb == nil {
			return nil, fmt.Errorf("failed to find lb for backendgroup %s", lbbg.Name)
		}
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, err
		}
		iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
		if err != nil {
			return nil, err
		}

		iBackend, err := iLoadbalancerBackendGroup.GetILoadbalancerBackendById(lbb.ExternalId)
		if err != nil {
			return nil, err
		}

		err = iBackend.SyncConf(lbb.Port, lbb.Weight)
		if err != nil {
			return nil, err
		}

		iBackend, err = iLoadbalancerBackendGroup.GetILoadbalancerBackendById(lbb.ExternalId)
		if err != nil {
			return nil, err
		}

		return nil, lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iBackend, nil)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		{
			certId, _ := task.GetParams().GetString("certificate_id")
			if len(certId) > 0 {
				provider := models.CloudproviderManager.FetchCloudproviderById(lblis.ManagerId)
				if provider == nil {
					return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
				}

				cert, err := models.LoadbalancerCertificateManager.FetchById(certId)
				if err != nil {
					return nil, err
				}

				lbcert, err := models.CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate(ctx, userCred, provider, lblis, cert.(*models.SLoadbalancerCertificate))
				if err != nil {
					return nil, err
				}

				if len(lbcert.ExternalId) == 0 {
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, err
					}
				}

				lblis.ExternalId = lbcert.ExternalId
			}
		}

		{
			aclId, _ := task.GetParams().GetString("acl_id")
			if len(aclId) > 0 {
				provider := models.CloudproviderManager.FetchCloudproviderById(lblis.ManagerId)
				if provider == nil {
					return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
				}

				acl, err := models.LoadbalancerAclManager.FetchById(aclId)
				if err != nil {
					return nil, err
				}

				lbacl, err := models.CachedLoadbalancerAclManager.GetOrCreateCachedAcl(ctx, userCred, provider, lblis, acl.(*models.SLoadbalancerAcl))
				if err != nil {
					return nil, err
				}

				if len(lbacl.ExternalId) == 0 {
					_, err = self.createLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, err
					}
				}

				lblis.AclId = lbacl.ExternalId
			}
		}

		params, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, err
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.CreateILoadBalancerListener(params)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lblis, userCred, iListener.GetGlobalId()); err != nil {
			return nil, err
		}
		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, nil)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, err
		}

		if len(loadbalancer.ExternalId) == 0 {
			return nil, nil
		}

		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iListener.Delete()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestStartLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			return nil, err
		}
		return nil, iListener.Start()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		{
			certId, _ := task.GetParams().GetString("certificate_id")
			if len(certId) > 0 {
				provider := models.CloudproviderManager.FetchCloudproviderById(lblis.ManagerId)
				if provider == nil {
					return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
				}

				cert, err := models.LoadbalancerCertificateManager.FetchById(certId)
				if err != nil {
					return nil, err
				}

				lbcert, err := models.CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate(ctx, userCred, provider, lblis, cert.(*models.SLoadbalancerCertificate))
				if err != nil {
					return nil, err
				}

				if len(lbcert.ExternalId) == 0 {
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, err
					}
				}

				// lblis.ExternalId = lbcert.ExternalId
			}
		}

		{
			aclId, _ := task.GetParams().GetString("acl_id")
			if len(aclId) > 0 {
				provider := models.CloudproviderManager.FetchCloudproviderById(lblis.ManagerId)
				if provider == nil {
					return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
				}

				var lbacl *models.SCachedLoadbalancerAcl
				// 先读取缓存，缓存不存在的情况下，从ACL表中取数据创建缓存
				if _lbacl, err := models.CachedLoadbalancerAclManager.FetchById(aclId); err == nil && _lbacl != nil {
					lbacl = _lbacl.(*models.SCachedLoadbalancerAcl)
				} else {
					acl, err := models.LoadbalancerAclManager.FetchById(aclId)
					if err != nil {
						return nil, err
					}

					lbacl, err = models.CachedLoadbalancerAclManager.GetOrCreateCachedAcl(ctx, userCred, provider, lblis, acl.(*models.SLoadbalancerAcl))
					if err != nil {
						return nil, err
					}
				}

				if len(lbacl.ExternalId) == 0 {
					_, err := self.createLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, err
					}
				}

				// lblis.AclId = lbacl.ExternalId
			}
		}

		params, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, err
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			return nil, err
		}
		if err := iListener.Sync(params); err != nil {
			return nil, err
		}
		if err := iListener.Refresh(); err != nil {
			return nil, err
		}
		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, nil)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestStopLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			return nil, err
		}
		return nil, iListener.Stop()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncstatusLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			return nil, err
		}
		status := iListener.GetStatus()
		if utils.IsInStringArray(status, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_DISABLED}) {
			return nil, lblis.SetStatus(userCred, status, "")
		}
		return nil, fmt.Errorf("Unknown loadbalancer listener status %s", status)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		listener := lbr.GetLoadbalancerListener()
		if listener == nil {
			return nil, fmt.Errorf("failed to find listener for listnener rule %s", lbr.Name)
		}
		loadbalancer := listener.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for listener %s", listener.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(listener.ExternalId)
		if err != nil {
			return nil, err
		}
		rule := &cloudprovider.SLoadbalancerListenerRule{
			Name:   lbr.Name,
			Domain: lbr.Domain,
			Path:   lbr.Path,
		}
		if len(lbr.BackendGroupId) > 0 {
			group := lbr.GetLoadbalancerBackendGroup()
			if group == nil {
				return nil, fmt.Errorf("failed to find backend group for listener rule %s", lbr.Name)
			}
			rule.BackendGroupID = group.ExternalId
			rule.BackendGroupType = group.Type
		}
		iListenerRule, err := iListener.CreateILoadBalancerListenerRule(rule)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lbr, userCred, iListenerRule.GetGlobalId()); err != nil {
			return nil, err
		}
		return nil, lbr.SyncWithCloudLoadbalancerListenerRule(ctx, userCred, iListenerRule, nil)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		listener := lbr.GetLoadbalancerListener()
		if listener == nil {
			return nil, fmt.Errorf("failed to find listener for listnener rule %s", lbr.Name)
		}
		loadbalancer := listener.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for listener %s", listener.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, err
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(listener.ExternalId)
		if err != nil {
			return nil, err
		}
		if len(lbr.ExternalId) == 0 {
			return nil, nil
		}
		iListenerRule, err := iListener.GetILoadBalancerListenerRuleById(lbr.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iListenerRule.Delete()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateEipData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := sp.GetIRegion()
		if err != nil {
			return nil, err
		}
		input, err := sp.GenerateCreateSpParams()
		if err != nil {
			return nil, err
		}
		policyId, err := iRegion.CreateSnapshotPolicy(input)
		if err != nil {
			return nil, err
		}
		err = db.SetExternalId(sp, userCred, policyId)
		if err != nil {
			return nil, err
		}

		iPolicy, err := iRegion.GetISnapshotPolicyById(policyId)
		if err != nil {
			return nil, err
		}
		err = cloudprovider.WaitStatus(iPolicy, api.SNAPSHOT_POLICY_READY, 10*time.Second, 300*time.Second)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := sp.GetIRegion()
		if err != nil {
			return nil, err
		}
		err = iRegion.DeleteSnapshotPolicy(sp.GetExternalId())
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestApplySnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, task taskman.ITask, diskIds []string) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := sp.GetIRegion()
		if err != nil {
			return nil, err
		}
		err = iRegion.ApplySnapshotPolicyToDisks(sp.GetExternalId(), diskIds)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCancelSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region cloudprovider.ICloudRegion, task taskman.ITask, diskIds []string) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		err := region.CancelSnapshotPolicyToDisks(diskIds)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return nil
}
