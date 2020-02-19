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
	"sort"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rand"
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

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	_, err := self.ValidateManagerId(ctx, userCred, data)
	if err != nil {
		return nil, err
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
		//公网lb,需要同步public ip
		if lb.AddressType == api.LB_ADDR_TYPE_INTERNET {
			publicIp, err := iLoadbalancer.GetIEIP()
			if err != nil {
				return nil, errors.Wrap(err, "iLoadbalancer.GetIEIP()")
			}
			lb.SyncLoadbalancerEip(ctx, userCred, lb.GetCloudprovider(), publicIp)
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

	acl := &cloudprovider.SLoadbalancerAccessControlList{
		Name:   lbacl.Name,
		Entrys: []cloudprovider.SLoadbalancerAccessControlListEntry{},
	}

	_originAcl, err := db.FetchById(models.LoadbalancerAclManager, lbacl.AclId)
	if err != nil {
		return nil, errors.Wrap(err, "regionDriver.FetchAcl")
	}

	originAcl := _originAcl.(*models.SLoadbalancerAcl)
	if originAcl.AclEntries != nil {
		for _, entry := range *originAcl.AclEntries {
			acl.Entrys = append(acl.Entrys, cloudprovider.SLoadbalancerAccessControlListEntry{CIDR: entry.Cidr, Comment: entry.Comment})
		}
	}
	iLoadbalancerAcl, err := iRegion.CreateILoadBalancerAcl(acl)
	if err != nil {
		return nil, err
	}

	lbacl.SetModelManager(models.CachedLoadbalancerAclManager, lbacl)
	if err := db.SetExternalId(lbacl, userCred, iLoadbalancerAcl.GetGlobalId()); err != nil {
		return nil, err
	}
	return nil, lbacl.SyncWithCloudLoadbalancerAcl(ctx, userCred, iLoadbalancerAcl, lbacl.GetOwnerId())
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

	acl := &cloudprovider.SLoadbalancerAccessControlList{
		Name:   lbacl.Name,
		Entrys: []cloudprovider.SLoadbalancerAccessControlListEntry{},
	}

	_localAcl, err := db.FetchById(models.LoadbalancerAclManager, lbacl.AclId)
	if err != nil {
		return nil, errors.Wrap(err, "regionDriver.FetchById.LoaclAcl")
	}

	localAcl := _localAcl.(*models.SLoadbalancerAcl)
	if localAcl.AclEntries != nil {
		for _, entry := range *localAcl.AclEntries {
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
		return nil, errors.Wrapf(err, "lbcert.GetIRegion")
	}

	_localCert, err := db.FetchById(models.LoadbalancerCertificateManager, lbcert.CertificateId)
	if err != nil {
		return nil, errors.Wrapf(err, "regionDriver.FetchById.localcert")
	}

	localCert := _localCert.(*models.SLoadbalancerCertificate)
	certificate := &cloudprovider.SLoadbalancerCertificate{
		Name:        fmt.Sprintf("%s-%s", lbcert.Name, rand.String(4)),
		PrivateKey:  localCert.PrivateKey,
		Certificate: localCert.Certificate,
	}
	iLoadbalancerCert, err := iRegion.CreateILoadBalancerCertificate(certificate)
	if err != nil {
		return nil, errors.Wrap(err, "iRegion.CreateILoadBalancerCertificate")
	}
	lbcert.SetModelManager(models.CachedLoadbalancerCertificateManager, lbcert)
	if err := db.SetExternalId(lbcert, userCred, iLoadbalancerCert.GetGlobalId()); err != nil {
		return nil, errors.Wrap(err, "db.SetExternalId")
	}

	err = cloudprovider.WaitCreated(3*time.Second, 30*time.Second, func() bool {
		err := iLoadbalancerCert.Refresh()
		if err == nil {
			return true
		}
		return false
	})
	if err != nil {
		return nil, errors.Wrap(err, "iRegion.createLoadbalancerCertificate.WaitCreated")
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
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
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

		err = iLoadbalancerBackendGroup.Delete()
		if err != nil {
			return nil, err
		}

		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestPullRegionLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults models.SSyncResultSet, provider *models.SCloudprovider, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestPullLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults models.SSyncResultSet, provider *models.SCloudprovider, localLoadbalancer *models.SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *models.SSyncRange) error {
	models.SyncLoadbalancerBackendgroups(ctx, userCred, syncResults, provider, localLoadbalancer, remoteLoadbalancer, syncRange)
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
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerBackend.GetIRegion")
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerBackend.GetILoadBalancerById")
		}
		iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerBackend.GetILoadBalancerBackendGroupById")
		}

		iBackend, err := iLoadbalancerBackendGroup.GetILoadbalancerBackendById(lbb.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerBackend.GetILoadbalancerBackendById")
		}

		err = iBackend.SyncConf(lbb.Port, lbb.Weight)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerBackend.SyncConf")
		}

		iBackend, err = iLoadbalancerBackendGroup.GetILoadbalancerBackendById(lbb.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerBackend.GetILoadbalancerBackendById")
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
					return nil, errors.Wrapf(err, "LoadbalancerCertificateManager.FetchById(%s)", certId)
				}

				lbcert, err := models.CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate(ctx, userCred, provider, lblis, cert.(*models.SLoadbalancerCertificate))
				if err != nil {
					return nil, errors.Wrap(err, "CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate")
				}

				if len(lbcert.ExternalId) == 0 {
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, errors.Wrap(err, "createLoadbalancerCertificate")
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedCertificateId = lbcert.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "regionDriver.RequestCreateLoadbalancerListener.UpdateCachedCertificateId")
				}
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
					return nil, errors.Wrap(err, "LoadbalancerAclManager.FetchById")
				}

				lbacl, err := models.CachedLoadbalancerAclManager.GetOrCreateCachedAcl(ctx, userCred, provider, lblis, acl.(*models.SLoadbalancerAcl))
				if err != nil {
					return nil, errors.Wrap(err, "CachedLoadbalancerAclManager.GetOrCreateCachedAcl")
				}

				if len(lbacl.ExternalId) == 0 {
					_, err = self.createLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, errors.Wrap(err, "createLoadbalancerAcl")
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedAclId = lbacl.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "regionDriver.RequestCreateLoadbalancerListener.UpdateCachedAclId")
				}
			}
		}

		params, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, errors.Wrapf(err, "lblis.GetLoadbalancerListenerParams")
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "loadbalancer.GetIRegion")
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "iRegion.GetILoadBalancerById(%s)", loadbalancer.ExternalId)
		}
		iListener, err := iLoadbalancer.CreateILoadBalancerListener(params)
		if err != nil {
			return nil, errors.Wrap(err, "iLoadbalancer.CreateILoadBalancerListener")
		}
		if err := db.SetExternalId(lblis, userCred, iListener.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "db.SetExternalId")
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
			return nil, errors.Wrap(err, "RegionDriver.RequestDeleteLoadbalancerListener.GetIRegion")
		}

		if len(loadbalancer.ExternalId) == 0 {
			return nil, nil
		}

		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrap(err, "RegionDriver.RequestDeleteLoadbalancerListener.GetILoadBalancerById")
		}

		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrap(err, "RegionDriver.RequestDeleteLoadbalancerListener.GetILoadBalancerListenerById")
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
					return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.FetchCert")
				}

				lbcert, err := models.CachedLoadbalancerCertificateManager.GetOrCreateCachedCertificate(ctx, userCred, provider, lblis, cert.(*models.SLoadbalancerCertificate))
				if err != nil {
					return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetCert")
				}

				if len(lbcert.ExternalId) == 0 {
					_, err = self.createLoadbalancerCertificate(ctx, userCred, lbcert)
					if err != nil {
						return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.CreateCert")
					}
				}

				_, err = db.Update(lblis, func() error {
					lblis.CachedCertificateId = lbcert.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.UpdateCachedCertificateId")
				}
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
						return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.FetchAcl")
					}

					lbacl, err = models.CachedLoadbalancerAclManager.GetOrCreateCachedAcl(ctx, userCred, provider, lblis, acl.(*models.SLoadbalancerAcl))
					if err != nil {
						return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetAcl")
					}
				}

				if len(lbacl.ExternalId) == 0 {
					_, err := self.createLoadbalancerAcl(ctx, userCred, lbacl)
					if err != nil {
						return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.CreateAcl")
					}
				}

				_, err := db.Update(lblis, func() error {
					lblis.CachedAclId = lbacl.GetId()
					return nil
				})
				if err != nil {
					return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.UpdateCachedAclId")
				}
			}
		}

		params, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetParams")
		}
		loadbalancer := lblis.GetLoadbalancer()
		if loadbalancer == nil {
			return nil, fmt.Errorf("failed to find loadbalancer for lblis %s", lblis.Name)
		}
		iRegion, err := loadbalancer.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetIRegion")
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(loadbalancer.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetILoadbalancer")
		}
		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.GetIListener")
		}
		if err := iListener.Sync(params); err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.SyncListener")
		}
		if err := iListener.Refresh(); err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.RefreshListener")
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

func (self *SManagedVirtualizationRegionDriver) ValidateCreateEipData(ctx context.Context, userCred mcclient.TokenCredential, input *api.SElasticipCreateInput) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iregion, err := vpc.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetIRegion")
		}
		ivpc, err := iregion.CreateIVpc(vpc.Name, vpc.Description, vpc.CidrBlock)
		if err != nil {
			return nil, errors.Wrap(err, "iregion.CreateIVpc")
		}
		db.SetExternalId(vpc, userCred, ivpc.GetGlobalId())

		err = cloudprovider.WaitStatus(ivpc, api.VPC_STATUS_AVAILABLE, 10*time.Second, 300*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "cloudprovider.WaitStatus")
		}

		err = vpc.SyncWithCloudVpc(ctx, userCred, ivpc)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncWithCloudVpc")
		}

		err = vpc.SyncRemoteWires(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncRemoteWires")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		region, err := vpc.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetIRegion")
		}
		ivpc, err := region.GetIVpcById(vpc.GetExternalId())
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				// already deleted, do nothing
				return nil, nil
			}
			return nil, errors.Wrap(err, "region.GetIVpcById")
		}
		err = ivpc.Delete()
		if err != nil {
			return nil, errors.Wrap(err, "ivpc.Delete(")
		}
		err = cloudprovider.WaitDeleted(ivpc, 10*time.Second, 300*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "cloudprovider.WaitDeleted")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestUpdateSnapshotPolicy(ctx context.Context, userCred mcclient.
	TokenCredential, sp *models.SSnapshotPolicy, input cloudprovider.SnapshotPolicyInput, task taskman.ITask) error {
	// it's too cumbersome to pass parameters in taskman, so change a simple way for the moment

	//spcache, err := models.SnapshotPolicyCacheManager.FetchSnapshotPolicyCache(sp.GetId(), sp.CloudregionId, sp.ManagerId)
	//if err != nil {
	//	return errors.Wrapf(err, "Fetch cache ofsnapshotpolicy %s", sp.GetId())
	//}
	//return spcache.UpdateCloudSnapshotPolicy(&input)

	return nil
}

// RequestApplySnapshotPolicy apply snapshotpolicy for public cloud.
// In our system, one disk only can hava one snapshot policy attached.
// Default, some public cloud such as Aliyun is same with us and this function shoule be used for these public cloud.
// But in Some public cloud such as Qcloud different with us,
// we should wirte a new function in corressponding regiondriver which detach all snapshotpolicy of disk after
// attache new one.
// You can refer to the implementations of function SQcloudRegionDriver.RequestApplySnapshotPolicy().
func (self *SManagedVirtualizationRegionDriver) RequestApplySnapshotPolicy(ctx context.Context,
	userCred mcclient.TokenCredential, task taskman.ITask, disk *models.SDisk, sp *models.SSnapshotPolicy,
	data jsonutils.JSONObject) error {

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		regionId := disk.GetStorage().GetRegion().GetId()
		providerId := disk.GetStorage().ManagerId
		spcache, err := models.SnapshotPolicyCacheManager.Register(ctx, userCred, sp.GetId(), regionId, providerId)
		if err != nil {
			return nil, errors.Wrap(err, "registersnapshotpolicy cache failed")
		}

		iRegion, err := disk.GetIRegion()
		if err != nil {
			return nil, err
		}

		err = iRegion.ApplySnapshotPolicyToDisks(spcache.GetExternalId(), disk.GetExternalId())
		if err != nil {
			return nil, err
		}
		data := jsonutils.NewDict()
		data.Add(jsonutils.NewString(sp.GetId()), "snapshotpolicy_id")
		return data, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCancelSnapshotPolicy(ctx context.Context, userCred mcclient.
	TokenCredential, task taskman.ITask, disk *models.SDisk, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) error {

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		regionId := disk.GetStorage().GetRegion().GetId()
		providerId := disk.GetStorage().ManagerId
		spcache, err := models.SnapshotPolicyCacheManager.FetchSnapshotPolicyCache(sp.GetId(), regionId, providerId)

		if err != nil {
			return nil, errors.Wrap(err, "registersnapshotpolicy cache failed")
		}

		iRegion, err := spcache.GetIRegion()
		if err != nil {
			return nil, err
		}
		data := jsonutils.NewDict()
		data.Add(jsonutils.NewString(sp.GetId()), "snapshotpolicy_id")
		err = iRegion.CancelSnapshotPolicyToDisks(spcache.GetExternalId(), disk.GetExternalId())
		if err == cloudprovider.ErrNotFound {
			return data, nil
		}
		if err != nil {
			return nil, err
		}
		return data, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		cloudRegion, err := snapshot.GetISnapshotRegion()
		if err != nil {
			return nil, err
		}
		cloudSnapshot, err := cloudRegion.GetISnapshotById(snapshot.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		if err := cloudSnapshot.Delete(); err != nil {
			return nil, err
		}
		if err := cloudprovider.WaitDeleted(cloudSnapshot, 10*time.Second, 300*time.Second); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SnapshotCreateInput) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	disk, err := snapshot.GetDisk()
	if err != nil {
		return err
	}
	iDisk, err := disk.GetIDisk()
	if err != nil {
		return err
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iSnapshot, err := iDisk.CreateISnapshot(ctx, snapshot.Name, "")
		if err != nil {
			return nil, err
		}
		_, err = db.Update(snapshot, func() error {
			snapshot.Size = int(iSnapshot.GetSizeMb())
			snapshot.ExternalId = iSnapshot.GetGlobalId()
			return nil
		})
		return nil, err
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) GetDiskResetParams(snapshot *models.SSnapshot) *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshot.ExternalId))
	return params
}

func (self *SManagedVirtualizationRegionDriver) OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential,
	disk *models.SDisk, snapshot *models.SSnapshot, data jsonutils.JSONObject) error {

	externalId, _ := data.GetString("exteranl_disk_id")
	if len(externalId) > 0 {
		_, err := db.Update(disk, func() error {
			disk.ExternalId = externalId
			return nil
		})
		if err != nil {
			return err
		}
	}
	iDisk, err := disk.GetIDisk()
	if err != nil {
		return err
	}
	err = iDisk.Refresh()
	if err != nil {
		return err
	}
	if disk.DiskSize != iDisk.GetDiskSizeMB() {
		_, err := db.Update(disk, func() error {
			disk.DiskSize = iDisk.GetDiskSizeMB()
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateSnapshopolicyDiskData(ctx context.Context,
	userCred mcclient.TokenCredential, disk *models.SDisk, snapshotPolicy *models.SSnapshotPolicy) error {

	err := self.SBaseRegionDriver.ValidateCreateSnapshopolicyDiskData(ctx, userCred, disk, snapshotPolicy)
	if err != nil {
		return err
	}

	if snapshotPolicy.RetentionDays < -1 || snapshotPolicy.RetentionDays == 0 || snapshotPolicy.RetentionDays > 65535 {
		return httperrors.NewInputParameterError("Retention days must in 1~65535 or -1")
	}
	return nil
}

func (self *SManagedVirtualizationRegionDriver) OnSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask, data jsonutils.JSONObject) error {
	task.SetStage("OnManagedSnapshotDelete", nil)
	task.ScheduleRun(data)
	return nil
}

func (self *SManagedVirtualizationRegionDriver) DealNatGatewaySpec(spec string) string {
	return spec
}

func (self *SManagedVirtualizationRegionDriver) RequestBindIPToNatgateway(ctx context.Context, task taskman.ITask,
	natgateway *models.SNatGateway, eipId string) error {

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		model, err := models.ElasticipManager.FetchById(eipId)
		if err != nil {
			return nil, err
		}
		lockman.LockObject(ctx, model)
		defer lockman.ReleaseObject(ctx, model)
		eip := model.(*models.SElasticip)
		// check again
		if len(eip.AssociateId) > 0 {
			return nil, fmt.Errorf("eip %s has been associated with resource %s", eip.Id, eip.AssociateId)
		}
		_, err = db.Update(eip, func() error {
			eip.AssociateType = api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
			eip.AssociateId = natgateway.GetId()
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "fail to update eip '%s' in database", eip.Id)
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestUnBindIPFromNatgateway(ctx context.Context, task taskman.ITask,
	nat models.INatHelper, natgateway *models.SNatGateway) error {

	eip := &models.SElasticip{}
	err := models.ElasticipManager.Query().Equals("associate_id", natgateway.Id).First(eip)
	if err != nil {
		return errors.Wrapf(err, "fail to fetch eip associate with natgateway %s", natgateway.Id)
	}
	eip.SetModelManager(models.ElasticipManager, eip)
	lockman.LockObject(ctx, eip)
	defer lockman.ReleaseObject(ctx, eip)
	iregion, err := eip.GetIRegion()
	if err != nil {
		return errors.Wrapf(err, "fail to fetch iregion of eip %s", eip.Id)
	}
	ieip, err := iregion.GetIEipById(eip.GetExternalId())
	if err != nil {
		return errors.Wrapf(err, "fail to fetch cloudeip of eip %s", eip.Id)
	}
	err = eip.SyncInstanceWithCloudEip(ctx, task.GetUserCred(), ieip)
	if err != nil {
		return errors.Wrapf(err, "fail to sync eip %s from cloud", eip.Id)
	}
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestPreSnapshotPolicyApply(ctx context.Context, userCred mcclient.
	TokenCredential, task taskman.ITask, disk *models.SDisk, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) error {

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		return data, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) BindIPToNatgatewayRollback(ctx context.Context, eipId string) error {
	model, err := models.ElasticipManager.FetchById(eipId)
	if err != nil {
		return err
	}
	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)
	eip := model.(*models.SElasticip)
	if eip.AssociateType != api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY {
		return nil
	}
	_, err = db.Update(eip, func() error {
		eip.AssociateId = ""
		eip.AssociateType = ""
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "rollback about binding eip %s failed", eip.Id)
	}
	return nil
}

func (self *SManagedVirtualizationRegionDriver) GetSecurityGroupVpcId(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, host *models.SHost, vpc *models.SVpc, classic bool) (string, error) {
	if region.GetDriver().IsSecurityGroupBelongGlobalVpc() {
		return strings.TrimPrefix(vpc.ExternalId, region.ExternalId+"/"), nil
	} else if region.GetDriver().IsSupportClassicSecurityGroup() && (classic || (host != nil && strings.HasSuffix(host.Name, "-classic"))) {
		return "classic", nil
	} else if region.GetDriver().IsSecurityGroupBelongVpc() {
		return vpc.ExternalId, nil
	}
	return region.GetDriver().GetDefaultSecurityGroupVpcId(), nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, vpcId string, vpc *models.SVpc, secgroup *models.SSecurityGroup) (string, error) {
	lockman.LockRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-%s-%s", secgroup.Id, vpcId, vpc.ManagerId))
	defer lockman.ReleaseRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-%s-%s", secgroup.Id, vpcId, vpc.ManagerId))

	region, err := vpc.GetRegion()
	if err != nil {
		return "", errors.Wrap(err, "vpc.GetRegon")
	}

	cache, err := models.SecurityGroupCacheManager.Register(ctx, userCred, secgroup.Id, vpcId, region.Id, vpc.ManagerId)
	if err != nil {
		return "", errors.Wrap(err, "SSecurityGroupCache.Register")
	}

	iRegion, err := vpc.GetIRegion()
	if err != nil {
		return "", errors.Wrap(err, "vpc.GetIRegion")
	}

	var iSecgroup cloudprovider.ICloudSecurityGroup = nil
	if len(cache.ExternalId) > 0 {
		iSecgroup, err = iRegion.GetISecurityGroupById(cache.ExternalId)
		if err != nil {
			if err != cloudprovider.ErrNotFound {
				return "", errors.Wrap(err, "iRegion.GetSecurityGroupById")
			}
			cache.ExternalId = ""
		}
	}

	if len(cache.ExternalId) == 0 {
		if strings.ToLower(secgroup.Name) == "default" { //避免有些云不支持default关键字
			secgroup.Name = "DefaultGroup"
		}
		// 避免有的云不支持重名安全组
		groupName := secgroup.Name
		for i := 0; i < 30; i++ {
			_, err := iRegion.GetISecurityGroupByName(vpc.ExternalId, groupName)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					break
				}
				if errors.Cause(err) != cloudprovider.ErrDuplicateId {
					return "", err
				}
			}
			groupName = fmt.Sprintf("%s-%d", secgroup.Name, i)
		}
		conf := &cloudprovider.SecurityGroupCreateInput{
			Name:  groupName,
			Desc:  secgroup.Description,
			VpcId: vpcId,
			Rules: secgroup.GetSecRules(""),
		}
		iSecgroup, err = iRegion.CreateISecurityGroup(conf)
		if err != nil {
			return "", errors.Wrap(err, "iRegion.CreateISecurityGroup")
		}
	}

	_, err = db.Update(cache, func() error {
		cache.ExternalId = iSecgroup.GetGlobalId()
		cache.Name = iSecgroup.GetName()
		cache.Status = api.SECGROUP_CACHE_STATUS_READY
		return nil
	})

	if err != nil {
		return "", errors.Wrap(err, "db.Update")
	}

	inAllowList := secgroup.GetInAllowList()
	outAllowList := secgroup.GetOutAllowList()

	rules, err := iSecgroup.GetRules()
	if err != nil {
		return "", errors.Wrap(err, "iSecgroup.GetRules")
	}

	inRules := secrules.SecurityRuleSet{}
	outRules := secrules.SecurityRuleSet{}
	for i := 0; i < len(rules); i++ {
		if rules[i].Direction == secrules.DIR_IN {
			inRules = append(inRules, rules[i])
		} else {
			outRules = append(outRules, rules[i])
		}
	}
	sort.Sort(inRules)
	sort.Sort(outRules)
	_inAllowList := inRules.AllowList()
	_outAllowList := outRules.AllowList()
	if inAllowList.Equals(_inAllowList) && outAllowList.Equals(_outAllowList) {
		return cache.ExternalId, nil
	}

	err = iSecgroup.SyncRules(secgroup.GetSecRules(""))
	if err != nil {
		return "", errors.Wrap(err, "iSecgroup.SyncRules")
	}
	return cache.ExternalId, nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCacheSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, secgroup *models.SSecurityGroup, classic bool, task taskman.ITask) error {

	vpcId, err := region.GetDriver().GetSecurityGroupVpcId(ctx, userCred, region, nil, vpc, classic)
	if err != nil {
		return errors.Wrap(err, "GetSecurityGroupVpcId")
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_, err := self.RequestSyncSecurityGroup(ctx, userCred, vpcId, vpc, secgroup)
		return nil, err
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *models.SDBInstance, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iregion, err := dbinstance.GetIRegion()
		if err != nil {
			return nil, err
		}

		vpc, err := dbinstance.GetVpc()
		if err != nil {
			return nil, errors.Wrap(err, "dbinstance.GetVpc()")
		}

		params := task.GetParams()
		networkId, _ := params.GetString("network_external_id")
		if len(networkId) == 0 {
			return nil, fmt.Errorf("failed to get network externalId")
		}
		address, _ := params.GetString("address")
		passwd, _ := params.GetString("password")
		desc := cloudprovider.SManagedDBInstanceCreateConfig{
			Name:          dbinstance.Name,
			Description:   dbinstance.Description,
			StorageType:   dbinstance.StorageType,
			DiskSizeGB:    dbinstance.DiskSizeGB,
			InstanceType:  dbinstance.InstanceType,
			VcpuCount:     dbinstance.VcpuCount,
			VmemSizeMb:    dbinstance.VmemSizeMb,
			VpcId:         vpc.ExternalId,
			NetworkId:     networkId,
			Address:       address,
			Engine:        dbinstance.Engine,
			EngineVersion: dbinstance.EngineVersion,
			Category:      dbinstance.Category,
			Port:          dbinstance.Port,
			Password:      passwd,
		}

		if len(dbinstance.InstanceType) > 0 {
			desc.ZoneIds, _ = dbinstance.GetAvailableZoneIds()
		} else {
			desc.InstanceTypes, _ = dbinstance.GetAvailableInstanceTypes()
		}

		region := dbinstance.GetRegion()

		err = region.GetDriver().InitDBInstanceUser(dbinstance, task, &desc)
		if err != nil {
			return nil, err
		}

		secgroup, _ := dbinstance.GetSecgroup()
		if secgroup != nil {
			vpcId, err := region.GetDriver().GetSecurityGroupVpcId(ctx, userCred, region, nil, vpc, false)
			if err != nil {
				return nil, errors.Wrap(err, "GetSecurityGroupVpcId")
			}
			desc.SecgroupId, err = region.GetDriver().RequestSyncSecurityGroup(ctx, userCred, vpcId, vpc, secgroup)
			if err != nil {
				return nil, errors.Wrap(err, "SyncSecurityGroup")
			}
		}

		if dbinstance.BillingType == billing_api.BILLING_TYPE_PREPAID {
			bc, err := billing.ParseBillingCycle(dbinstance.BillingCycle)
			if err != nil {
				log.Errorf("failed to parse billing cycle %s: %v", dbinstance.BillingCycle, err)
			} else if bc.IsValid() {
				desc.BillingCycle = &bc
			}
		}

		if len(dbinstance.MasterInstanceId) > 0 {
			master, err := dbinstance.GetMasterInstance()
			if err != nil {
				return nil, errors.Wrap(err, "dbinstnace.GetMasterInstance()")
			}
			desc.MasterInstanceId = master.ExternalId
		}

		log.Debugf("create dbinstance params: %s", jsonutils.Marshal(desc).String())

		idbinstance, err := iregion.CreateIDBInstance(&desc)
		if idbinstance != nil { //避免创建失败后,删除本地的未能同步删除云上失败的RDS
			db.SetExternalId(dbinstance, userCred, idbinstance.GetGlobalId())
		}
		if err != nil {
			return nil, err
		}

		err = cloudprovider.WaitStatus(idbinstance, api.DBINSTANCE_RUNNING, time.Second*5, time.Hour*1)
		if err != nil {
			log.Errorf("timeout for waiting dbinstance running error: %v", err)
		}

		dbinstance.ZoneId = idbinstance.GetIZoneId()
		err = dbinstance.SetZoneInfo(ctx, userCred)
		if err != nil {
			log.Errorf("failed to set dbinstance %s(%s) zoneInfo from cloud dbinstance: %v", dbinstance.Name, dbinstance.Id, err)
		}

		_, err = db.Update(dbinstance, func() error {
			dbinstance.Engine = idbinstance.GetEngine()
			dbinstance.EngineVersion = idbinstance.GetEngineVersion()
			dbinstance.StorageType = idbinstance.GetStorageType()
			dbinstance.DiskSizeGB = idbinstance.GetDiskSizeGB()
			dbinstance.Category = idbinstance.GetCategory()
			dbinstance.VcpuCount = idbinstance.GetVcpuCount()
			dbinstance.VmemSizeMb = idbinstance.GetVmemSizeMB()
			dbinstance.InstanceType = idbinstance.GetInstanceType()
			dbinstance.ConnectionStr = idbinstance.GetConnectionStr()
			dbinstance.InternalConnectionStr = idbinstance.GetInternalConnectionStr()
			dbinstance.MaintainTime = idbinstance.GetMaintainTime()
			dbinstance.Port = idbinstance.GetPort()

			if createdAt := idbinstance.GetCreatedAt(); !createdAt.IsZero() {
				dbinstance.CreatedAt = idbinstance.GetCreatedAt()
			}
			if expiredAt := idbinstance.GetExpiredAt(); !expiredAt.IsZero() {
				dbinstance.ExpiredAt = expiredAt
			}
			return nil
		})
		if err != nil {
			log.Errorf("failed to update dbinstance conf: %v", err)
		}

		network, err := idbinstance.GetDBNetwork()
		if err != nil {
			log.Errorf("failed to get get network for dbinstance %s(%s) error: %v", dbinstance.Name, dbinstance.Id, err)
		} else {
			models.DBInstanceNetworkManager.SyncDBInstanceNetwork(ctx, userCred, dbinstance, network)
		}

		parameters, err := idbinstance.GetIDBInstanceParameters()
		if err != nil {
			log.Errorf("failed to get parameters for dbinstance %s(%s) error: %v", dbinstance.Name, dbinstance.Id, err)
		} else {
			models.DBInstanceParameterManager.SyncDBInstanceParameters(ctx, userCred, dbinstance, parameters)
		}

		backups, err := idbinstance.GetIDBInstanceBackups()
		if err != nil {
			log.Errorf("failed to get backups for dbinstance %s(%s) error: %v", dbinstance.Name, dbinstance.Id, err)
		} else {
			models.DBInstanceBackupManager.SyncDBInstanceBackups(ctx, userCred, dbinstance.GetCloudprovider(), dbinstance, dbinstance.GetRegion(), backups)
		}

		databases, err := idbinstance.GetIDBInstanceDatabases()
		if err != nil {
			log.Errorf("failed to get databases for databases %s(%s) error: %v", dbinstance.Name, dbinstance.Id, err)
		} else {
			models.DBInstanceDatabaseManager.SyncDBInstanceDatabases(ctx, userCred, dbinstance, databases)
		}

		return nil, nil
	})

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.ValidateManagerId(ctx, userCred, data)
}

func (self *SManagedVirtualizationRegionDriver) RequestRestartElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestRestartElasticcache.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestRestartElasticcache.GetIElasticcacheById")
	}

	err = iec.Restart()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestRestartElasticcache.Restart")
	}

	err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
	if err != nil {
		return err
	}

	return ec.SetStatus(userCred, api.ELASTIC_CACHE_STATUS_RUNNING, "")
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			ec.SetStatus(userCred, api.ELASTIC_CACHE_STATUS_UNKNOWN, "")
			return nil
		}

		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetIElasticcacheById")
	}

	provider := ec.GetCloudprovider()
	if provider == nil {
		return errors.Wrap(fmt.Errorf("provider is nil"), "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetCloudprovider")
	}

	lockman.LockClass(ctx, models.ElasticcacheManager, db.GetLockClassKey(models.ElasticcacheManager, provider.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, models.ElasticcacheManager, db.GetLockClassKey(models.ElasticcacheManager, provider.GetOwnerId()))

	err = ec.SyncWithCloudElasticcache(ctx, userCred, provider, iec)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetIElasticcacheById")
	}

	if fullsync, _ := task.GetParams().Bool("full"); fullsync {
		lockman.LockObject(ctx, ec)
		defer lockman.ReleaseObject(ctx, ec)

		parameters, err := iec.GetICloudElasticcacheParameters()
		if err != nil {
			return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetICloudElasticcacheParameters")
		}

		result := models.ElasticcacheParameterManager.SyncElasticcacheParameters(ctx, userCred, ec, parameters)
		log.Debugf("managedVirtualizationRegionDriver.RequestSyncElasticcache.SyncElasticcacheParameters %s", result.Result())

		// acl
		acls, err := iec.GetICloudElasticcacheAcls()
		if err != nil {
			return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetICloudElasticcacheAcls")
		}

		result = models.ElasticcacheAclManager.SyncElasticcacheAcls(ctx, userCred, ec, acls)
		log.Debugf("managedVirtualizationRegionDriver.RequestSyncElasticcache.SyncElasticcacheAcls %s", result.Result())

		// account
		accounts, err := iec.GetICloudElasticcacheAccounts()
		if err != nil {
			return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetICloudElasticcacheAccounts")
		}

		result = models.ElasticcacheAccountManager.SyncElasticcacheAccounts(ctx, userCred, ec, accounts)
		log.Debugf("managedVirtualizationRegionDriver.RequestSyncElasticcache.SyncElasticcacheAccounts %s", result.Result())

		// backups
		backups, err := iec.GetICloudElasticcacheBackups()
		if err != nil {
			return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetICloudElasticcacheBackups")
		}

		result = models.ElasticcacheBackupManager.SyncElasticcacheBackups(ctx, userCred, ec, backups)
		log.Debugf("managedVirtualizationRegionDriver.RequestSyncElasticcache.SyncElasticcacheBackups %s", result.Result())
	}

	return ec.SetStatus(userCred, api.ELASTIC_CACHE_STATUS_RUNNING, "")
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcache.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcache.GetIElasticcacheById")
	}

	err = iec.Delete()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcache.Delete")
	}

	return cloudprovider.WaitDeleted(iec, 10*time.Second, 300*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) RequestSetElasticcacheMaintainTime(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	mStart, err := task.GetParams().GetString("maintain_start_time")
	if err != nil {
		return errors.Wrap(fmt.Errorf("missing parameter maintain_start_time"), "managedVirtualizationRegionDriver.RequestSetElasticcacheMaintainTime")
	}

	mEnd, err := task.GetParams().GetString("maintain_end_time")
	if err != nil {
		return errors.Wrap(fmt.Errorf("missing parameter maintain_end_time"), "managedVirtualizationRegionDriver.RequestSetElasticcacheMaintainTime")
	}

	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSetElasticcacheMaintainTime.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSetElasticcacheMaintainTime.GetIElasticcacheById")
	}

	err = iec.SetMaintainTime(mStart, mEnd)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSetElasticcacheMaintainTime.SetMaintainTime")
	}

	return cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheChangeSpec(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	sku, err := task.GetParams().GetString("sku_ext_id")
	if err != nil {
		return errors.Wrap(fmt.Errorf("missing parameter sku"), "managedVirtualizationRegionDriver.RequestElasticcacheChangeSpec")
	}

	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheChangeSpec.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheChangeSpec.GetIElasticcacheById")
	}

	err = iec.ChangeInstanceSpec(sku)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheChangeSpec.ChangeInstanceSpec")
	}

	err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 1800*time.Second)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheChangeSpec.ChangeInstanceSpec")
	}

	err = ec.SyncWithCloudElasticcache(ctx, userCred, nil, iec)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheChangeSpec.SyncWithCloudElasticcache")
	}

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestUpdateElasticcacheAuthMode(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	authMode, err := task.GetParams().GetString("auth_mode")
	if err != nil {
		return errors.Wrap(fmt.Errorf("missing parameter auth_mode"), "managedVirtualizationRegionDriver.RequestUpdateElasticcacheAuthMode")
	}

	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestUpdateElasticcacheAuthMode.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestUpdateElasticcacheAuthMode.GetIElasticcacheById")
	}

	noPassword := true
	if authMode == "on" {
		noPassword = false
	}

	err = iec.UpdateAuthMode(noPassword)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestUpdateElasticcacheAuthMode.UpdateAuthMode")
	}

	_, err = db.Update(ec, func() error {
		ec.AuthMode = authMode
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestUpdateElasticcacheAuthMode.UpdatedbAuthMode")
	}

	return cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheSetMaintainTime(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	startTime, err := task.GetParams().GetString("maintain_start_time")
	if err != nil {
		return errors.Wrap(fmt.Errorf("missing parameter maintain_start_time"), "managedVirtualizationRegionDriver.RequestElasticcacheSetMaintainTime")
	}

	endTime, err := task.GetParams().GetString("maintain_end_time")
	if err != nil {
		return errors.Wrap(fmt.Errorf("missing parameter maintain_end_time"), "managedVirtualizationRegionDriver.RequestElasticcacheSetMaintainTime")
	}

	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheSetMaintainTime.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheSetMaintainTime.GetIElasticcacheById")
	}

	err = iec.SetMaintainTime(startTime, endTime)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheSetMaintainTime.SetMaintainTime")
	}

	// todo: sync instance spec
	return cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheAllocatePublicConnection(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	port, _ := task.GetParams().Int("port")
	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAllocatePublicConnection.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAllocatePublicConnection.GetIElasticcacheById")
	}

	_, err = iec.AllocatePublicConnection(int(port))
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAllocatePublicConnection.AllocatePublicConnection")
	}

	err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAllocatePublicConnection.WaitStatusWithDelay")
	}

	err = ec.SyncWithCloudElasticcache(ctx, task.GetUserCred(), nil, iec)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAllocatePublicConnection.SyncWithCloudElasticcache")
	}

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheReleasePublicConnection(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheReleasePublicConnection.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheReleasePublicConnection.GetIElasticcacheById")
	}

	err = iec.ReleasePublicConnection()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheReleasePublicConnection.AllocatePublicConnection")
	}

	err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
	if err != nil {
		return errors.Wrap(errors.ErrTimeout, "managedVirtualizationRegionDriver.RequestElasticcacheReleasePublicConnection.WaitStatusWithDelay")
	}

	err = ec.SyncWithCloudElasticcache(ctx, task.GetUserCred(), nil, iec)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAllocatePublicConnection.SyncWithCloudElasticcache")
	}

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheFlushInstance(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheFlushInstance.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheFlushInstance.GetIElasticcacheById")
	}

	err = iec.FlushInstance()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheFlushInstance.FlushInstance")
	}

	// todo: sync instance spec
	return cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheUpdateInstanceParameters(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	parameters, err := task.GetParams().Get("parameters")
	if err != nil {
		return errors.Wrap(fmt.Errorf("missing parameter parameters"), "managedVirtualizationRegionDriver.RequestElasticcacheUpdateInstanceParameters")
	}

	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheUpdateInstanceParameters.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheUpdateInstanceParameters.GetIElasticcacheById")
	}

	err = iec.UpdateInstanceParameters(parameters)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheUpdateInstanceParameters.UpdateInstanceParameters")
	}

	// todo: sync instance spec
	return cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheUpdateBackupPolicy(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	backupType, _ := task.GetParams().GetString("backup_type")
	backupReservedDays, _ := task.GetParams().Int("backup_reserved_days")
	preferredBackupPeriod, _ := task.GetParams().GetString("preferred_backup_period")
	preferredBackupTime, _ := task.GetParams().GetString("preferred_backup_time")

	config := cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput{
		BackupType:            backupType,
		BackupReservedDays:    int(backupReservedDays),
		PreferredBackupPeriod: preferredBackupPeriod,
		PreferredBackupTime:   preferredBackupTime,
	}

	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheUpdateBackupPolicy.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheUpdateBackupPolicy.GetIElasticcacheById")
	}

	err = iec.UpdateBackupPolicy(config)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheUpdateBackupPolicy.UpdateBackupPolicy")
	}

	// todo: sync instance spec
	return cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateElasticcacheAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateElasticcacheAclData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ips, err := data.GetString("ip_list")
	if err != nil || ips == "" {
		return nil, httperrors.NewMissingParameterError("ip_list")
	}

	ipV := validators.NewIPv4AddrValidator("ip")
	cidrV := validators.NewIPv4PrefixValidator("ip")
	_ips := strings.Split(ips, ",")
	for _, ip := range _ips {
		params := jsonutils.NewDict()
		params.Set("ip", jsonutils.NewString(ip))
		if strings.Contains(ip, "/") {
			if err := cidrV.Validate(params); err != nil {
				return nil, err
			}
		} else {
			if err := ipV.Validate(params); err != nil {
				return nil, err
			}
		}
	}

	elasticcacheV := validators.NewModelIdOrNameValidator("elasticcache", "elasticcache", ownerId)
	if err := elasticcacheV.Validate(data); err != nil {
		return nil, err
	}

	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) AllowCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, elasticcache *models.SElasticcache) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateElasticcacheBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	elasticcacheV := validators.NewModelIdOrNameValidator("elasticcache", "elasticcache", ownerId)
	if err := elasticcacheV.Validate(data); err != nil {
		return nil, err
	}

	ec := elasticcacheV.Model.(*models.SElasticcache)
	if !utils.IsInStringArray(ec.Status, []string{api.ELASTIC_CACHE_STATUS_RUNNING}) {
		return nil, httperrors.NewInputParameterError("can not make backup in status %s", ec.Status)
	}

	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheAccount *models.SElasticcacheAccount, task taskman.ITask) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
		if err != nil {
			return nil, errors.Wrap(nil, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.GetElasticcache")
		}

		ec := _ec.(*models.SElasticcache)
		iregion, err := ec.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(nil, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.GetIRegion")
		}

		iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.GetIElasticcacheById")
		}

		iea, err := iec.CreateAcl(ea.Name, ea.IpList)
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.CreateAcl")
		}

		// todo: wait elastic cache instance running
		ea.SetModelManager(models.ElasticcacheAclManager, ea)
		if err := db.SetExternalId(ea, userCred, iea.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.SetExternalId")
		}

		if err := ea.SyncWithCloudElasticcacheAcl(ctx, userCred, iea); err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.SyncWithCloudElasticcache")
		}

		return nil, nil
	})

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestChangeDBInstanceConfig(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		input := &api.SDBInstanceChangeConfigInput{}
		err := task.GetParams().Unmarshal(input)
		if err != nil {
			return nil, errors.Wrap(err, "task.GetParams().Unmarshal")
		}
		if len(input.StorageType) > 0 {
			instance.StorageType = input.StorageType
		}

		conf := &cloudprovider.SManagedDBInstanceChangeConfig{
			DiskSizeGB:  input.DiskSizeGB,
			StorageType: instance.StorageType,
		}

		instanceTypes := []string{}

		if len(input.InstanceType) > 0 {
			conf.InstanceType = input.InstanceType
		} else if input.VCpuCount == 0 && input.VmemSizeMb == 0 {
			conf.InstanceType = instance.InstanceType
		} else {
			instance.InstanceType = ""
			if input.VCpuCount > 0 {
				instance.VcpuCount = input.VCpuCount
			}
			if input.VmemSizeMb > 0 {
				instance.VmemSizeMb = input.VmemSizeMb
			}

			skus, err := instance.GetDBInstanceSkus()
			if err != nil {
				return nil, errors.Wrap(err, "instance.GetDBInstanceSkus")
			}
			for _, sku := range skus {
				instanceTypes = append(instanceTypes, sku.Name)
			}
		}

		if len(conf.InstanceType) == 0 && len(instanceTypes) == 0 {
			return nil, fmt.Errorf("No available dbinstance sku for change config")
		}

		iRds, err := instance.GetIDBInstance()
		if err != nil {
			return nil, errors.Wrap(err, "instance.GetIDBInstance")
		}

		log.Infof("change config: %s", jsonutils.Marshal(conf).String())

		if len(conf.InstanceType) > 0 {
			err = iRds.ChangeConfig(ctx, conf)
			if err != nil {
				return nil, errors.Wrapf(err, "iRds.ChangeConfig(%s)", conf.InstanceType)
			}
		} else {
			for _, instanceType := range instanceTypes {
				conf.InstanceType = instanceType
				log.Infof("try change instance type to %s", instance.InstanceType)
				err = iRds.ChangeConfig(ctx, conf)
				if err != nil {
					log.Warningf("change failed: %v try another", err)
				}
			}
			return nil, fmt.Errorf("no available dbinstance sku to change")
		}

		err = cloudprovider.WaitStatus(iRds, api.DBINSTANCE_RUNNING, time.Second*10, time.Minute*40)
		if err != nil {
			log.Errorf("failed to wait rds %s(%s) status running", instance.Name, instance.Id)
		}
		_, err = db.Update(instance, func() error {
			instance.InstanceType = iRds.GetInstanceType()
			instance.Category = iRds.GetCategory()
			instance.VcpuCount = iRds.GetVcpuCount()
			instance.VmemSizeMb = iRds.GetVmemSizeMB()
			instance.StorageType = iRds.GetStorageType()
			instance.DiskSizeGB = iRds.GetDiskSizeGB()
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "db.Update(instance)")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateDBInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, backup *models.SDBInstanceBackup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRds, err := instance.GetIDBInstance()
		if err != nil {
			return nil, errors.Wrap(err, "instance.GetIDBInstance")
		}

		iRegion, err := backup.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "backup.GetIRegion")
		}

		desc := &cloudprovider.SDBInstanceBackupCreateConfig{
			Name: backup.Name,
		}

		if len(backup.DBNames) > 0 {
			desc.Databases = strings.Split(backup.DBNames, ",")
		}

		backupId, err := iRds.CreateIBackup(desc)
		if err != nil {
			return nil, errors.Wrap(err, "iRds.CreateBackup")
		}

		db.SetExternalId(backup, userCred, backupId)

		iBackup, err := iRegion.GetIDBInstanceBackupById(backupId)
		if err != nil {
			return nil, errors.Wrapf(err, "iRegion.GetIDBInstanceBackupById(%s)", backupId)
		}

		_, err = db.Update(backup, func() error {
			backup.StartTime = iBackup.GetStartTime()
			backup.EndTime = iBackup.GetEndTime()
			backup.BackupSizeMb = iBackup.GetBackupSizeMb()
			return nil
		})

		return nil, err
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateChangeDBInstanceConfigData(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, input *api.SDBInstanceChangeConfigInput) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateResetDBInstancePassword(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, account string) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, eb *models.SElasticcacheBackup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_ec, err := db.FetchById(models.ElasticcacheManager, eb.ElasticcacheId)
		if err != nil {
			return nil, errors.Wrap(nil, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.GetElasticcache")
		}

		ec := _ec.(*models.SElasticcache)
		iregion, err := ec.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(nil, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.GetIRegion")
		}

		iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.GetIElasticcacheById")
		}

		ieb, err := iec.CreateBackup()
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.CreateBackup")
		}

		err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 30*time.Second, 30*time.Second, 1800*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.WaitStatusWithDelay")
		}

		eb.SetModelManager(models.ElasticcacheBackupManager, eb)
		if err := db.SetExternalId(eb, userCred, ieb.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.SetExternalId")
		}

		if err := eb.SyncWithCloudElasticcacheBackup(ctx, userCred, ieb); err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheBackup.SyncWithCloudElasticcacheBackup")
		}

		return nil, nil
	})

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	iregion, err := ea.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAccount.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAccount.FetchById")
	}

	ec := _ec.(*models.SElasticcache)
	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if err == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAccount.GetIElasticcacheById")
	}

	iea, err := iec.GetICloudElasticcacheAccount(ea.GetExternalId())
	if err == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAccount.GetICloudElasticcacheAccount")
	}

	err = iea.Delete()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAccount.Delete")
	}

	err = cloudprovider.WaitDeleted(iea, 10*time.Second, 300*time.Second)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAccount.WaitDeleted")
	}

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAcl, task taskman.ITask) error {
	iregion, err := ea.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAcl.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAcl.FetchElasticcacheById")
	}

	ec := _ec.(*models.SElasticcache)

	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if err == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAcl.GetIElasticcacheById")
	}

	iea, err := iec.GetICloudElasticcacheAcl(ea.GetExternalId())
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			return nil
		}

		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAcl.GetICloudElasticcacheAccount")
	}

	err = iea.Delete()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAcl.Delete")
	}

	return cloudprovider.WaitDeleted(iea, 10*time.Second, 300*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, eb *models.SElasticcacheBackup, task taskman.ITask) error {
	iregion, err := eb.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheBackup.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheAclManager, eb.ElasticcacheId)
	if err != nil {
		return err
	}

	ec := _ec.(*models.SElasticcache)
	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if err == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheBackup.GetIElasticcacheById")
	}

	ieb, err := iec.GetICloudElasticcacheBackup(eb.GetExternalId())
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheBackup.GetICloudElasticcacheBackup")
	}

	err = ieb.Delete()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheBackup.Delete")
	}

	return cloudprovider.WaitDeleted(ieb, 10*time.Second, 300*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheAccountResetPassword(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAccount, task taskman.ITask) error {
	iregion, err := ea.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAccountResetPassword.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAccountResetPassword.FetchById")
	}

	ec := _ec.(*models.SElasticcache)
	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if err == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAccountResetPassword.GetIElasticcacheById")
	}

	iea, err := iec.GetICloudElasticcacheAccount(ea.GetExternalId())
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAccountResetPassword.GetICloudElasticcacheBackup")
	}

	input := cloudprovider.SCloudElasticCacheAccountUpdateInput{}
	passwd, _ := task.GetParams().GetString("password")
	input.Password = &passwd

	err = iea.UpdateAccount(input)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAccountResetPassword.UpdateAccount")
	}

	err = ea.SavePassword(passwd)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheAccountResetPassword.SavePassword")
	}

	return ea.SetStatus(userCred, api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE, "")
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheAclUpdate(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
		if err != nil {
			return nil, errors.Wrap(nil, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.GetElasticcache")
		}

		ec := _ec.(*models.SElasticcache)
		iregion, err := ec.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(nil, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.GetIRegion")
		}

		iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.GetIElasticcacheById")
		}

		iea, err := iec.GetICloudElasticcacheAcl(ea.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.GetICloudElasticcacheAcl")
		}

		ipList, _ := task.GetParams().GetString("ip_list")
		err = iea.UpdateAcl(ipList)
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.UpdateAcl")
		}

		err = ea.SetStatus(userCred, api.ELASTIC_CACHE_ACL_STATUS_AVAILABLE, "")
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.UpdateAclStatus")
		}
		return nil, nil
	})

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheBackupRestoreInstance(ctx context.Context, userCred mcclient.TokenCredential, eb *models.SElasticcacheBackup, task taskman.ITask) error {
	iregion, err := eb.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheBackupRestoreInstance.GetIRegion")
	}

	_ec, err := db.FetchById(models.ElasticcacheManager, eb.ElasticcacheId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheBackupRestoreInstance.FetchById")
	}

	ec := _ec.(*models.SElasticcache)
	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if err == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheBackupRestoreInstance.GetIElasticcacheById")
	}

	ieb, err := iec.GetICloudElasticcacheBackup(eb.GetExternalId())
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheBackupRestoreInstance.GetICloudElasticcacheBackup")
	}

	err = ieb.RestoreInstance(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheBackupRestoreInstance.RestoreInstance")
	}

	_, err = db.Update(ec, func() error {
		ec.Status = api.ELASTIC_CACHE_STATUS_BACKUPRECOVERING
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheBackupRestoreInstance.UpdateStatus")
	}

	err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 30*time.Second, 30*time.Second, 1800*time.Second)
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheBackupRestoreInstance.WaitStatusWithDelay")
	}

	_, err = db.Update(ec, func() error {
		ec.Status = api.ELASTIC_CACHE_STATUS_RUNNING
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestElasticcacheBackupRestoreInstance.UpdateStatus")
	}

	return nil
}

func (self *SManagedVirtualizationRegionDriver) AllowUpdateElasticcacheAuthMode(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, elasticcache *models.SElasticcache) error {
	return nil
}
