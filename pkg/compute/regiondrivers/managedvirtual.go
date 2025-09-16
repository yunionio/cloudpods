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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/sshkeys"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
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

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, owerId mcclient.IIdentityProvider, input *api.LoadbalancerCreateInput) (*api.LoadbalancerCreateInput, error) {
	if len(input.ManagerId) == 0 {
		return nil, httperrors.NewMissingParameterError("manager_id")
	}
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateManagerId(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if managerId, _ := data.GetString("manager_id"); len(managerId) == 0 {
		return nil, httperrors.NewMissingParameterError("manager")
	}
	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.ValidateManagerId(ctx, userCred, data)
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup,
	input *api.LoadbalancerBackendCreateInput) (*api.LoadbalancerBackendCreateInput, error) {
	if input.BackendType != api.LB_BACKEND_GUEST {
		return nil, httperrors.NewUnsupportOperationError("internal error: unexpected backend type %s", input.BackendType)
	}
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, input *api.LoadbalancerBackendUpdateInput) (*api.LoadbalancerBackendUpdateInput, error) {
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, input *api.LoadbalancerBackendGroupCreateInput) (*api.LoadbalancerBackendGroupCreateInput, error) {
	for _, backend := range input.Backends {
		if len(backend.ExternalId) == 0 {
			return nil, httperrors.NewInputParameterError("invalid guest %s", backend.Name)
		}
	}
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) IsSupportLoadbalancerListenerRuleRedirect() bool {
	return false
}

func validateUniqueById(ctx context.Context, userCred mcclient.TokenCredential, man db.IResourceModelManager, id string) error {
	q := man.Query().Equals("id", id)
	q = man.FilterByOwner(ctx, q, man, userCred, userCred, man.NamespaceScope())
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

func (self *SManagedVirtualizationRegionDriver) ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	lblis *models.SLoadbalancerListener, input *api.LoadbalancerListenerUpdateInput) (*api.LoadbalancerListenerUpdateInput, error) {
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) GetBackendStatusForAdd() []string {
	return []string{api.VM_RUNNING}
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerInstance(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, input *api.LoadbalancerCreateInput, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lb.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIRegion")
		}

		params := &cloudprovider.SLoadbalancerCreateOptions{
			Name:             lb.Name,
			Desc:             lb.Description,
			Address:          lb.Address,
			AddressType:      lb.AddressType,
			ChargeType:       lb.ChargeType,
			EgressMbps:       lb.EgressMbps,
			LoadbalancerSpec: lb.LoadbalancerSpec,
		}
		params.Tags, _ = lb.GetAllUserMetadata()

		if len(input.EipId) > 0 {
			eipObj, err := models.ElasticipManager.FetchById(input.EipId)
			if err != nil {
				return nil, errors.Wrapf(err, "eip.FetchById(%s)", input.EipId)
			}
			eip := eipObj.(*models.SElasticip)
			params.EipId = eip.ExternalId
		}

		if len(lb.ZoneId) > 0 {
			zone, err := lb.GetZone()
			if err != nil {
				return nil, errors.Wrapf(err, "GetZone")
			}
			iZone, err := iRegion.GetIZoneById(zone.ExternalId)
			if err != nil {
				return nil, errors.Wrapf(err, "GetIZoneById(%s)", zone.ExternalId)
			}
			params.ZoneId = iZone.GetId()
		}

		if len(lb.Zone1) > 0 {
			z1 := models.ZoneManager.FetchZoneById(lb.Zone1)
			if z1 == nil {
				return nil, fmt.Errorf("failed to find zone 1 for lb %s", lb.Name)
			}
			iZone, err := iRegion.GetIZoneById(z1.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "GetIZoneById")
			}
			params.SlaveZoneId = iZone.GetId()
		}
		if len(lb.VpcId) > 0 {
			vpc, err := lb.GetVpc()
			if err != nil {
				return nil, errors.Wrapf(err, "GetVpc")
			}
			params.VpcId = vpc.ExternalId
		}
		networks := []string{input.NetworkId}
		networks = append(networks, input.Networks...)
		for i := range networks {
			if len(networks[i]) > 0 {
				netObj, err := validators.ValidateModel(ctx, userCred, models.NetworkManager, &networks[i])
				if err != nil {
					return nil, err
				}
				network := netObj.(*models.SNetwork)
				if !utils.IsInStringArray(network.ExternalId, params.NetworkIds) {
					params.NetworkIds = append(params.NetworkIds, network.ExternalId)
				}
			}
		}

		manager := lb.GetCloudprovider()
		params.ProjectId, err = manager.SyncProject(ctx, userCred, lb.ProjectId)
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				logclient.AddSimpleActionLog(lb, logclient.ACT_SYNC_CLOUD_PROJECT, err, userCred, false)
			}
		}

		log.Debugf("create lb with params: %s", jsonutils.Marshal(params).String())
		iLoadbalancer, err := iRegion.CreateILoadBalancer(params)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateILoadBalancer")
		}
		err = db.SetExternalId(lb, userCred, iLoadbalancer.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "SetExternalId")
		}

		//wait async create result
		err = cloudprovider.WaitMultiStatus(iLoadbalancer, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_UNKNOWN}, 10*time.Second, 8*time.Minute)
		if err != nil {
			return nil, errors.Wrap(err, "cloudprovider.WaitMultiStatus")
		}

		err = lb.SyncWithCloudLoadbalancer(ctx, userCred, iLoadbalancer, lb.GetCloudprovider())
		if err != nil {
			return nil, errors.Wrapf(err, "SyncWithCloudLoadbalancer")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestStartLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lb.GetIRegion(ctx)
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
		iRegion, err := lb.GetIRegion(ctx)
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
		iLb, err := lb.GetILoadbalancer(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetILoadbalancer")
		}
		provider := lb.GetCloudprovider()
		return nil, lb.SyncWithCloudLoadbalancer(ctx, userCred, iLb, provider)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRemoteUpdateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iLb, err := lb.GetILoadbalancer(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetILoadbalancer")
		}
		oldTags, err := iLb.GetTags()
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
		err = cloudprovider.SetTags(ctx, iLb, lb.ManagerId, tags, replaceTags)
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
		iLb, err := lb.GetILoadbalancer(ctx)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "GetILoadbalancer")
		}
		err = iLb.Delete(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "iLb.Delete")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lbacl.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIRegion")
		}
		opts := &cloudprovider.SLoadbalancerAccessControlList{
			Name:   lbacl.Name,
			Entrys: []cloudprovider.SLoadbalancerAccessControlListEntry{},
		}
		if lbacl.AclEntries != nil {
			for _, entry := range *lbacl.AclEntries {
				opts.Entrys = append(opts.Entrys, cloudprovider.SLoadbalancerAccessControlListEntry{
					Comment: entry.Comment,
					CIDR:    entry.Cidr,
				})
			}
		}
		iAcl, err := iRegion.CreateILoadBalancerAcl(opts)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateILoadBalancerAcl")
		}
		_, err = db.Update(lbacl, func() error {
			lbacl.ExternalId = iAcl.GetGlobalId()
			return nil
		})
		return nil, err
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestUpdateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iAcl, err := lbacl.GetILoadbalancerAcl(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetILoadbalancerAcl")
		}
		opts := &cloudprovider.SLoadbalancerAccessControlList{
			Name:   lbacl.Name,
			Entrys: []cloudprovider.SLoadbalancerAccessControlListEntry{},
		}
		if lbacl.AclEntries != nil {
			for _, entry := range *lbacl.AclEntries {
				opts.Entrys = append(opts.Entrys, cloudprovider.SLoadbalancerAccessControlListEntry{
					Comment: entry.Comment,
					CIDR:    entry.Cidr,
				})
			}
		}
		return nil, iAcl.Sync(opts)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestLoadbalancerAclSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iAcl, err := lbacl.GetILoadbalancerAcl(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetILoadbalancerAcl")
		}
		return nil, lbacl.SyncWithCloudAcl(ctx, userCred, iAcl, lbacl.GetCloudprovider())
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iAcl, err := lbacl.GetILoadbalancerAcl(ctx)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "GetILoadbalancerAcl")
		}
		return nil, iAcl.Delete()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := lbcert.GetIRegion(ctx)
		if err != nil {
			return nil, err
		}

		opts := &cloudprovider.SLoadbalancerCertificate{
			Name:        lbcert.Name,
			PrivateKey:  lbcert.PrivateKey,
			Certificate: lbcert.Certificate,
		}

		iCert, err := iRegion.CreateILoadBalancerCertificate(opts)
		if err != nil {
			return nil, err
		}

		_, err = db.Update(lbcert, func() error {
			lbcert.ExternalId = iCert.GetGlobalId()
			return nil
		})
		return nil, err
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iCert, err := lbcert.GetILoadbalancerCertificate(ctx)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
		return nil, iCert.Delete()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestLoadbalancerCertificateSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iCert, err := lbcert.GetILoadbalancerCertificate(ctx)
		if err != nil {
			return nil, err
		}
		return nil, lbcert.SyncWithCloudCert(ctx, userCred, iCert, lbcert.GetCloudprovider())
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lb, err := lbbg.GetLoadbalancer()
		if err != nil {
			return nil, errors.Wrapf(err, "GetLoadbalancer")
		}
		iLb, err := lb.GetILoadbalancer(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetILoadBalancer(%s)", lb.ExternalId)
		}
		group := &cloudprovider.SLoadbalancerBackendGroup{
			Name:      lbbg.Name,
			GroupType: lbbg.Type,
		}
		iLbbg, err := iLb.CreateILoadBalancerBackendGroup(group)
		if err != nil {
			return nil, err
		}
		err = db.SetExternalId(lbbg, userCred, iLbbg.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "db.SetExternalId")
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
		iRegion, err := lbbg.GetIRegion(ctx)
		if err != nil {
			return nil, err
		}
		loadbalancer, err := lbbg.GetLoadbalancer()
		if err != nil {
			return nil, err
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

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		lbbg, err := lbb.GetLoadbalancerBackendGroup()
		if err != nil {
			return nil, err
		}
		lb, err := lbbg.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := lb.GetIRegion(ctx)
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
		return nil, lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iLoadbalancerBackend, lb.GetCloudprovider())
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if jsonutils.QueryBoolean(task.GetParams(), "purge", false) {
			return nil, nil
		}
		lbbg, err := lbb.GetLoadbalancerBackendGroup()
		if err != nil {
			return nil, err
		}
		lb, err := lbbg.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := lb.GetIRegion(ctx)
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
		_, err = guest.GetIVM(ctx)
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
		lbbg, err := lbb.GetLoadbalancerBackendGroup()
		if err != nil {
			return nil, err
		}
		lb, err := lbbg.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := lb.GetIRegion(ctx)
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

		return nil, lbb.SyncWithCloudLoadbalancerBackend(ctx, userCred, iBackend, lb.GetCloudprovider())
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		provider := lblis.GetCloudprovider()
		if provider == nil {
			return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
		}

		params, err := lblis.GetLoadbalancerListenerParams()
		if err != nil {
			return nil, errors.Wrapf(err, "lblis.GetLoadbalancerListenerParams")
		}

		{
			if lblis.ListenerType == api.LB_LISTENER_TYPE_HTTPS && len(lblis.CertificateId) > 0 {
				cert, err := lblis.GetCertificate()
				if err != nil {
					return nil, errors.Wrapf(err, "GetCertificate")
				}
				params.CertificateId = cert.ExternalId
			}
		}

		{
			if len(lblis.AclId) > 0 {
				acl, err := lblis.GetAcl()
				if err != nil {
					return nil, errors.Wrap(err, "GetAcl")
				}
				params.AccessControlListId = acl.ExternalId
				params.AccessControlListType = lblis.AclType
				params.AccessControlListStatus = lblis.AclStatus
			}
		}

		loadbalancer, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
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
		if len(lblis.ExternalId) == 0 {
			return nil, nil
		}
		lb, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iLb, err := lb.GetILoadbalancer(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetILoadbalancer")
		}

		iListener, err := iLb.GetILoadBalancerListenerById(lblis.ExternalId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "GetILoadBalancerListenerById(%s)", lblis.ExternalId)
		}
		return nil, iListener.Delete(ctx)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestStartLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		loadbalancer, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
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

func (self *SManagedVirtualizationRegionDriver) RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, input *api.LoadbalancerListenerUpdateInput, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iLis, err := lblis.GetILoadbalancerListener(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetILoadbalancerListener")
		}

		provider := lblis.GetCloudprovider()
		if provider == nil {
			return nil, fmt.Errorf("failed to find provider for lblis %s", lblis.Name)
		}

		{
			if lblis.ListenerType == api.LB_LISTENER_TYPE_HTTPS && input.CertificateId != nil {
				cert, err := lblis.GetCertificate()
				if err != nil {
					return nil, errors.Wrapf(err, "GetCertificate")
				}
				err = iLis.ChangeCertificate(ctx, &cloudprovider.ListenerCertificateOptions{
					CertificateId: cert.ExternalId,
				})
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
					return nil, errors.Wrapf(err, "ChangeCertificate")
				}
			}
		}

		{
			if input.AclStatus != nil {
				opts := &cloudprovider.ListenerAclOptions{AclStatus: *input.AclStatus, AclType: lblis.AclType}
				if *input.AclStatus == api.LB_BOOL_ON {
					acl, err := lblis.GetAcl()
					if err != nil {
						return nil, errors.Wrapf(err, "GetAcl")
					}
					opts.AclId = acl.ExternalId
				}
				err := iLis.SetAcl(ctx, opts)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotSupported {
					return nil, errors.Wrapf(err, "SetAcl")
				}
			}
		}

		if iLis.GetScheduler() != lblis.Scheduler {
			err := iLis.ChangeScheduler(ctx, &cloudprovider.ChangeListenerSchedulerOptions{
				Scheduler: lblis.Scheduler,
				ListenerStickySessionOptions: cloudprovider.ListenerStickySessionOptions{
					StickySession:              lblis.StickySession,
					StickySessionCookie:        lblis.StickySessionCookie,
					StickySessionType:          lblis.StickySessionType,
					StickySessionCookieTimeout: lblis.StickySessionCookieTimeout,
				},
			})
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				return nil, errors.Wrapf(err, "ChangeScheduler")
			}
		}
		if input.HealthCheck != nil {
			err := iLis.SetHealthCheck(ctx, &cloudprovider.ListenerHealthCheckOptions{
				HealthCheckReq: lblis.HealthCheckReq,
				HealthCheckExp: lblis.HealthCheckExp,

				HealthCheck:         lblis.HealthCheck,
				HealthCheckType:     lblis.HealthCheckType,
				HealthCheckTimeout:  lblis.HealthCheckTimeout,
				HealthCheckDomain:   lblis.HealthCheckDomain,
				HealthCheckHttpCode: lblis.HealthCheckHttpCode,
				HealthCheckURI:      lblis.HealthCheckURI,
				HealthCheckInterval: lblis.HealthCheckInterval,

				HealthCheckRise: lblis.HealthCheckRise,
				HealthCheckFail: lblis.HealthCheckFall,
			})
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				return nil, errors.Wrapf(err, "SetHealthCheck")
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestStopLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		loadbalancer, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
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
		loadbalancer, err := lblis.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
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
			return nil, lblis.SetStatus(ctx, userCred, status, "")
		}
		return nil, fmt.Errorf("Unknown loadbalancer listener status %s", status)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		listener, err := lbr.GetLoadbalancerListener()
		if err != nil {
			return nil, err
		}
		loadbalancer, err := listener.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
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
			rule.BackendGroupId = group.ExternalId
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
		listener, err := lbr.GetLoadbalancerListener()
		if err != nil {
			return nil, err
		}
		loadbalancer, err := listener.GetLoadbalancer()
		if err != nil {
			return nil, err
		}
		iRegion, err := loadbalancer.GetIRegion(ctx)
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
		iregion, err := vpc.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetIRegion")
		}
		opts := &cloudprovider.VpcCreateOptions{
			NAME: vpc.Name,
			CIDR: vpc.CidrBlock,
			Desc: vpc.Description,
		}
		ivpc, err := iregion.CreateIVpc(opts)
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
		secgroups, err := vpc.GetSecurityGroups()
		if err != nil {
			return nil, errors.Wrapf(err, "GetSecurityGroups")
		}
		for i := range secgroups {
			iGroup, err := secgroups[i].GetISecurityGroup(ctx)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					continue
				}
				return nil, errors.Wrapf(err, "GetISecurityGroup")
			}
			err = iGroup.Delete()
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotSupported {
				return nil, errors.Wrapf(err, "delete secgroup %s", secgroups[i].Name)
			}
			err = secgroups[i].RealDelete(ctx, userCred)
			if err != nil {
				return nil, errors.Wrapf(err, "real delete secgroup %s", secgroups[i].Name)
			}
		}
		ivpc, err := vpc.GetIVpc(ctx)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				// already deleted, do nothing
				return nil, nil
			}
			return nil, errors.Wrap(err, "GetIVpc")
		}
		err = ivpc.Delete()
		if err != nil {
			return nil, errors.Wrap(err, "Delete")
		}
		err = cloudprovider.WaitDeleted(ivpc, 10*time.Second, 300*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "cloudprovider.WaitDeleted")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		cloudRegion, err := snapshot.GetISnapshotRegion(ctx)
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
	iDisk, err := disk.GetIDisk(ctx)
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

func (self *SManagedVirtualizationRegionDriver) RequestCreateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *models.SDBInstance, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iregion, err := dbinstance.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "GetIRegionAndProvider")
		}

		vpc, err := dbinstance.GetVpc()
		if err != nil {
			return nil, errors.Wrap(err, "dbinstance.GetVpc()")
		}

		input := api.DBInstanceCreateInput{}
		task.GetParams().Unmarshal(&input)
		if len(input.Password) == 0 && jsonutils.QueryBoolean(task.GetParams(), "reset_password", true) {
			input.Password = seclib2.RandomPassword2(12)
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
			Password:      input.Password,
			MultiAz:       input.MultiAZ,
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
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				logclient.AddSimpleActionLog(dbinstance, logclient.ACT_SYNC_CLOUD_PROJECT, err, userCred, false)
			}
		}

		region, err := dbinstance.GetRegion()
		if err != nil {
			return nil, err
		}

		err = region.GetDriver().InitDBInstanceUser(ctx, dbinstance, task, &desc)
		if err != nil {
			return nil, err
		}

		secgroups, err := dbinstance.GetSecgroups()
		if err != nil {
			return nil, errors.Wrapf(err, "GetSecgroups")
		}
		driver := region.GetDriver()
		ownerId := dbinstance.GetOwnerId()
		for i := range secgroups {
			if secgroups[i].Id == api.SECGROUP_DEFAULT_ID {
				filter, err := driver.GetSecurityGroupFilter(vpc)
				if err != nil {
					return nil, errors.Wrapf(err, "GetSecurityGroupFilter")
				}
				group, err := vpc.GetDefaultSecurityGroup(ownerId, filter)
				if err != nil && errors.Cause(err) != sql.ErrNoRows {
					return nil, err
				}
				if gotypes.IsNil(group) {
					group, err = driver.CreateDefaultSecurityGroup(ctx, userCred, ownerId, vpc)
					if err != nil {
						return nil, errors.Wrapf(err, "CreateDefaultSecurityGroup")
					}
				}
				if !utils.IsInStringArray(group.ExternalId, desc.SecgroupIds) {
					desc.SecgroupIds = append(desc.SecgroupIds, group.ExternalId)
				}
				continue
			}
			if !utils.IsInStringArray(secgroups[i].ExternalId, desc.SecgroupIds) {
				desc.SecgroupIds = append(desc.SecgroupIds, secgroups[i].ExternalId)
			}
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
		iBackup, err := backup.GetIDBInstanceBackup(ctx)
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
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				logclient.AddSimpleActionLog(rds, logclient.ACT_SYNC_CLOUD_PROJECT, err, userCred, false)
			}
		}

		region, err := rds.GetRegion()
		if err != nil {
			return nil, err
		}

		err = region.GetDriver().InitDBInstanceUser(ctx, rds, task, &desc)
		if err != nil {
			return nil, err
		}

		secgroups, err := rds.GetSecgroups()
		if err != nil {
			return nil, errors.Wrapf(err, "GetSecgroups")
		}
		driver := region.GetDriver()
		ownerId := rds.GetOwnerId()
		for i := range secgroups {
			if secgroups[i].Id == api.SECGROUP_DEFAULT_ID {
				filter, err := driver.GetSecurityGroupFilter(vpc)
				if err != nil {
					return nil, errors.Wrapf(err, "GetSecurityGroupFilter")
				}
				group, err := vpc.GetDefaultSecurityGroup(ownerId, filter)
				if err != nil && errors.Cause(err) != sql.ErrNoRows {
					return nil, err
				}
				if gotypes.IsNil(group) {
					group, err = driver.CreateDefaultSecurityGroup(ctx, userCred, ownerId, vpc)
					if err != nil {
						return nil, errors.Wrapf(err, "CreateDefaultSecurityGroup")
					}
				}
				if !utils.IsInStringArray(group.ExternalId, desc.SecgroupIds) {
					desc.SecgroupIds = append(desc.SecgroupIds, group.ExternalId)
				}
				continue
			}
			if !utils.IsInStringArray(secgroups[i].ExternalId, desc.SecgroupIds) {
				desc.SecgroupIds = append(desc.SecgroupIds, secgroups[i].ExternalId)
			}
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

func (self *SManagedVirtualizationRegionDriver) RequestCreateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask, data *jsonutils.JSONDict) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := ec.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "GetIRegion")
		}

		iprovider, err := db.FetchById(models.CloudproviderManager, ec.GetCloudproviderId())
		if err != nil {
			return nil, errors.Wrap(err, "GetProvider")
		}
		vpc, err := ec.GetVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "GetVpc")
		}

		networkObj, err := db.FetchById(models.NetworkManager, ec.NetworkId)
		if err != nil {
			return nil, errors.Wrap(err, "GetNetwork")
		}
		network := networkObj.(*models.SNetwork)

		params := &cloudprovider.SCloudElasticCacheInput{
			InstanceType:     ec.InstanceType,
			InstanceName:     ec.Name,
			Engine:           ec.Engine,
			EngineVersion:    ec.EngineVersion,
			PrivateIpAddress: ec.PrivateIpAddr,
			CapacityGB:       int64(ec.CapacityMB / 1024),
			NodeType:         ec.NodeType,
			NetworkType:      ec.NetworkType,
			VpcId:            vpc.ExternalId,
			NetworkId:        network.ExternalId,
			MaintainBegin:    ec.MaintainStartTime,
			MaintainEnd:      ec.MaintainEndTime,
		}
		params.Password, _ = task.GetParams().GetString("password")
		if ec.BillingType == billing_api.BILLING_TYPE_PREPAID {
			bc, err := billing.ParseBillingCycle(ec.BillingCycle)
			if err != nil {
				return nil, errors.Wrapf(err, "ParseBillingCycle(%s)", ec.BillingCycle)
			}
			bc.AutoRenew = ec.AutoRenew
			params.BillingCycle = &bc
		}
		params.Tags, _ = ec.GetAllUserMetadata()

		zone, err := ec.GetZone()
		if zone != nil {
			izone, err := iRegion.GetIZoneById(zone.ExternalId)
			if err != nil {
				return nil, errors.Wrapf(err, "GetIZoneById(%s)", zone.ExternalId)
			}
			params.ZoneIds = []string{izone.GetId()}
		}
		slaveZones, err := ec.GetSlaveZones()
		if err != nil {
			return nil, errors.Wrapf(err, "GetSlaveZones")
		}
		for i := range slaveZones {
			izone, err := iRegion.GetIZoneById(slaveZones[i].ExternalId)
			if err != nil {
				return nil, errors.Wrapf(err, "GetSlaveZoneBy(%s)", slaveZones[i].ExternalId)
			}
			if !utils.IsInStringArray(izone.GetId(), params.ZoneIds) {
				params.ZoneIds = append(params.ZoneIds, izone.GetId())
			}
		}

		data.Unmarshal(&params.SecurityGroupIds, "ext_secgroup_ids")

		provider := iprovider.(*models.SCloudprovider)
		params.ProjectId, err = provider.SyncProject(ctx, userCred, ec.ProjectId)
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				logclient.AddSimpleActionLog(ec, logclient.ACT_SYNC_CLOUD_PROJECT, err, userCred, false)
			}
		}

		iec, err := iRegion.CreateIElasticcaches(params)
		if err != nil {
			return nil, errors.Wrap(err, "CreateIElasticcaches")
		}

		err = db.SetExternalId(ec, userCred, iec.GetGlobalId())
		if err != nil {
			return nil, errors.Wrap(err, "SetExternalId")
		}

		err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 30*time.Second, 15*time.Second, 30*time.Minute)
		if err != nil {
			return nil, errors.Wrap(err, "WaitStatusWithDelay")
		}

		err = ec.SyncWithCloudElasticcache(ctx, userCred, provider, iec)
		if err != nil {
			return nil, errors.Wrap(err, "SyncWithCloudElasticcache")
		}

		// sync accounts
		{
			iaccounts, err := iec.GetICloudElasticcacheAccounts()
			if err != nil {
				return nil, errors.Wrap(err, "GetICloudElasticcacheAccounts")
			}

			models.ElasticcacheAccountManager.SyncElasticcacheAccounts(ctx, userCred, ec, iaccounts)

			account, err := ec.GetAdminAccount()
			if err == nil {
				account.SavePassword(params.Password)
			}
		}

		// sync acl
		{
			iacls, err := iec.GetICloudElasticcacheAcls()
			if err != nil {
				if !(errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented) {
					return nil, errors.Wrap(err, "GetICloudElasticcacheAcls")
				}
				models.ElasticcacheAclManager.SyncElasticcacheAcls(ctx, userCred, ec, iacls)
			}
		}

		// sync parameters
		{
			iparams, err := iec.GetICloudElasticcacheParameters()
			if err != nil {
				if !(errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotSupported) {
					return nil, errors.Wrap(err, "GetICloudElasticcacheParameters")
				}
				models.ElasticcacheParameterManager.SyncElasticcacheParameters(ctx, userCred, ec, iparams)
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ElasticcacheCreateInput) (*api.ElasticcacheCreateInput, error) {
	if len(input.ManagerId) == 0 {
		return nil, httperrors.NewMissingParameterError("manager_id")
	}
	if len(input.NetworkType) == 0 {
		input.NetworkType = api.LB_NETWORK_TYPE_VPC
	}
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRestartElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	iregion, err := ec.GetIRegion(ctx)
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

	return ec.SetStatus(ctx, userCred, api.ELASTIC_CACHE_STATUS_RUNNING, "")
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	iregion, err := ec.GetIRegion(ctx)
	if err != nil {
		return errors.Wrap(err, "GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if err != nil {
		return errors.Wrap(err, "GetIElasticcacheById")
	}

	provider := ec.GetCloudprovider()
	if provider == nil {
		return errors.Wrap(fmt.Errorf("provider is nil"), "GetCloudprovider")
	}

	lockman.LockRawObject(ctx, "elastic-cache", ec.Id)
	defer lockman.ReleaseRawObject(ctx, "elastic-cache", ec.Id)

	err = ec.SyncWithCloudElasticcache(ctx, userCred, provider, iec)
	if err != nil {
		return errors.Wrap(err, "SyncWithCloudElasticcache")
	}

	if fullsync, _ := task.GetParams().Bool("full"); fullsync {
		lockman.LockObject(ctx, ec)
		defer lockman.ReleaseObject(ctx, ec)

		parameters, err := iec.GetICloudElasticcacheParameters()
		if err != nil {
			if !(errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported) {
				return errors.Wrapf(err, "GetICloudElasticcacheParameters")
			}
			result := models.ElasticcacheParameterManager.SyncElasticcacheParameters(ctx, userCred, ec, parameters)
			log.Infof("SyncElasticcacheParameters %s", result.Result())
		}

		// acl
		acls, err := iec.GetICloudElasticcacheAcls()
		if err != nil {
			if !(errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported) {
				return errors.Wrapf(err, "GetICloudElasticcacheAcls")
			}
			result := models.ElasticcacheAclManager.SyncElasticcacheAcls(ctx, userCred, ec, acls)
			log.Infof("SyncElasticcacheAcls %s", result.Result())
		}

		// account
		accounts, err := iec.GetICloudElasticcacheAccounts()
		if err != nil {
			if !(errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported) {
				return errors.Wrapf(err, "GetICloudElasticcacheAccounts")
			}
			result := models.ElasticcacheAccountManager.SyncElasticcacheAccounts(ctx, userCred, ec, accounts)
			log.Infof("SyncElasticcacheAccounts %s", result.Result())
		}

		// backups
		backups, err := iec.GetICloudElasticcacheBackups()
		if err != nil {
			if !(errors.Cause(err) == cloudprovider.ErrNotImplemented || errors.Cause(err) == cloudprovider.ErrNotSupported) {
				return errors.Wrapf(err, "GetICloudElasticcacheAccounts")
			}
			result := models.ElasticcacheBackupManager.SyncElasticcacheBackups(ctx, userCred, ec, backups)
			log.Infof("SyncElasticcacheBackups %s", result.Result())
		}
	}

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	if len(ec.ExternalId) == 0 {
		return nil
	}

	iregion, err := ec.GetIRegion(ctx)
	if err != nil {
		return errors.Wrap(err, "GetIRegion")
	}

	iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "GetIElasticcacheById")
	}

	err = iec.Delete()
	if err != nil {
		return errors.Wrap(err, "Delete")
	}

	return cloudprovider.WaitDeleted(iec, 10*time.Second, 10*time.Minute)
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

	iregion, err := ec.GetIRegion(ctx)
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

	iregion, err := ec.GetIRegion(ctx)
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

	iregion, err := ec.GetIRegion(ctx)
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

	iregion, err := ec.GetIRegion(ctx)
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
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		port, _ := task.GetParams().Int("port")
		iregion, err := ec.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "GerIRegion")
		}

		iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIElasticcacheById(%s)", ec.ExternalId)
		}

		_, err = iec.AllocatePublicConnection(int(port))
		if err != nil {
			return nil, errors.Wrapf(err, "AllocatePublicConnection(%d)", port)
		}

		err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "WaitStatusWithDelay")
		}

		err = ec.SyncWithCloudElasticcache(ctx, task.GetUserCred(), ec.GetCloudprovider(), iec)
		if err != nil {
			return nil, errors.Wrap(err, "SyncWithCloudElasticcache")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheReleasePublicConnection(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iregion, err := ec.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "GetIRegion")
		}

		iec, err := iregion.GetIElasticcacheById(ec.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIElasticcacheById(%s)", ec.ExternalId)
		}

		err = iec.ReleasePublicConnection()
		if err != nil {
			return nil, errors.Wrap(err, "ReleasePublicConnection")
		}

		err = cloudprovider.WaitStatusWithDelay(iec, api.ELASTIC_CACHE_STATUS_RUNNING, 10*time.Second, 10*time.Second, 300*time.Second)
		if err != nil {
			return nil, errors.Wrap(errors.ErrTimeout, "WaitStatusWithDelay")
		}

		err = ec.SyncWithCloudElasticcache(ctx, task.GetUserCred(), ec.GetCloudprovider(), iec)
		if err != nil {
			return nil, errors.Wrap(err, "SyncWithCloudElasticcache")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheFlushInstance(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	iregion, err := ec.GetIRegion(ctx)
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

	iregion, err := ec.GetIRegion(ctx)
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

	iregion, err := ec.GetIRegion(ctx)
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
			if err := cidrV.Validate(ctx, params); err != nil {
				return nil, err
			}
		} else {
			if err := ipV.Validate(ctx, params); err != nil {
				return nil, err
			}
		}
	}

	elasticcacheV := validators.NewModelIdOrNameValidator("elasticcache", "elasticcache", ownerId)
	if err := elasticcacheV.Validate(ctx, data); err != nil {
		return nil, err
	}

	return data, nil
}

func (self *SManagedVirtualizationRegionDriver) AllowCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, elasticcache *models.SElasticcache) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateElasticcacheBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	elasticcacheV := validators.NewModelIdOrNameValidator("elasticcache", "elasticcache", ownerId)
	if err := elasticcacheV.Validate(ctx, data); err != nil {
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
		iregion, err := ec.GetIRegion(ctx)
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

		iRds, err := rds.GetIDBInstance(ctx)
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
		iRds, err := instance.GetIDBInstance(ctx)
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
			iBackup, err := backup.GetIDBInstanceBackup(ctx)
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

		instance.SetStatus(ctx, userCred, api.DBINSTANCE_RUNNING, "")
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateResetDBInstancePassword(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, account string) error {
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRemoteUpdateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRds, err := instance.GetIDBInstance(ctx)
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

		err = iRds.Update(ctx, cloudprovider.SDBInstanceUpdateOptions{NAME: instance.Name, Description: instance.Description})
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			return nil, errors.Wrap(err, "iRds.Update")
		}
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
		iregion, err := ec.GetIRegion(ctx)
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
	iregion, err := ea.GetIRegion(ctx)
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
	iregion, err := ea.GetIRegion(ctx)
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
	iregion, err := eb.GetIRegion(ctx)
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
	iregion, err := ea.GetIRegion(ctx)
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

	return ea.SetStatus(ctx, userCred, api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE, "")
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheAclUpdate(ctx context.Context, userCred mcclient.TokenCredential, ea *models.SElasticcacheAcl, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		_ec, err := db.FetchById(models.ElasticcacheManager, ea.ElasticcacheId)
		if err != nil {
			return nil, errors.Wrap(nil, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.GetElasticcache")
		}

		ec := _ec.(*models.SElasticcache)
		iregion, err := ec.GetIRegion(ctx)
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

		err = ea.SetStatus(ctx, userCred, api.ELASTIC_CACHE_ACL_STATUS_AVAILABLE, "")
		if err != nil {
			return nil, errors.Wrap(err, "managedVirtualizationRegionDriver.CreateElasticcacheAcl.UpdateAclStatus")
		}
		return nil, nil
	})

	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestElasticcacheBackupRestoreInstance(ctx context.Context, userCred mcclient.TokenCredential, eb *models.SElasticcacheBackup, task taskman.ITask) error {
	iregion, err := eb.GetIRegion(ctx)
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
		iDisk, err := disk.GetIDisk(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "disk.GetIDisk")
		}

		return nil, disk.SetStatus(ctx, userCred, iDisk.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncDiskBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, backup *models.SDiskBackup, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncDiskBackupStatus")
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncSnapshotStatus(ctx context.Context, userCred mcclient.TokenCredential, snapshot *models.SSnapshot, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := snapshot.GetISnapshotRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "snapshot.GetISnapshotRegion")
		}

		iSnapshot, err := iRegion.GetISnapshotById(snapshot.ExternalId)
		if err != nil {
			return nil, errors.Wrapf(err, "iRegion.GetISnapshotById(%s)", snapshot.ExternalId)
		}

		return nil, snapshot.SetStatus(ctx, userCred, iSnapshot.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncNatGatewayStatus(ctx context.Context, userCred mcclient.TokenCredential, nat *models.SNatGateway, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

		iNat, err := nat.GetINatGateway(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "nat.GetINatGateway")
		}

		return nil, nat.SyncWithCloudNatGateway(ctx, userCred, nat.GetCloudprovider(), iNat)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncBucketStatus(ctx context.Context, userCred mcclient.TokenCredential, bucket *models.SBucket, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iBucket, err := bucket.GetIBucket(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "bucket.GetIBucket")
		}

		return nil, bucket.SyncWithCloudBucket(ctx, userCred, iBucket, false)
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncDBInstanceBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, backup *models.SDBInstanceBackup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iDBInstanceBackup, err := backup.GetIDBInstanceBackup(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "backup.GetIDBInstanceBackup")
		}

		return nil, backup.SetStatus(ctx, userCred, iDBInstanceBackup.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSyncElasticcacheStatus(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := elasticcache.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "elasticcache.GetIRegion")
		}

		iElasticcache, err := iRegion.GetIElasticcacheById(elasticcache.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "elasticcache.GetIElasticcache")
		}
		if account := elasticcache.GetCloudaccount(); account != nil {
			models.SyncVirtualResourceMetadata(ctx, userCred, elasticcache, iElasticcache, account.ReadOnly)
		}
		return nil, elasticcache.SetStatus(ctx, userCred, iElasticcache.GetStatus(), "syncstatus")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRemoteUpdateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := elasticcache.GetIRegion(ctx)
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
		if vpc, _ := elasticcache.GetVpc(); vpc != nil {
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
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		// sync secgroups to cloud
		secgroupExternalIds := []string{}
		{
			vpc, err := ec.GetVpc()
			if err != nil {
				return nil, errors.Wrapf(err, "GetVpc")
			}
			region, err := vpc.GetRegion()
			if err != nil {
				return nil, errors.Wrapf(err, "GetRegion")
			}
			secgroups, err := ec.GetSecgroups()
			if err != nil {
				return nil, errors.Wrapf(err, "GetSecgroups")
			}
			driver := region.GetDriver()
			ownerId := ec.GetOwnerId()
			for i := range secgroups {
				if secgroups[i].Id == api.SECGROUP_DEFAULT_ID {
					filter, err := driver.GetSecurityGroupFilter(vpc)
					if err != nil {
						return nil, errors.Wrapf(err, "GetSecurityGroupFilter")
					}
					group, err := vpc.GetDefaultSecurityGroup(ownerId, filter)
					if err != nil && errors.Cause(err) != sql.ErrNoRows {
						return nil, err
					}
					if gotypes.IsNil(group) {
						group, err = driver.CreateDefaultSecurityGroup(ctx, userCred, ownerId, vpc)
						if err != nil {
							return nil, errors.Wrapf(err, "CreateDefaultSecurityGroup")
						}
					}
					if !utils.IsInStringArray(group.ExternalId, secgroupExternalIds) {
						secgroupExternalIds = append(secgroupExternalIds, group.ExternalId)
					}
					continue
				}
				if !utils.IsInStringArray(secgroups[i].ExternalId, secgroupExternalIds) {
					secgroupExternalIds = append(secgroupExternalIds, secgroups[i].ExternalId)
				}
			}
		}

		ret := jsonutils.NewDict()
		ret.Set("ext_secgroup_ids", jsonutils.NewStringArray(secgroupExternalIds))
		return ret, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRenewElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, bc billing.SBillingCycle) (time.Time, error) {
	iregion, err := ec.GetIRegion(ctx)
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
	iregion, err := ec.GetIRegion(ctx)
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

	bc := billing.SBillingCycle{}
	bc.AutoRenew = autoRenew
	err = iec.SetAutoRenew(bc)
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
		secgroups, err := rds.GetSecgroups()
		if err != nil {
			return nil, errors.Wrapf(err, "GetSecgroups")
		}
		vpc, err := rds.GetVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "GetVpc")
		}
		region, err := vpc.GetRegion()
		if err != nil {
			return nil, errors.Wrapf(err, "GetRegion")
		}
		driver := region.GetDriver()
		ownerId := rds.GetOwnerId()
		secgroupIds := []string{}
		for i := range secgroups {
			if secgroups[i].Id == api.SECGROUP_DEFAULT_ID {
				filter, err := driver.GetSecurityGroupFilter(vpc)
				if err != nil {
					return nil, errors.Wrapf(err, "GetSecurityGroupFilter")
				}
				group, err := vpc.GetDefaultSecurityGroup(ownerId, filter)
				if err != nil && errors.Cause(err) != sql.ErrNoRows {
					return nil, err
				}
				if gotypes.IsNil(group) {
					group, err = driver.CreateDefaultSecurityGroup(ctx, userCred, ownerId, vpc)
					if err != nil {
						return nil, errors.Wrapf(err, "CreateDefaultSecurityGroup")
					}
				}
				if !utils.IsInStringArray(group.ExternalId, secgroupIds) {
					secgroupIds = append(secgroupIds, group.ExternalId)
				}
				continue
			}
			if !utils.IsInStringArray(secgroups[i].ExternalId, secgroupIds) {
				secgroupIds = append(secgroupIds, secgroups[i].ExternalId)
			}
		}

		iRds, err := rds.GetIDBInstance(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIDBInstance")
		}
		err = iRds.SetSecurityGroups(secgroupIds)
		if err != nil {
			return nil, errors.Wrapf(err, "SetSecurityGroups")
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, eip *models.SElasticip, input api.ElasticipAssociateInput, obj db.IStatusStandaloneModel, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iEip, err := eip.GetIEip(ctx)
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
			db.StatusBaseSetStatus(ctx, obj, userCred, api.INSTANCE_ASSOCIATE_EIP, "associate eip")
		}

		err = eip.AssociateInstance(ctx, userCred, input.InstanceType, obj)
		if err != nil {
			return nil, errors.Wrapf(err, "eip.AssociateVM")
		}

		eip.SetStatus(ctx, userCred, api.EIP_STATUS_READY, api.EIP_STATUS_ASSOCIATE)
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateNetwork(ctx context.Context, userCred mcclient.TokenCredential, net *models.SNetwork, task taskman.ITask) error {
	wire, err := net.GetWire()
	if err != nil {
		return errors.Wrapf(err, "GetWire")
	}

	iwire, err := wire.GetIWire(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetIWire")
	}

	prefix, err := net.GetPrefix()
	if err != nil {
		return errors.Wrapf(err, "GetPrefix")
	}

	opts := cloudprovider.SNetworkCreateOptions{
		Name: net.Name,
		Cidr: prefix.String(),
		Desc: net.Description,
	}
	opts.AssignPublicIp, _ = task.GetParams().Bool("assign_public_ip")

	provider := wire.GetCloudprovider()
	opts.ProjectId, err = provider.SyncProject(ctx, userCred, net.ProjectId)
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
			logclient.AddSimpleActionLog(net, logclient.ACT_SYNC_CLOUD_PROJECT, err, userCred, false)
		}
	}

	inet, err := iwire.CreateINetwork(&opts)
	if err != nil {
		return errors.Wrapf(err, "CreateINetwork")
	}

	err = db.SetExternalId(net, userCred, inet.GetGlobalId())
	if err != nil {
		return errors.Wrapf(err, "db.SetExternalId")
	}

	err = cloudprovider.WaitStatus(inet, api.NETWORK_STATUS_AVAILABLE, 10*time.Second, 5*time.Minute)
	if err != nil {
		return errors.Wrapf(err, "wait network available after 5 minutes, current status: %s", inet.GetStatus())
	}

	return net.SyncWithCloudNetwork(ctx, userCred, inet)
}

func (self *SManagedVirtualizationRegionDriver) RequestRemoteUpdateElasticSearch(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SElasticSearch, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ies, err := instance.GetIElasticSearch(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "instance.GetIESInstance")
		}
		oldTags, err := ies.GetTags()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			return nil, errors.Wrap(err, "ies.GetTags()")
		}
		tags, err := instance.GetAllUserMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "instance.GetAllUserMetadata")
		}
		tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
		err = cloudprovider.SetTags(ctx, ies, instance.ManagerId, tags, replaceTags)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			logclient.AddActionLogWithStartable(task, instance, logclient.ACT_UPDATE_TAGS, err, userCred, false)
			return nil, errors.Wrap(err, "ies.SetTags")
		}
		logclient.AddActionLogWithStartable(task, instance, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, userCred, true)
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRemoteUpdateKafka(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SKafka, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		kafka, err := instance.GetIKafka(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "instance.GetIKafka")
		}
		oldTags, err := kafka.GetTags()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			return nil, errors.Wrap(err, "ies.GetTags()")
		}
		tags, err := instance.GetAllUserMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "instance.GetAllUserMetadata")
		}
		tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
		err = cloudprovider.SetTags(ctx, kafka, instance.ManagerId, tags, replaceTags)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			logclient.AddActionLogWithStartable(task, instance, logclient.ACT_UPDATE_TAGS, err, userCred, false)
			return nil, errors.Wrap(err, "ies.SetTags")
		}
		logclient.AddActionLogWithStartable(task, instance, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, userCred, true)
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateKubeClusterData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.KubeClusterCreateInput) (*api.KubeClusterCreateInput, error) {
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateKubeCluster(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SKubeCluster, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		opts := &cloudprovider.KubeClusterCreateOptions{
			NAME:       cluster.Name,
			Desc:       cluster.Description,
			Version:    cluster.Version,
			NetworkIds: []string{},
		}
		vpc, err := cluster.GetVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "cluster.GetVpc")
		}
		opts.VpcId = vpc.ExternalId
		networks, err := cluster.GetNetworks()
		if err != nil {
			return nil, errors.Wrapf(err, "GetNetworks")
		}
		for _, net := range networks {
			opts.NetworkIds = append(opts.NetworkIds, net.ExternalId)
		}
		opts.Tags, _ = cluster.GetAllUserMetadata()
		params := task.GetParams()
		opts.ServiceCIDR, _ = params.GetString("service_cidr")
		opts.RoleName, _ = params.GetString("role_name")
		opts.PrivateAccess, _ = params.Bool("private_access")
		opts.PublicAccess, _ = params.Bool("public_access")
		_, opts.PublicKey, _ = sshkeys.GetSshAdminKeypair(ctx)

		iregion, err := cluster.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIRegion")
		}
		icluster, err := iregion.CreateIKubeCluster(opts)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateIKubeCluster")
		}
		err = db.SetExternalId(cluster, userCred, icluster.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "db.SetExternalId")
		}
		err = cloudprovider.WaitStatusWithSync(icluster, api.KUBE_CLUSTER_STATUS_RUNNING, func(status string) {
			cluster.SetStatus(ctx, userCred, status, "")
		}, time.Second*30, time.Hour*1)
		if err != nil {
			return nil, errors.Wrapf(err, "wait cluster status timeout, current status: %s", icluster.GetStatus())
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateKubeNodePoolData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.KubeNodePoolCreateInput) (*api.KubeNodePoolCreateInput, error) {
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateKubeNodePool(ctx context.Context, userCred mcclient.TokenCredential, pool *models.SKubeNodePool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		opts := &cloudprovider.KubeNodePoolCreateOptions{
			NAME: pool.Name,
			Desc: pool.Description,

			MinInstanceCount:     pool.MinInstanceCount,
			MaxInstanceCount:     pool.MaxInstanceCount,
			DesiredInstanceCount: pool.DesiredInstanceCount,

			RootDiskSizeGb: pool.RootDiskSizeGb,

			NetworkIds:    []string{},
			InstanceTypes: []string{},
		}
		opts.PublicKey, _ = task.GetParams().GetString("public_key")

		if pool.InstanceTypes != nil {
			for _, instanceType := range *pool.InstanceTypes {
				opts.InstanceTypes = append(opts.InstanceTypes, instanceType)
			}
		}
		networks, err := pool.GetNetworks()
		if err != nil {
			return nil, errors.Wrapf(err, "GetNetworks")
		}
		for _, net := range networks {
			opts.NetworkIds = append(opts.NetworkIds, net.ExternalId)
		}
		opts.Tags, _ = pool.GetAllUserMetadata()

		icluster, err := pool.GetIKubeCluster(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIKubeCluster")
		}
		ipool, err := icluster.CreateIKubeNodePool(opts)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateIKubeNodePool")
		}
		err = db.SetExternalId(pool, userCred, ipool.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "db.SetExternalId")
		}
		err = cloudprovider.WaitStatus(ipool, api.KUBE_CLUSTER_STATUS_RUNNING, time.Second*30, time.Hour*1)
		if err != nil {
			return nil, errors.Wrapf(err, "wait node pool status timeout, current status: %s", icluster.GetStatus())
		}
		return nil, pool.SetStatus(ctx, userCred, api.KUBE_CLUSTER_STATUS_RUNNING, "")
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, secgroup *models.SSecurityGroup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iGroup, err := secgroup.GetISecurityGroup(ctx)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound || errors.Cause(err) == sql.ErrNoRows {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "GetISecurityGroup")
		}
		return nil, iGroup.Delete()
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	secgroup *models.SSecurityGroup,
	rules api.SSecgroupRuleResourceSet,
) error {

	vpcId := ""
	if len(secgroup.VpcId) > 0 {
		vpc, err := secgroup.GetVpc()
		if err != nil {
			return errors.Wrapf(err, "GetVpc")
		}
		vpcId = vpc.ExternalId
	}

	provider, err := secgroup.GetCloudprovider()
	if err != nil {
		return errors.Wrapf(err, "GetCloudprovider")
	}

	iRegion, err := secgroup.GetIRegion(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetIRegion")
	}

	opts := &cloudprovider.SecurityGroupCreateInput{
		Name:  secgroup.Name,
		Desc:  secgroup.Description,
		VpcId: vpcId,
	}
	opts.Tags, _ = secgroup.GetAllUserMetadata()

	opts.ProjectId, err = provider.SyncProject(ctx, userCred, secgroup.ProjectId)
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
			logclient.AddSimpleActionLog(secgroup, logclient.ACT_SYNC_CLOUD_PROJECT, err, userCred, false)
		}
	}

	iGroup, err := iRegion.CreateISecurityGroup(opts)
	if err != nil {
		return errors.Wrapf(err, "CreateISecurityGroup")
	}

	_, err = db.Update(secgroup, func() error {
		secgroup.ExternalId = iGroup.GetGlobalId()
		if len(iGroup.GetVpcId()) == 0 {
			secgroup.VpcId = ""
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SetExternalId")
	}

	for i := range rules {
		opts := cloudprovider.SecurityGroupRuleCreateOptions{
			Desc:      rules[i].Description,
			Direction: secrules.TSecurityRuleDirection(rules[i].Direction),
			Action:    secrules.TSecurityRuleAction(rules[i].Action),
			Protocol:  rules[i].Protocol,
			CIDR:      rules[i].CIDR,
			Ports:     rules[i].Ports,
		}
		_, err := iGroup.CreateRule(&opts)
		if err != nil {
			return errors.Wrapf(err, "CreateRule")
		}
	}

	iRules, err := iGroup.GetRules()
	if err != nil {
		return errors.Wrapf(err, "GetRules")
	}

	result := secgroup.SyncRules(ctx, userCred, iRules)
	if result.IsError() {
		return result.AllError()
	}
	secgroup.SetStatus(ctx, userCred, api.SECGROUP_STATUS_READY, "")
	return nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	for i := range input.Rules {
		rule := input.Rules[i]
		if !utils.IsInStringArray(rule.Action, []string{string(secrules.SecurityRuleAllow), string(secrules.SecurityRuleDeny)}) {
			return nil, httperrors.NewInputParameterError("invalid action %s", rule.Action)
		}
		if !utils.IsInStringArray(rule.Protocol, []string{
			secrules.PROTO_ANY,
			secrules.PROTO_UDP,
			secrules.PROTO_TCP,
			secrules.PROTO_ICMP,
		}) {
			return nil, httperrors.NewInputParameterError("invalid protocol %s", rule.Protocol)
		}

		if len(rule.Ports) > 0 {
			r := secrules.SecurityRule{}
			err := r.ParsePorts(rule.Ports)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid ports %s", rule.Ports)
			}
		}

		if len(rule.CIDR) > 0 && !regutils.MatchCIDR(rule.CIDR) && !regutils.MatchCIDR6(rule.CIDR) && !regutils.MatchIP4Addr(rule.CIDR) && !regutils.MatchIP6Addr(rule.CIDR) {
			return nil, httperrors.NewInputParameterError("invalid cidr %s", rule.CIDR)
		}
	}
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	if input.Action != nil {
		if !utils.IsInStringArray(*input.Action, []string{string(secrules.SecurityRuleAllow), string(secrules.SecurityRuleDeny)}) {
			return nil, httperrors.NewInputParameterError("invalid action %s", *input.Action)
		}
	}
	if input.Protocol != nil {
		if !utils.IsInStringArray(*input.Protocol, []string{
			secrules.PROTO_ANY,
			secrules.PROTO_UDP,
			secrules.PROTO_TCP,
			secrules.PROTO_ICMP,
		}) {
			return nil, httperrors.NewInputParameterError("invalid protocol %s", *input.Protocol)
		}
	}

	if input.Ports != nil {
		rule := secrules.SecurityRule{}
		err := rule.ParsePorts(*input.Ports)
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid ports %s", *input.Ports)
		}
	}

	if input.CIDR != nil && len(*input.CIDR) > 0 && !regutils.MatchCIDR(*input.CIDR) && !regutils.MatchIP4Addr(*input.CIDR) && !regutils.MatchCIDR6(*input.CIDR) && !regutils.MatchIP6Addr(*input.CIDR) {
		return nil, httperrors.NewInputParameterError("invalid cidr %s", *input.CIDR)
	}

	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) RequestPrepareSecurityGroups(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	secgroups []models.SSecurityGroup,
	vpc *models.SVpc,
	callback func(ids []string) error,
	task taskman.ITask,
) error {
	region, err := vpc.GetRegion()
	if err != nil {
		return errors.Wrapf(err, "GetRegion")
	}
	driver := region.GetDriver()
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		groupIds := []string{}
		for i := range secgroups {
			if secgroups[i].Id == api.SECGROUP_DEFAULT_ID {
				filter, err := driver.GetSecurityGroupFilter(vpc)
				if err != nil {
					return nil, errors.Wrapf(err, "GetSecurityGroupFilter")
				}
				group, err := vpc.GetDefaultSecurityGroup(ownerId, filter)
				if err != nil && errors.Cause(err) != sql.ErrNoRows {
					return nil, err
				}
				if gotypes.IsNil(group) {
					group, err = driver.CreateDefaultSecurityGroup(ctx, userCred, ownerId, vpc)
					if err != nil {
						return nil, errors.Wrapf(err, "CreateDefaultSecurityGroup")
					}
				}
				if !utils.IsInStringArray(group.Id, groupIds) {
					groupIds = append(groupIds, group.Id)
				}
				continue
			}
			if len(secgroups[i].ExternalId) > 0 && !utils.IsInStringArray(secgroups[i].Id, groupIds) {
				groupIds = append(groupIds, secgroups[i].Id)
			}
		}
		if callback != nil {
			return nil, callback(groupIds)
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("vpc_id", vpc.Id)
	}, nil
}

func (self *SManagedVirtualizationRegionDriver) CreateDefaultSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	vpc *models.SVpc,
) (*models.SSecurityGroup, error) {
	region, err := vpc.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	driver := region.GetDriver()
	newGroup := &models.SSecurityGroup{}
	newGroup.SetModelManager(models.SecurityGroupManager, newGroup)
	newGroup.Name = fmt.Sprintf("%s-%d", driver.GetDefaultSecurityGroupNamePrefix(), time.Now().Unix())
	newGroup.Description = "auto generage"
	// 部分云可能不需要vpcId, 创建完安全组后会自动置空
	newGroup.VpcId = vpc.Id
	newGroup.ManagerId = vpc.ManagerId
	newGroup.CloudregionId = vpc.CloudregionId
	newGroup.DomainId = ownerId.GetProjectDomainId()
	newGroup.ProjectId = ownerId.GetProjectId()
	newGroup.ProjectSrc = string(apis.OWNER_SOURCE_LOCAL)
	err = models.SecurityGroupManager.TableSpec().Insert(ctx, newGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "insert")
	}

	err = driver.RequestCreateSecurityGroup(ctx, userCred, newGroup, api.SSecgroupRuleResourceSet{})
	if err != nil {
		return nil, errors.Wrapf(err, "RequestCreateSecurityGroup")
	}
	return newGroup, nil
}

func (self *SManagedVirtualizationRegionDriver) ValidateCreateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, input *api.SSnapshotPolicyCreateInput) (*api.SSnapshotPolicyCreateInput, error) {
	if len(input.CloudproviderId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudprovider_id")
	}
	managerObj, err := validators.ValidateModel(ctx, userCred, models.CloudproviderManager, &input.CloudproviderId)
	if err != nil {
		return nil, err
	}
	input.ManagerId = input.CloudproviderId
	manager := managerObj.(*models.SCloudprovider)
	if manager.Provider != region.Provider {
		return nil, httperrors.NewConflictError("manager %s is not %s cloud", manager.Name, region.Provider)
	}
	return input, nil
}

func (self *SManagedVirtualizationRegionDriver) RequestCreateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, sp *models.SSnapshotPolicy, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRegion, err := sp.GetIRegion(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIRegion")
		}
		opts := &cloudprovider.SnapshotPolicyInput{
			Name:           sp.Name,
			Desc:           sp.Description,
			RetentionDays:  sp.RetentionDays,
			TimePoints:     sp.TimePoints,
			RepeatWeekdays: sp.RepeatWeekdays,
		}
		opts.Tags, _ = sp.GetAllUserMetadata()
		id, err := iRegion.CreateSnapshotPolicy(opts)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateSnapshotPolicy")
		}
		_, err = db.Update(sp, func() error {
			sp.ExternalId = id
			sp.Status = apis.STATUS_AVAILABLE
			return nil
		})
		return nil, err
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestDeleteSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, sp *models.SSnapshotPolicy, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iPolicy, err := sp.GetISnapshotPolicy(ctx)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows || errors.Cause(err) == cloudprovider.ErrNotFound {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "GetISnapshotPolicy")
		}

		err = iPolicy.Delete()
		if err != nil {
			return nil, errors.Wrapf(err, "Delete")
		}

		return nil, nil
	})
	return nil

}

func (self *SManagedVirtualizationRegionDriver) RequestSnapshotPolicyBindDisks(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, diskIds []string, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iPolicy, err := sp.GetISnapshotPolicy(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetISnapshotPolicy")
		}
		disks, err := sp.GetUnbindDisks(diskIds)
		if err != nil {
			return nil, errors.Wrapf(err, "GetUnbindDisks")
		}
		externalIds := []string{}
		for _, disk := range disks {
			if len(disk.ExternalId) > 0 && !utils.IsInStringArray(disk.ExternalId, externalIds) {
				externalIds = append(externalIds, disk.ExternalId)
			}
		}
		if len(externalIds) > 0 {
			err = iPolicy.ApplyDisks(externalIds)
			if err != nil {
				return nil, errors.Wrapf(err, "ApplyDisks %s", externalIds)
			}
			return nil, sp.BindDisks(ctx, disks)
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestSnapshotPolicyUnbindDisks(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, diskIds []string, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iPolicy, err := sp.GetISnapshotPolicy(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetISnapshotPolicy")
		}
		disks, err := sp.GetBindDisks(diskIds)
		if err != nil {
			return nil, errors.Wrapf(err, "GetBindDisks")
		}
		externalIds := []string{}
		for _, disk := range disks {
			if len(disk.ExternalId) > 0 && !utils.IsInStringArray(disk.ExternalId, externalIds) {
				externalIds = append(externalIds, disk.ExternalId)
			}
		}
		if len(externalIds) > 0 {
			err = iPolicy.CancelDisks(externalIds)
			if err != nil {
				return nil, errors.Wrapf(err, "CancelDisks %s", externalIds)
			}
			return nil, sp.UnbindDisks(diskIds)
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizationRegionDriver) RequestRemoteUpdateNetwork(ctx context.Context, userCred mcclient.TokenCredential, net *models.SNetwork, replaceTags bool, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iNet, err := net.GetINetwork(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "GetINetwork")
		}
		vpc, err := net.GetVpc()
		if err != nil {
			return nil, errors.Wrapf(err, "GetVpc")
		}
		oldTags, err := iNet.GetTags()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			return nil, errors.Wrap(err, "GetTags()")
		}
		tags, err := net.GetAllUserMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "GetAllUserMetadata")
		}
		tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
		err = cloudprovider.SetTags(ctx, iNet, vpc.ManagerId, tags, replaceTags)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			logclient.AddActionLogWithStartable(task, net, logclient.ACT_UPDATE_TAGS, err, userCred, false)
			return nil, errors.Wrap(err, "SetTags")
		}
		logclient.AddActionLogWithStartable(task, net, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, userCred, true)
		return nil, nil
	})
	return nil
}
