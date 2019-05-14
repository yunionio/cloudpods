package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerBackendManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
}

var LoadbalancerBackendManager *SLoadbalancerBackendManager

func init() {
	LoadbalancerBackendManager = &SLoadbalancerBackendManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerBackend{},
			"loadbalancerbackends_tbl",
			"loadbalancerbackend",
			"loadbalancerbackends",
		),
	}
}

type SLoadbalancerBackend struct {
	db.SVirtualResourceBase
	SManagedResourceBase
	SCloudregionResourceBase

	BackendGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	BackendId      string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	BackendType    string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	BackendRole    string `width:"36" charset:"ascii" nullable:"false" list:"user" default:"default" create:"optional"`
	Weight         int    `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	Address        string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	Port           int    `nullable:"false" list:"user" create:"required" update:"user"`
}

func (man *SLoadbalancerBackendManager) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	lbbs := []SLoadbalancerBackend{}
	db.FetchModelObjects(man, q, &lbbs)
	for _, lbb := range lbbs {
		lbb.DoPendingDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerBackendManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "backend_group", ModelKeyword: "loadbalancerbackendgroup", ProjectId: userProjId},
		{Key: "backend", ModelKeyword: "server", ProjectId: userProjId}, // NOTE extend this when new backend_type was added
		{Key: "cloudregion", ModelKeyword: "cloudregion", ProjectId: userProjId},
		{Key: "manager", ModelKeyword: "cloudprovider", ProjectId: userProjId},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SLoadbalancerBackendManager) ValidateBackendVpc(lb *SLoadbalancer, guest *SGuest, backendgroup *SLoadbalancerBackendGroup) error {
	region := lb.GetRegion()
	if region == nil {
		return httperrors.NewResourceNotFoundError("failed to find region for loadbalancer %s", lb.Name)
	}
	requireStatus := region.GetDriver().GetBackendStatusForAdd()
	if !utils.IsInStringArray(guest.Status, requireStatus) {
		return httperrors.NewUnsupportOperationError("%s requires the virtual machine state to be %s before it can be added backendgroup, but current state of the virtual machine is %s", region.GetDriver().GetProvider(), requireStatus, guest.Status)
	}
	vpc, err := guest.GetVpc()
	if err != nil {
		return err
	}
	if len(lb.VpcId) > 0 {
		if vpc.Id != lb.VpcId {
			return fmt.Errorf("guest %s(%s) vpc %s(%s) not same as loadbalancer vpc %s", guest.Name, guest.Id, vpc.Name, vpc.Id, lb.VpcId)
		}
		return nil
	}
	backends, err := backendgroup.GetBackends()
	if err != nil {
		return err
	}
	for _, backend := range backends {
		_server, err := GuestManager.FetchById(backend.BackendId)
		if err != nil {
			return err
		}
		server := _server.(*SGuest)
		_vpc, err := server.GetVpc()
		if err != nil {
			return err
		}
		if _vpc.Id != vpc.Id {
			return fmt.Errorf("guest %s(%s) vpc %s(%s) not same as vpc %s(%s)", guest.Name, guest.Id, vpc.Name, vpc.Id, _vpc.Name, _vpc.Id)
		}
		if _server.GetId() == guest.Id {
			return fmt.Errorf("guest %s(%s) is already in the backendgroup %s(%s)", guest.Name, guest.Id, backendgroup.Name, backendgroup.Id)
		}
	}
	return nil
}

func (man *SLoadbalancerBackendManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerProjId)
	backendTypeV := validators.NewStringChoicesValidator("backend_type", api.LB_BACKEND_TYPES)
	keyV := map[string]validators.IValidator{
		"backend_group": backendGroupV,
		"backend_type":  backendTypeV,
		"weight":        validators.NewRangeValidator("weight", 1, 256).Default(1),
		"port":          validators.NewPortValidator("port"),
	}
	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	backendGroup := backendGroupV.Model.(*SLoadbalancerBackendGroup)
	lb := backendGroup.GetLoadbalancer()
	data.Set("manager_id", jsonutils.NewString(lb.ManagerId))
	data.Set("cloudregion_id", jsonutils.NewString(lb.CloudregionId))
	backendType := backendTypeV.Value
	var baseName string
	var backendV *validators.ValidatorModelIdOrName
	switch backendType {
	case api.LB_BACKEND_GUEST:
		backendV = validators.NewModelIdOrNameValidator("backend", "server", ownerProjId)
		err := backendV.Validate(data)
		if err != nil {
			return nil, err
		}
		guest := backendV.Model.(*SGuest)
		baseName = guest.Name
		err = man.ValidateBackendVpc(lb, guest, backendGroup)
		if err != nil {
			return nil, err
		}
	case api.LB_BACKEND_HOST:
		if !db.IsAdminAllowCreate(userCred, man) {
			return nil, fmt.Errorf("only sysadmin can specify host as backend")
		}
		backendV = validators.NewModelIdOrNameValidator("backend", "host", userCred.GetProjectId())
		err := backendV.Validate(data)
		if err != nil {
			return nil, err
		}
		host := backendV.Model.(*SHost)
		{
			if len(host.AccessIp) == 0 {
				return nil, fmt.Errorf("host %s has no access ip", host.GetId())
			}
			data.Set("address", jsonutils.NewString(host.AccessIp))
		}
		baseName = host.Name
	default:
		return nil, fmt.Errorf("internal error: unexpected backend type %s", backendType)
	}
	// name it
	//
	// NOTE it's okay for name to be not unique.
	//
	//  - Mix in loadbalancer name if needed
	//  - Use name from input query
	name := fmt.Sprintf("%s-%s-%s", backendGroup.Name, backendType, baseName)
	data.Set("name", jsonutils.NewString(name))
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data); err != nil {
		return nil, err
	}
	region := lb.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer %s", lb.Name)
	}
	return region.GetDriver().ValidateCreateLoadbalancerBackendData(ctx, userCred, data, backendType, lb, backendGroup, backendV.Model)
}

func (lbb *SLoadbalancerBackend) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbb *SLoadbalancerBackend) GetLoadbalancerBackendGroup() *SLoadbalancerBackendGroup {
	backendgroup, err := LoadbalancerBackendGroupManager.FetchById(lbb.BackendGroupId)
	if err != nil {
		log.Errorf("failed to find backendgroup for backend %s", lbb.Name)
		return nil
	}
	return backendgroup.(*SLoadbalancerBackendGroup)
}

func (lbb *SLoadbalancerBackend) GetGuest() *SGuest {
	guest, err := GuestManager.FetchById(lbb.BackendId)
	if err != nil {
		return nil
	}
	return guest.(*SGuest)
}

func (lbb *SLoadbalancerBackend) GetRegion() *SCloudregion {
	if backendgroup := lbb.GetLoadbalancerBackendGroup(); backendgroup != nil {
		return backendgroup.GetRegion()
	}
	return nil
}

func (lbb *SLoadbalancerBackend) GetIRegion() (cloudprovider.ICloudRegion, error) {
	if backendgroup := lbb.GetLoadbalancerBackendGroup(); backendgroup != nil {
		return backendgroup.GetIRegion()
	}
	return nil, fmt.Errorf("failed to find region for backend %s", lbb.Name)
}

func (man *SLoadbalancerBackendManager) GetGuestAddress(guest *SGuest) (string, error) {
	gns, err := guest.GetNetworks("")
	if err != nil || len(gns) == 0 {
		return "", fmt.Errorf("guest %s has no network attached", guest.GetId())
	}
	for _, gn := range gns {
		if !gn.IsExit() {
			return gn.IpAddr, nil
		}
	}
	return "", fmt.Errorf("guest %s has no intranet address attached", guest.GetId())
}

func (lbb *SLoadbalancerBackend) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	keyV := map[string]validators.IValidator{
		"weight": validators.NewRangeValidator("weight", 1, 256),
		"port":   validators.NewPortValidator("port"),
	}
	for _, v := range keyV {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	_, err := lbb.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	region := lbb.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to found region for loadbalancer backend %s", lbb.Name)
	}
	lbbg := lbb.GetLoadbalancerBackendGroup()
	if lbbg == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to found backendgroup for backend %s(%s)", lbb.Name, lbb.Id)
	}
	return region.GetDriver().ValidateUpdateLoadbalancerBackendData(ctx, userCred, data, lbbg)
}

func (lbb *SLoadbalancerBackend) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbb.SVirtualResourceBase.PostUpdate(ctx, userCred, query, data)
	if data.Contains("port") || data.Contains("weight") {
		lbb.StartLoadBalancerBackendSyncTask(ctx, userCred, "")
	}
}

func (lbb *SLoadbalancerBackend) StartLoadBalancerBackendSyncTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	lbb.SetStatus(userCred, api.LB_SYNC_CONF, "")
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendSyncTask", lbb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbb *SLoadbalancerBackend) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbb.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	lbb.SetStatus(userCred, api.LB_CREATING, "")
	if err := lbb.StartLoadBalancerBackendCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("Failed to create loadbalancer backend error: %v", err)
	}
}

func (lbb *SLoadbalancerBackend) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbb.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	providerInfo := lbb.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if providerInfo != nil {
		extra.Update(providerInfo)
	}
	_guest, err := GuestManager.FetchById(lbb.BackendId)
	if err != nil {
		log.Errorf("failed to find guest for loadbalancer backend %s(%s)", lbb.Name, lbb.Id)
		return extra
	}
	guest := _guest.(*SGuest)
	vpc, err := guest.GetVpc()
	if err != nil {
		log.Errorf("failed to find vpc for guest %s(%s)", guest.Name, guest.Id)
		return extra
	}
	extra.Set("vpc_id", jsonutils.NewString(vpc.Id))
	regionInfo := lbb.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	return extra
}

func (lbb *SLoadbalancerBackend) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbb.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbb *SLoadbalancerBackend) StartLoadBalancerBackendCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendCreateTask", lbb, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbb *SLoadbalancerBackend) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (lbb *SLoadbalancerBackend) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbb, "purge")
}

func (lbb *SLoadbalancerBackend) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbb.StartLoadBalancerBackendDeleteTask(ctx, userCred, parasm, "")
}

func (lbb *SLoadbalancerBackend) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbb.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return lbb.StartLoadBalancerBackendDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbb *SLoadbalancerBackend) StartLoadBalancerBackendDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendDeleteTask", lbb, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (man *SLoadbalancerBackendManager) getLoadbalancerBackendsByLoadbalancerBackendgroup(loadbalancerBackendgroup *SLoadbalancerBackendGroup) ([]SLoadbalancerBackend, error) {
	loadbalancerBackends := []SLoadbalancerBackend{}
	q := man.Query().Equals("backend_group_id", loadbalancerBackendgroup.Id)
	q = q.IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &loadbalancerBackends); err != nil {
		return nil, err
	}
	return loadbalancerBackends, nil
}

func (lbb *SLoadbalancerBackend) ValidateDeleteCondition(ctx context.Context) error {
	if err := lbb.SVirtualResourceBase.ValidateDeleteCondition(ctx); err != nil {
		return err
	}
	region := lbb.GetRegion()
	if region == nil {
		return nil
	}
	return region.GetDriver().ValidateDeleteLoadbalancerBackendCondition(ctx, lbb)
}

func (man *SLoadbalancerBackendManager) SyncLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, loadbalancerBackendgroup *SLoadbalancerBackendGroup, lbbs []cloudprovider.ICloudLoadbalancerBackend, syncRange *SSyncRange) compare.SyncResult {
	syncOwnerId := provider.ProjectId

	lockman.LockClass(ctx, man, syncOwnerId)
	defer lockman.ReleaseClass(ctx, man, syncOwnerId)

	syncResult := compare.SyncResult{}

	dbLbbs, err := man.getLoadbalancerBackendsByLoadbalancerBackendgroup(loadbalancerBackendgroup)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SLoadbalancerBackend{}
	commondb := []SLoadbalancerBackend{}
	commonext := []cloudprovider.ICloudLoadbalancerBackend{}
	added := []cloudprovider.ICloudLoadbalancerBackend{}

	err = compare.CompareSets(dbLbbs, lbbs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerBackend(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerBackend(ctx, userCred, commonext[i], provider.ProjectId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		local, err := man.newFromCloudLoadbalancerBackend(ctx, userCred, loadbalancerBackendgroup, added[i], syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func (lbb *SLoadbalancerBackend) constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend) error {
	// lbb.Name = extLoadbalancerBackend.GetName()
	lbb.Status = extLoadbalancerBackend.GetStatus()

	lbb.Weight = extLoadbalancerBackend.GetWeight()
	lbb.Port = extLoadbalancerBackend.GetPort()

	lbb.BackendType = extLoadbalancerBackend.GetBackendType()
	lbb.BackendId = extLoadbalancerBackend.GetBackendId()
	lbb.BackendRole = extLoadbalancerBackend.GetBackendRole()

	instance, err := GuestManager.FetchByExternalId(lbb.BackendId)
	if err != nil {
		return err
	}
	guest := instance.(*SGuest)
	lbb.BackendId = guest.Id
	address, err := LoadbalancerBackendManager.GetGuestAddress(guest)
	if err != nil {
		return err
	}
	lbb.Address = address
	return nil
}

func (lbb *SLoadbalancerBackend) syncRemoveCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbb)
	defer lockman.ReleaseObject(ctx, lbb)

	err := lbb.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = lbb.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		// err = lbb.MarkPendingDelete(userCred)
		err = lbb.DoPendingDelete(ctx, userCred)
	}
	return err
}

func (lbb *SLoadbalancerBackend) SyncWithCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, projectId string) error {
	diff, err := db.UpdateWithLock(ctx, lbb, func() error {
		return lbb.constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend)
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogSyncUpdate(lbb, diff, userCred)

	SyncCloudProject(userCred, lbb, projectId, extLoadbalancerBackend, lbb.ManagerId)

	return nil
}

func (man *SLoadbalancerBackendManager) newFromCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, loadbalancerBackendgroup *SLoadbalancerBackendGroup, extLoadbalancerBackend cloudprovider.ICloudLoadbalancerBackend, projectId string) (*SLoadbalancerBackend, error) {
	lbb := &SLoadbalancerBackend{}
	lbb.SetModelManager(man)

	lbb.BackendGroupId = loadbalancerBackendgroup.Id
	lbb.ExternalId = extLoadbalancerBackend.GetGlobalId()

	lbb.CloudregionId = loadbalancerBackendgroup.CloudregionId
	lbb.ManagerId = loadbalancerBackendgroup.ManagerId

	lbb.Name = db.GenerateName(man, projectId, extLoadbalancerBackend.GetName())

	if err := lbb.constructFieldsFromCloudLoadbalancerBackend(extLoadbalancerBackend); err != nil {
		return nil, err
	}

	err := man.TableSpec().Insert(lbb)

	if err != nil {
		return nil, err
	}

	SyncCloudProject(userCred, lbb, projectId, extLoadbalancerBackend, loadbalancerBackendgroup.ManagerId)

	db.OpsLog.LogEvent(lbb, db.ACT_CREATE, lbb.GetShortDesc(ctx), userCred)

	return lbb, nil
}

func (manager *SLoadbalancerBackendManager) InitializeData() error {
	backends := []SLoadbalancerBackend{}
	q := manager.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
	if err := db.FetchModelObjects(manager, q, &backends); err != nil {
		return err
	}
	for i := 0; i < len(backends); i++ {
		backend := &backends[i]
		if group := backend.GetLoadbalancerBackendGroup(); group != nil && len(group.CloudregionId) > 0 {
			_, err := db.Update(backend, func() error {
				backend.CloudregionId = group.CloudregionId
				backend.ManagerId = group.ManagerId
				return nil
			})
			if err != nil {
				log.Errorf("failed to update loadbalancer backend %s cloudregion_id", group.Name)
			}
		}
	}
	return nil
}
