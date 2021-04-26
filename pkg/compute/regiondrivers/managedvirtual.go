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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
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
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SManagedVirtualizationRegionDriver struct {
	SVirtualizationRegionDriver
}

func (self *SManagedVirtualizationRegionDriver) IsSupportedElasticcacheSecgroup() bool {
	return false
}

func (self *SManagedVirtualizationRegionDriver) GetMaxElasticcacheSecurityGroupCount() int {
	return 0
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, owerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
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

func (self *SManagedVirtualizationRegionDriver) IsSupportLoadbalancerListenerRuleRedirect() bool {
	return false
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	return data, nil
}

func validateUniqueById(ctx context.Context, userCred mcclient.TokenCredential, man db.IResourceModelManager, id string) error {
	q := man.Query().Equals("id", id)
	q = man.FilterByOwner(q, userCred, man.NamespaceScope())
	count, err := q.CountWithError()
	if err != nil {
		if err == sql.ErrNoRows {
			return httperrors.NewResourceNotFoundError("failed to find %s %s", man.Keyword(), id)
		}
		return httperrors.NewGeneralError(err)
	}

	if count > 1 {
		return httperrors.NewDuplicateResourceError(id)
	}

	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, lb *models.SLoadbalancer, backendGroup db.IModel) (*jsonutils.JSONDict, error) {
	if aclStatus, _ := data.GetString("acl_status"); aclStatus == api.LB_BOOL_ON {
		aclId, _ := data.GetString("acl_id")
		if len(aclId) == 0 {
			return nil, httperrors.NewMissingParameterError("acl")
		}

		err := validateUniqueById(ctx, userCred, models.LoadbalancerAclManager, aclId)
		if err != nil {
			return nil, err
		}
	}

	if lt, _ := data.GetString("listener_type"); lt == api.LB_LISTENER_TYPE_HTTPS {
		certId, _ := data.GetString("certificate_id")
		if len(certId) == 0 {
			return nil, httperrors.NewMissingParameterError("certificate_id")
		}

		err := validateUniqueById(ctx, userCred, models.LoadbalancerCertificateManager, certId)
		if err != nil {
			return nil, err
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

	aclId, _ := data.GetString("acl_id")
	if len(aclId) > 0 && lblis.AclId != aclId {
		err := validateUniqueById(ctx, userCred, models.LoadbalancerAclManager, aclId)
		if err != nil {
			return nil, err
		}
	}

	if certId, _ := data.GetString("certificate_id"); len(certId) > 0 {
		err := validateUniqueById(ctx, userCred, models.LoadbalancerCertificateManager, certId)
		if err != nil {
			return nil, err
		}
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

		_cloudprovider := lb.GetCloudprovider()
		params.ProjectId, err = _cloudprovider.SyncProject(ctx, userCred, lb.ProjectId)
		if err != nil {
			log.Errorf("failed to sync project %s for create %s lb %s error: %v", lb.ProjectId, _cloudprovider.Provider, lb.Name, err)
		}

		iLoadbalancer, err := iRegion.CreateILoadBalancer(params)
		if err != nil {
			return nil, err
		}
		if err := db.SetExternalId(lb, userCred, iLoadbalancer.GetGlobalId()); err != nil {
			return nil, err
		}
		if err := lb.SyncWithCloudLoadbalancer(ctx, userCred, iLoadbalancer, nil, lb.GetCloudprovider(), lb.GetRegion()); err != nil {
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
		models.SyncVirtualResourceMetadata(ctx, userCred, lb, iLoadbalancer)
		status := iLoadbalancer.GetStatus()
		if utils.IsInStringArray(status, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_DISABLED}) {
			return nil, lb.SetStatus(userCred, status, "")
		}
		return nil, fmt.Errorf("Unknown loadbalancer status %s", status)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRemoteUpdateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lb.GetIRegion()
		if err != nil {
			return nil, err
		}
		iLoadbalancer, err := iRegion.GetILoadBalancerById(lb.ExternalId)
		if err != nil {
			return nil, err
		}
		oldTags, err := iLoadbalancer.GetTags()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			return nil, errors.Wrap(err, "iLoadbalancer.GetTags()")
		}
		tags, err := lb.GetAllUserMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "lb.GetAllUserMetadata")
		}
		tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
		err = cloudprovider.SetTags(ctx, iLoadbalancer, lb.ManagerId, tags, replaceTags)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			logclient.AddActionLogWithStartable(task, lb, logclient.ACT_UPDATE_TAGS, err, userCred, false)
			return nil, errors.Wrap(err, "iLoadbalancer.SetMetadata")
		}
		logclient.AddActionLogWithStartable(task, lb, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, userCred, true)
		return nil, nil
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
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iLoadbalancer.Delete(ctx)
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
		if errors.Cause(err) == cloudprovider.ErrNotFound {
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
			if errors.Cause(err) == cloudprovider.ErrNotFound {
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
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}

		if len(lbbg.ExternalId) == 0 {
			return nil, nil
		}

		iLoadbalancerBackendGroup, err := iLoadbalancer.GetILoadBalancerBackendGroupById(lbbg.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}

		err = iLoadbalancerBackendGroup.Delete(ctx)
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
		return nil, lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iLoadbalancerBackend, lbbg.GetOwnerId(), lb.GetCloudprovider())
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
			if errors.Cause(err) == cloudprovider.ErrNotFound {
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

		err = iBackend.SyncConf(ctx, lbb.Port, lbb.Weight)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerBackend.SyncConf")
		}

		iBackend, err = iLoadbalancerBackendGroup.GetILoadbalancerBackendById(lbb.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerBackend.GetILoadbalancerBackendById")
		}

		return nil, lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iBackend, lbbg.GetOwnerId(), lb.GetCloudprovider())
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		{
			certId, _ := task.GetParams().GetString("certificate_id")
			if len(certId) > 0 {
				provider := lblis.GetCloudprovider()
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
				provider := lblis.GetCloudprovider()
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
		iListener, err := iLoadbalancer.CreateILoadBalancerListener(ctx, params)
		if err != nil {
			return nil, errors.Wrap(err, "iLoadbalancer.CreateILoadBalancerListener")
		}
		if err := db.SetExternalId(lblis, userCred, iListener.GetGlobalId()); err != nil {
			return nil, errors.Wrap(err, "db.SetExternalId")
		}
		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, loadbalancer.GetOwnerId(), lblis.GetCloudprovider())
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
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrap(err, "RegionDriver.RequestDeleteLoadbalancerListener.GetILoadBalancerById")
		}

		iListener, err := iLoadbalancer.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrap(err, "RegionDriver.RequestDeleteLoadbalancerListener.GetILoadBalancerListenerById")
		}

		return nil, iListener.Delete(ctx)
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
				provider := lblis.GetCloudprovider()
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
				provider := lblis.GetCloudprovider()
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
		if err := iListener.Sync(ctx, params); err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.SyncListener")
		}
		if err := iListener.Refresh(); err != nil {
			return nil, errors.Wrap(err, "regionDriver.RequestSyncLoadbalancerListener.RefreshListener")
		}
		return nil, lblis.SyncWithCloudLoadbalancerListener(ctx, userCred, loadbalancer, iListener, loadbalancer.GetOwnerId(), lblis.GetCloudprovider())
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
		return nil, lbr.SyncWithCloudLoadbalancerListenerRule(ctx, userCred, iListenerRule, listener.GetOwnerId(), loadbalancer.GetCloudprovider())
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
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iListenerRule.Delete(ctx)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error) {
	return input, nil
}
func (self *SManagedVirtualizationRegionDriver) GetEipDefaultChargeType() string {
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}
func (self *SManagedVirtualizationRegionDriver) ValidateEipChargeType(chargeType string) error {
	return nil
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

		if ivpc.IsSupportSetExternalAccess() && vpc.ExternalAccessMode == api.VPC_EXTERNAL_ACCESS_MODE_EIP {
			igw, err := iregion.CreateInternetGateway()
			if err != nil {
				return nil, errors.Wrap(err, "vpc.AttachInternetGateway")

			}

			err = ivpc.AttachInternetGateway(igw.GetId())
			if err != nil {
				return nil, errors.Wrap(err, "vpc.AttachInternetGateway")
			}
		}

		err = vpc.SyncRemoteWires(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncRemoteWires")
		}

		err = vpc.SyncWithCloudVpc(ctx, userCred, ivpc, nil)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncWithCloudVpc")
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
		if errors.Cause(err) == cloudprovider.ErrNotFound {
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
			if errors.Cause(err) == cloudprovider.ErrNotFound {
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

func (self *SManagedVirtualizationRegionDriver) RequestCreateInstanceSnapshot(ctx context.Context, guest *models.SGuest, isp *models.SInstanceSnapshot, task taskman.ITask, params *jsonutils.JSONDict) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteInstanceSnapshot(ctx context.Context, isp *models.SInstanceSnapshot, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestResetToInstanceSnapshot(ctx context.Context, guest *models.SGuest, isp *models.SInstanceSnapshot, task taskman.ITask, params *jsonutils.JSONDict) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return nil, nil
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

func (self *SManagedVirtualizationRegionDriver) RequestAssociateEipForNAT(ctx context.Context, userCred mcclient.TokenCredential, nat *models.SNatGateway, eip *models.SElasticip, task taskman.ITask) error {
	opts := api.ElasticipAssociateInput{
		InstanceType: api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY,
		InstanceId:   nat.Id,
	}
	return eip.StartEipAssociateTask(ctx, userCred, jsonutils.Marshal(opts).(*jsonutils.JSONDict), task.GetTaskId())
}

func (self *SManagedVirtualizationRegionDriver) RequestPreSnapshotPolicyApply(ctx context.Context, userCred mcclient.
	TokenCredential, task taskman.ITask, disk *models.SDisk, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) error {

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		return data, nil
	})
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

func (self *SManagedVirtualizationRegionDriver) RequestSyncSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, vpcId string, vpc *models.SVpc, secgroup *models.SSecurityGroup, remoteProjectId, service string) (string, error) {
	lockman.LockRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-%s-%s", secgroup.Id, vpcId, vpc.ManagerId))
	defer lockman.ReleaseRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-%s-%s", secgroup.Id, vpcId, vpc.ManagerId))

	region, err := vpc.GetRegion()
	if err != nil {
		return "", errors.Wrap(err, "vpc.GetRegon")
	}

	if region.GetDriver().GetSecurityGroupPublicScope(service) == rbacutils.ScopeSystem {
		remoteProjectId = ""
	}

	cache, err := models.SecurityGroupCacheManager.Register(ctx, userCred, secgroup.Id, vpcId, region.Id, vpc.ManagerId, remoteProjectId)
	if err != nil {
		return "", errors.Wrap(err, "SSecurityGroupCache.Register")
	}

	return cache.ExternalId, cache.SyncRules()
}

func (self *SManagedVirtualizationRegionDriver) RequestCacheSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, secgroup *models.SSecurityGroup, classic bool, removeProjectId string, task taskman.ITask) error {
	vpcId, err := region.GetDriver().GetSecurityGroupVpcId(ctx, userCred, region, nil, vpc, classic)
	if err != nil {
		return errors.Wrap(err, "GetSecurityGroupVpcId")
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_, err := self.RequestSyncSecurityGroup(ctx, userCred, vpcId, vpc, secgroup, removeProjectId, "")
		return nil, err
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *models.SDBInstance, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iregion, err := dbinstance.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "GetIRegionAndProvider")
		}

		vpc, err := dbinstance.GetVpc()
		if err != nil {
			return nil, errors.Wrap(err, "dbinstance.GetVpc()")
		}

		params := task.GetParams()
		passwd, _ := params.GetString("password")
		if len(passwd) == 0 && jsonutils.QueryBoolean(params, "reset_password", true) {
			passwd = seclib2.RandomPassword2(12)
		}
		desc := cloudprovider.SManagedDBInstanceCreateConfig{
			Name:          dbinstance.Name,
			Description:   dbinstance.Description,
			StorageType:   dbinstance.StorageType,
			DiskSizeGB:    dbinstance.DiskSizeGB,
			VcpuCount:     dbinstance.VcpuCount,
			VmemSizeMb:    dbinstance.VmemSizeMb,
			VpcId:         vpc.ExternalId,
			Engine:        dbinstance.Engine,
			EngineVersion: dbinstance.EngineVersion,
			Category:      dbinstance.Category,
			Port:          dbinstance.Port,
			Password:      passwd,
		}
		desc.Tags, _ = dbinstance.GetAllUserMetadata()

		networks, err := dbinstance.GetDBNetworks()
		if err != nil {
			return nil, errors.Wrapf(err, "dbinstance.GetDBNetworks")
		}

		if len(networks) > 0 {
			net, err := networks[0].GetNetwork()
			if err != nil {
				return nil, errors.Wrapf(err, "GetNetwork")
			}
			desc.NetworkId, desc.Address = net.ExternalId, networks[0].IpAddr
		}

		_cloudprovider := dbinstance.GetCloudprovider()
		desc.ProjectId, err = _cloudprovider.SyncProject(ctx, userCred, dbinstance.ProjectId)
		if err != nil {
			log.Errorf("failed to sync project %s for create %s rds %s error: %v", dbinstance.ProjectId, _cloudprovider.Provider, dbinstance.Name, err)
		}

		region := dbinstance.GetRegion()

		err = region.GetDriver().InitDBInstanceUser(ctx, dbinstance, task, &desc)
		if err != nil {
			return nil, err
		}

		secgroups, err := dbinstance.GetSecgroups()
		if err != nil {
			return nil, errors.Wrapf(err, "GetSecgroups")
		}
		for i := range secgroups {
			vpcId, err := region.GetDriver().GetSecurityGroupVpcId(ctx, userCred, region, nil, vpc, false)
			if err != nil {
				return nil, errors.Wrap(err, "GetSecurityGroupVpcId")
			}
			secId, err := region.GetDriver().RequestSyncSecurityGroup(ctx, userCred, vpcId, vpc, &secgroups[i], desc.ProjectId, "")
			if err != nil {
				return nil, errors.Wrap(err, "SyncSecurityGroup")
			}
			desc.SecgroupIds = append(desc.SecgroupIds, secId)
		}

		if dbinstance.BillingType == billing_api.BILLING_TYPE_PREPAID {
			bc, err := billing.ParseBillingCycle(dbinstance.BillingCycle)
			if err != nil {
				log.Errorf("failed to parse billing cycle %s: %v", dbinstance.BillingCycle, err)
			} else if bc.IsValid() {
				desc.BillingCycle = &bc
				desc.BillingCycle.AutoRenew = dbinstance.AutoRenew
			}
		}

		if len(dbinstance.MasterInstanceId) > 0 {
			master, err := dbinstance.GetMasterInstance()
			if err != nil {
				return nil, errors.Wrap(err, "dbinstnace.GetMasterInstance()")
			}
			desc.MasterInstanceId = master.ExternalId
		}

		instanceTypes, err := dbinstance.GetAvailableInstanceTypes()
		if err != nil {
			return nil, errors.Wrapf(err, "GetAvailableInstanceTypes")
		}
		if len(instanceTypes) == 0 {
			return nil, fmt.Errorf("no avaiable sku for create")
		}

		var createFunc = func() (cloudprovider.ICloudDBInstance, error) {
			errMsgs := []string{}
			for i := range instanceTypes {
				desc.SInstanceType = instanceTypes[i]
				log.Debugf("create dbinstance params: %s", jsonutils.Marshal(desc).String())

				iRds, err := iregion.CreateIDBInstance(&desc)
				if err != nil {
					errMsgs = append(errMsgs, err.Error())
					continue
				}
				return iRds, nil
			}
			if len(errMsgs) > 0 {
				return nil, fmt.Errorf(strings.Join(errMsgs, "\n"))
			}
			return nil, fmt.Errorf("no avaiable skus %s(%dC%d) for create", dbinstance.InstanceType, desc.VcpuCount, desc.VmemSizeMb)
		}

		iRds, err := createFunc()
		if err != nil {
			return nil, errors.Wrapf(err, "create")
		}

		err = db.SetExternalId(dbinstance, userCred, iRds.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "db.SetExternalId")
		}

		err = cloudprovider.WaitStatus(iRds, api.DBINSTANCE_RUNNING, time.Second*10, time.Hour*1)
		if err != nil {
			return nil, errors.Wrapf(err, "cloudprovider.WaitStatus runing")
		}

		secgroupIds, err := iRds.GetSecurityGroupIds()
		if err == nil && len(secgroupIds) != len(desc.SecgroupIds) {
			err = iRds.SetSecurityGroups(desc.SecgroupIds)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				logclient.AddSimpleActionLog(dbinstance, logclient.ACT_SYNC_CONF, map[string][]string{"secgroup_ids": desc.SecgroupIds}, userCred, false)
			}
		}
		return nil, nil
	})

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateDBInstanceFromBackup(ctx context.Context, userCred mcclient.TokenCredential, rds *models.SDBInstance, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_backup, err := models.DBInstanceBackupManager.FetchById(rds.DBInstancebackupId)
		if err != nil {
			return nil, errors.Wrapf(err, "DBInstanceBackupManager.FetchById(%s)", rds.DBInstancebackupId)
		}
		backup := _backup.(*models.SDBInstanceBackup)
		iBackup, err := backup.GetIDBInstanceBackup()
		if err != nil {
			return nil, errors.Wrapf(err, "backup.GetIDBInstanceBackup")
		}
		vpc, err := rds.GetVpc()
		if err != nil {
			return nil, errors.Wrap(err, "rds.GetVpc()")
		}
		params := task.GetParams()
		passwd, _ := params.GetString("password")
		if len(passwd) == 0 && jsonutils.QueryBoolean(params, "reset_password", true) {
			passwd = seclib2.RandomPassword2(12)
		}
		desc := cloudprovider.SManagedDBInstanceCreateConfig{
			Name:          rds.Name,
			Description:   rds.Description,
			StorageType:   rds.StorageType,
			DiskSizeGB:    rds.DiskSizeGB,
			VcpuCount:     rds.VcpuCount,
			VmemSizeMb:    rds.VmemSizeMb,
			VpcId:         vpc.ExternalId,
			Engine:        rds.Engine,
			EngineVersion: rds.EngineVersion,
			Category:      rds.Category,
			Port:          rds.Port,
			Password:      passwd,
		}
		if len(backup.DBInstanceId) > 0 {
			parentRds, err := backup.GetDBInstance()
			if err != nil {
				return nil, errors.Wrapf(err, "backup.GetDBInstance")
			}
			desc.RdsId = parentRds.ExternalId
		}

		log.Debugf("create from backup params: %s", jsonutils.Marshal(desc).String())

		networks, err := rds.GetDBNetworks()
		if err != nil {
			return nil, errors.Wrapf(err, "dbinstance.GetDBNetworks")
		}

		if len(networks) > 0 {
			net, err := networks[0].GetNetwork()
			if err != nil {
				return nil, errors.Wrapf(err, "GetNetwork")
			}
			desc.NetworkId, desc.Address = net.ExternalId, networks[0].IpAddr
		}

		_cloudprovider := rds.GetCloudprovider()
		desc.ProjectId, err = _cloudprovider.SyncProject(ctx, userCred, rds.ProjectId)
		if err != nil {
			log.Errorf("failed to sync project %s for create %s rds %s error: %v", rds.ProjectId, _cloudprovider.Provider, rds.Name, err)
		}

		region := rds.GetRegion()

		err = region.GetDriver().InitDBInstanceUser(ctx, rds, task, &desc)
		if err != nil {
			return nil, err
		}

		secgroups, err := rds.GetSecgroups()
		if err != nil {
			return nil, errors.Wrapf(err, "GetSecgroups")
		}
		for i := range secgroups {
			vpcId, err := region.GetDriver().GetSecurityGroupVpcId(ctx, userCred, region, nil, vpc, false)
			if err != nil {
				return nil, errors.Wrap(err, "GetSecurityGroupVpcId")
			}
			secId, err := region.GetDriver().RequestSyncSecurityGroup(ctx, userCred, vpcId, vpc, &secgroups[i], desc.ProjectId, "")
			if err != nil {
				return nil, errors.Wrap(err, "SyncSecurityGroup")
			}
			desc.SecgroupIds = append(desc.SecgroupIds, secId)
		}

		if rds.BillingType == billing_api.BILLING_TYPE_PREPAID {
			bc, err := billing.ParseBillingCycle(rds.BillingCycle)
			if err != nil {
				log.Errorf("failed to parse billing cycle %s: %v", rds.BillingCycle, err)
			} else if bc.IsValid() {
				desc.BillingCycle = &bc
				desc.BillingCycle.AutoRenew = rds.AutoRenew
			}
		}

		instanceTypes, err := rds.GetAvailableInstanceTypes()
		if err != nil {
			return nil, errors.Wrapf(err, "GetAvailableInstanceTypes")
		}
		if len(instanceTypes) == 0 {
			return nil, fmt.Errorf("no avaiable sku for create")
		}

		var createFunc = func() (cloudprovider.ICloudDBInstance, error) {
			errMsgs := []string{}
			for i := range instanceTypes {
				desc.SInstanceType = instanceTypes[i]
				log.Debugf("create dbinstance params: %s", jsonutils.Marshal(desc).String())

				iRds, err := iBackup.CreateICloudDBInstance(&desc)
				if err != nil {
					errMsgs = append(errMsgs, err.Error())
					continue
				}
				return iRds, nil
			}
			if len(errMsgs) > 0 {
				return nil, fmt.Errorf(strings.Join(errMsgs, "\n"))
			}
			return nil, fmt.Errorf("no avaiable skus %s(%dC%d) for create", rds.InstanceType, desc.VcpuCount, desc.VmemSizeMb)
		}

		iRds, err := createFunc()
		if err != nil {
			return nil, errors.Wrapf(err, "create")
		}

		err = db.SetExternalId(rds, userCred, iRds.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "db.SetExternalId")
		}

		err = cloudprovider.WaitStatus(iRds, api.DBINSTANCE_RUNNING, time.Second*5, time.Hour*1)
		if err != nil {
			return nil, errors.Wrapf(err, "cloudprovider.WaitStatus runing")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask, data *jsonutils.JSONDict) error {
	task.ScheduleRun(nil)
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.ElasticcacheCreateInput) (*jsonutils.JSONDict, error) {
	m := jsonutils.NewDict()
	m.Set("manager_id", jsonutils.NewString(input.ManagerId))
	_, err := self.ValidateManagerId(ctx, userCred, m)
	if err != nil {
		return nil, errors.Wrap(err, "ValidateManagerId")
	}

	return input.JSON(input), nil
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

	err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 1800*time.Second)
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
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			ec.SetStatus(userCred, api.ELASTIC_CACHE_STATUS_UNKNOWN, "")
			return nil
		}

		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetIElasticcacheById")
	}

	provider := ec.GetCloudprovider()
	if provider == nil {
		return errors.Wrap(fmt.Errorf("provider is nil"), "managedVirtualizationRegionDriver.RequestSyncElasticcache.GetCloudprovider")
	}

	lockman.LockRawObject(ctx, "elastic-cache", ec.Id)
	defer lockman.ReleaseRawObject(ctx, "elastic-cache", ec.Id)

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

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcache.GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
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

	provider := ec.GetCloudprovider()
	if provider == nil {
		return errors.Wrap(fmt.Errorf("provider is nil"), "managedVirtualizationRegionDriver.RequestElasticcacheChangeSpec.GetCloudprovider")
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

	err = ec.SyncWithCloudElasticcache(ctx, userCred, provider, iec)
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

	err = iec.UpdateAuthMode(noPassword, "")
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

	return cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 600*time.Second)
}

func (self *SManagedVirtualizationRegionDriver) RequestUpdateElasticcacheSecgroups(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	// todo: finish me
	return nil
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

	password, _ := task.GetParams().GetString("password")
	input := cloudprovider.SCloudElasticCacheFlushInstanceInput{}
	if len(password) > 0 {
		input.Password = password
	} else {
		if info, err := ec.GetDetailsLoginInfo(ctx, userCred, jsonutils.NewDict()); err == nil && info != nil {
			pwd, _ := info.GetString("password")
			input.Password = pwd
		}
	}

	err = iec.FlushInstance(input)
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
	reservedNames := []string{
		"add", "admin", "all", "alter", "analyze", "and", "as", "asc",
		"asensitive", "aurora", "before", "between", "bigint", "binary",
		"blob", "both", "by", "call", "cascade", "case", "change",
		"char", "character", "check", "collate", "column", "condition",
		"connection", "constraint", "continue", "convert", "create",
		"cross", "current_date", "current_time", "current_timestamp",
		"current_user", "cursor", " database", "databases", "day_hour",
		"day_microsecond", "day_minute", "day_second", "dec", "decimal",
		"declare", "default", "delayed", "delete", "desc", "describe",
		"deterministic", "distinct", "distinctrow", "div", "double",
		"drc_rds", "drop", "dual", "each", "eagleye", "else", "elseif",
		"enclosed", "escaped", "exists", "exit", "explain", "false",
		"fetch", "float", "float4", "float8", "for", "force", "foreign",
		"from", "fulltext", " goto", "grant", "group", "guest", "having",
		"high_priority", "hour_microsecond", "hour_minute", "hour_second",
		"if", "ignore", "in", "index", "infile", "information_schema", "inner",
		"inout", "insensitive", "insert", "int", "int1", "int2", "int3", "int4",
		"int8", "integer", "interval", "into", "is", "iterate", "join", "key", "keys",
		"kill", "label", "leading", "leave", "left", "like", "limit", "linear", "lines",
		"load", "localtime", "localtimestamp", "lock", "long", "longblob", "longtext",
		"loop", "low_priority", " match", "mediumblob", "mediumint", "mediumtext",
		"middleint", "minute_microsecond", "minute_second", "mod", "modifies",
		"mysql", "natural", "no_write_to_binlog", "not", "null", "numeric",
		"on", "optimize", "option", "optionally", "or", "order", "out", "outer",
		"outfile", "precision", "primary", "procedure", "purge",
		"raid0", "range", "read", "reads", "real",
		"references", "regexp", "release", "rename", "repeat",
		"replace", "replicator", "require", "restrict", "return", "revoke", "right",
		"rlike", "root", "schema", "schemas", "second_microsecond", "select", "sensitive",
		"separator", "set", "show", "smallint", "spatial", "specific", "sql", "sql_big_result",
		"sql_calc_found_rows", "sql_small_result", "sqlexception", "sqlstate", "sqlwarning",
		"ssl", "starting", "straight_join", "table", "terminated", "test", "then", "tinyblob",
		"tinyint", "tinytext", "to", "trailing", "trigger", "true", "undo",
		"union", "unique", "unlock", "unsigned", "update", "usage", "use", "using",
		"utc_date", "utc_time", "utc_timestamp", "values", "varbinary", "varchar",
		"varcharacter", "varying", "when", "where", "while", "with", "write", "x509",
		"xor", "xtrabak", "year_month", "zerofill"}
	if name, _ := data.GetString("name"); utils.IsInStringArray(name, reservedNames) {
		return nil, httperrors.NewConflictError("account name '%s' is not allowed", name)
	}

	return data, nil
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

	data.Set("backup_mode", jsonutils.NewString(api.BACKUP_MODE_MANUAL))
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

func (self *SManagedVirtualizationRegionDriver) RequestChangeDBInstanceConfig(ctx context.Context, userCred mcclient.TokenCredential, rds *models.SDBInstance, input *api.SDBInstanceChangeConfigInput, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		conf := cloudprovider.SManagedDBInstanceChangeConfig{}

		if input.DiskSizeGB > 0 && input.DiskSizeGB != rds.DiskSizeGB {
			conf.DiskSizeGB = input.DiskSizeGB
		}

		if len(input.InstanceType) > 0 && input.InstanceType != rds.InstanceType {
			conf.InstanceType = input.InstanceType
		}

		iRds, err := rds.GetIDBInstance()
		if err != nil {
			return nil, errors.Wrap(err, "rds.GetIDBInstance")
		}

		log.Infof("change config: %s", jsonutils.Marshal(conf).String())
		err = iRds.ChangeConfig(ctx, &conf)
		if err != nil {
			return nil, errors.Wrapf(err, "iRds.ChangeConfig")
		}

		err = cloudprovider.WaitStatus(iRds, api.DBINSTANCE_RUNNING, time.Second*10, time.Minute*40)
		if err != nil {
			return nil, errors.Wrapf(err, "cloudprovider.WaitStatus")
		}

		err = iRds.Refresh()
		if err != nil {
			return nil, errors.Wrapf(err, "iRds.Refresh")
		}

		_, err = db.Update(rds, func() error {
			rds.InstanceType = iRds.GetInstanceType()
			rds.Category = iRds.GetCategory()
			rds.VcpuCount = iRds.GetVcpuCount()
			rds.VmemSizeMb = iRds.GetVmemSizeMB()
			rds.StorageType = iRds.GetStorageType()
			rds.DiskSizeGB = iRds.GetDiskSizeGB()
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

		err = db.SetExternalId(backup, userCred, backupId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.SetExternalId")
		}

		err = cloudprovider.Wait(time.Second*5, time.Minute*15, func() (bool, error) {
			iBackup, err := backup.GetIDBInstanceBackup()
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					log.Warningf("GetIDBInstanceBackup: %v", err)
					return false, nil
				}
				return false, errors.Wrapf(err, "GetIDBInstanceBackup")
			}

			err = backup.SyncWithCloudDBInstanceBackup(ctx, userCred, iBackup, instance.GetCloudprovider())
			if err != nil {
				log.Warningf("sync backup info error: %v", err)
			}

			return true, nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "cloudprovider.Wait backup sync")
		}

		instance.SetStatus(userCred, api.DBINSTANCE_RUNNING, "")
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateResetDBInstancePassword(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, account string) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRemoteUpdateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRds, err := instance.GetIDBInstance()
		if err != nil {
			return nil, errors.Wrap(err, "instance.GetIDBInstance")
		}
		oldTags, err := iRds.GetTags()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			return nil, errors.Wrap(err, "iRds.GetTags()")
		}
		tags, err := instance.GetAllUserMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "instance.GetAllUserMetadata")
		}
		tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
		err = cloudprovider.SetTags(ctx, iRds, instance.ManagerId, tags, replaceTags)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			logclient.AddActionLogWithStartable(task, instance, logclient.ACT_UPDATE_TAGS, err, userCred, false)
			return nil, errors.Wrap(err, "iRds.SetTags")
		}
		logclient.AddActionLogWithStartable(task, instance, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, userCred, true)
		return nil, nil
	})
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

		ieb, err := iec.CreateBackup(eb.Name)
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
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAccount.GetIElasticcacheById")
	}

	iea, err := iec.GetICloudElasticcacheAccount(ea.GetExternalId())
	if errors.Cause(err) == cloudprovider.ErrNotFound {
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
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "managedVirtualizationRegionDriver.RequestDeleteElasticcacheAcl.GetIElasticcacheById")
	}

	iea, err := iec.GetICloudElasticcacheAcl(ea.GetExternalId())
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
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
	if errors.Cause(err) == cloudprovider.ErrNotFound {
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
	if errors.Cause(err) == cloudprovider.ErrNotFound {
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
	if errors.Cause(err) == cloudprovider.ErrNotFound {
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

func (self *SManagedVirtualizationRegionDriver) RequestSyncDiskStatus(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iDisk, err := disk.GetIDisk()
		if err != nil {
			return nil, errors.Wrap(err, "disk.GetIDisk")
		}

		return nil, disk.SetStatus(userCred, iDisk.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncSnapshotStatus(ctx context.Context, userCred mcclient.TokenCredential, snapshot *models.SSnapshot, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := snapshot.GetISnapshotRegion()
		if err != nil {
			return nil, errors.Wrap(err, "snapshot.GetISnapshotRegion")
		}

		iSnapshot, err := iRegion.GetISnapshotById(snapshot.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "iRegion.GetISnapshotById(%s)", snapshot.ExternalId)
		}

		return nil, snapshot.SetStatus(userCred, iSnapshot.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncNatGatewayStatus(ctx context.Context, userCred mcclient.TokenCredential, nat *models.SNatGateway, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		iNat, err := nat.GetINatGateway()
		if err != nil {
			return nil, errors.Wrap(err, "nat.GetINatGateway")
		}

		return nil, nat.SyncWithCloudNatGateway(ctx, userCred, nat.GetCloudprovider(), iNat)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncBucketStatus(ctx context.Context, userCred mcclient.TokenCredential, bucket *models.SBucket, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iBucket, err := bucket.GetIBucket()
		if err != nil {
			return nil, errors.Wrap(err, "bucket.GetIBucket")
		}

		return nil, bucket.SetStatus(userCred, iBucket.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncDBInstanceBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, backup *models.SDBInstanceBackup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iDBInstanceBackup, err := backup.GetIDBInstanceBackup()
		if err != nil {
			return nil, errors.Wrap(err, "backup.GetIDBInstanceBackup")
		}

		return nil, backup.SetStatus(userCred, iDBInstanceBackup.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncElasticcacheStatus(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := elasticcache.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "elasticcache.GetIRegion")
		}

		iElasticcache, err := iRegion.GetIElasticcacheById(elasticcache.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "elasticcache.GetIElasticcache")
		}
		models.SyncVirtualResourceMetadata(ctx, userCred, elasticcache, iElasticcache)
		return nil, elasticcache.SetStatus(userCred, iElasticcache.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRemoteUpdateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := elasticcache.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "elasticcache.GetIRegion")
		}

		iElasticcache, err := iRegion.GetIElasticcacheById(elasticcache.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIElasticcacheById(%s)", elasticcache.ExternalId)
		}

		oldTags, err := iElasticcache.GetTags()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			return nil, errors.Wrap(err, "iElasticcache.GetTags()")
		}
		tags, err := elasticcache.GetAllUserMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "GetAllUserMetadata")
		}
		tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
		mangerId := ""
		if vpc := elasticcache.GetVpc(); vpc != nil {
			mangerId = vpc.ManagerId
		}
		err = cloudprovider.SetTags(ctx, iElasticcache, mangerId, tags, replaceTags)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}

			logclient.AddActionLogWithStartable(task, elasticcache, logclient.ACT_UPDATE_TAGS, err, userCred, false)
			return nil, errors.Wrap(err, "iElasticcache.SetTags")
		}
		logclient.AddActionLogWithStartable(task, elasticcache, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, userCred, true)
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncSecgroupsForElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncSecgroupsForElasticcache")
}

func (self *SManagedVirtualizationRegionDriver) RequestRenewElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, bc billing.SBillingCycle) (time.Time, error) {
	iregion, err := ec.GetIRegion()
	if err != nil {
		return time.Time{}, errors.Wrap(err, "GetIRegion")
	}

	if len(ec.GetExternalId()) == 0 {
		return time.Time{}, errors.Wrap(err, "ExternalId is empty")
	}

	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if err != nil {
		return time.Time{}, errors.Wrap(err, "GetIElasticcacheById")
	}

	oldExpired := iec.GetExpiredAt()
	err = iec.Renew(bc)
	if err != nil {
		return time.Time{}, err
	}
	//避免有些云续费后过期时间刷新比较慢问题
	cloudprovider.WaitCreated(15*time.Second, 5*time.Minute, func() bool {
		err := iec.Refresh()
		if err != nil {
			log.Errorf("failed refresh instance %s error: %v", ec.Name, err)
		}
		newExipred := iec.GetExpiredAt()
		if newExipred.After(oldExpired) {
			return true
		}
		return false
	})
	return iec.GetExpiredAt(), nil
}

func (self *SManagedVirtualizationRegionDriver) IsSupportedElasticcacheAutoRenew() bool {
	return true
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheSetAutoRenew(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, autoRenew bool, task taskman.ITask) error {
	iregion, err := ec.GetIRegion()
	if err != nil {
		return errors.Wrap(err, "GetIRegion")
	}

	if len(ec.GetExternalId()) == 0 {
		return errors.Wrap(err, "ExternalId is empty")
	}

	iec, err := iregion.GetIElasticcacheById(ec.GetExternalId())
	if err != nil {
		return errors.Wrap(err, "GetIElasticcacheById")
	}

	err = iec.SetAutoRenew(autoRenew)
	if err != nil {
		return errors.Wrap(err, "SetAutoRenew")
	}

	return ec.SetAutoRenew(autoRenew)
}

func IsInPrivateIpRange(ar netutils.IPV4AddrRange) error {
	iprs := netutils.GetPrivateIPRanges()
	match := false
	for _, ipr := range iprs {
		if ipr.ContainsRange(ar) {
			match = true
			break
		}
	}

	if !match {
		return httperrors.NewInputParameterError("invalid cidr range %s", ar.String())
	}

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncRdsSecurityGroups(ctx context.Context, userCred mcclient.TokenCredential, rds *models.SDBInstance, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		vpc, err := rds.GetVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "rds.GetVpc")
		}
		secgroups, err := rds.GetSecgroups()
		if err != nil {
			return nil, errors.Wrapf(err, "GetSecgroups")
		}
		iRds, err := rds.GetIDBInstance()
		if err != nil {
			return nil, errors.Wrapf(err, "GetIDBInstance")
		}
		secgroupIds := []string{}
		for i := range secgroups {
			secgroupId, err := self.RequestSyncSecurityGroup(ctx, userCred, vpc.ExternalId, vpc, &secgroups[i], iRds.GetProjectId(), "rds")
			if err != nil {
				return nil, errors.Wrapf(err, "RequestSyncSecurityGroup")
			}
			secgroupIds = append(secgroupIds, secgroupId)
		}
		err = iRds.SetSecurityGroups(secgroupIds)
		if err != nil {
			return nil, errors.Wrapf(err, "SetSecurityGroups")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestAssociatEip(ctx context.Context, userCred mcclient.TokenCredential, eip *models.SElasticip, input api.ElasticipAssociateInput, obj db.IStatusStandaloneModel, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iEip, err := eip.GetIEip()
		if err != nil {
			return nil, errors.Wrapf(err, "eip.GetIEip")
		}

		conf := &cloudprovider.AssociateConfig{
			InstanceId:    input.InstanceExternalId,
			Bandwidth:     eip.Bandwidth,
			AssociateType: input.InstanceType,
			ChargeType:    eip.ChargeType,
		}

		err = iEip.Associate(conf)
		if err != nil {
			return nil, errors.Wrapf(err, "iEip.Associate")
		}

		err = cloudprovider.WaitStatus(iEip, api.EIP_STATUS_READY, 3*time.Second, 60*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "cloudprovider.WaitStatus")
		}

		if obj.GetStatus() != api.INSTANCE_ASSOCIATE_EIP {
			db.StatusBaseSetStatus(obj, userCred, api.INSTANCE_ASSOCIATE_EIP, "associate eip")
		}

		err = eip.AssociateInstance(ctx, userCred, input.InstanceType, obj)
		if err != nil {
			return nil, errors.Wrapf(err, "eip.AssociateVM")
		}

		eip.SetStatus(userCred, api.EIP_STATUS_READY, api.EIP_STATUS_ASSOCIATE)
		return nil, nil
	})
	return nil
}
