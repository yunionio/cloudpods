package models

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerBackendGroupManager struct {
	db.SVirtualResourceBaseManager
}

var LoadbalancerBackendGroupManager *SLoadbalancerBackendGroupManager

func init() {
	LoadbalancerBackendGroupManager = &SLoadbalancerBackendGroupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerBackendGroup{},
			"loadbalancerbackendgroups_tbl",
			"loadbalancerbackendgroup",
			"loadbalancerbackendgroups",
		),
	}
}

type SLoadbalancerBackendGroup struct {
	db.SVirtualResourceBase

	Type           string `width:"36" charset:"ascii" nullable:"false" list:"user" default:"normal" create:"optional"`
	LoadbalancerId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
}

func (man *SLoadbalancerBackendGroupManager) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerBackendGroup{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.PreDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerBackendGroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		{Key: "loadbalancer", ModelKeyword: "loadbalancer", ProjectId: userProjId},
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SLoadbalancerBackendGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", ownerProjId)
	err := lbV.Validate(data)
	if err != nil {
		return nil, err
	}
	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data); err != nil {
		return nil, err
	}
	lb := lbV.Model.(*SLoadbalancer)
	backends := []cloudprovider.SLoadbalancerBackend{}
	for idx := 0; data.Contains(fmt.Sprintf("backend.%d", idx)); idx++ {
		_backend, _ := data.GetString(fmt.Sprintf("backend.%d", idx))
		backend := cloudprovider.SLoadbalancerBackend{BackendType: LB_BACKEND_GUEST, Index: idx, BackendRole: "default"}
		for _, info := range strings.Split(_backend, ",") {
			value := strings.Split(info, ":")
			if len(value) != 2 {
				return nil, httperrors.NewInputParameterError("Unknown backend info %s", info)
			}
			switch strings.ToLower(value[0]) {
			case "id", "name":
				backend.ID = value[1]
			case "backend_type", "backendtype", "type":
				backend.BackendType = value[1]
			case "weight":
				v, err := strconv.Atoi(value[1])
				if err != nil || v < 0 || v > 256 {
					return nil, httperrors.NewInputParameterError("weight %s not support, only support range 0 ~ 100")
				}
				backend.Weight = v
			case "port":
				v, err := strconv.Atoi(value[1])
				if err != nil || v < 1 || v > 65535 {
					return nil, httperrors.NewInputParameterError("port %s not support, only support range 1 ~ 65535")
				}
				backend.Port = v
			}
		}
		if len(backend.ID) == 0 {
			return nil, httperrors.NewMissingParameterError("Missing backend id or name params")
		}
		if backend.Port < 1 || backend.Port > 65535 {
			return nil, httperrors.NewInputParameterError("port %s not support, only support range 1 ~ 65535")
		}
		switch backend.BackendType {
		case LB_BACKEND_GUEST:
			_guest, err := GuestManager.FetchByIdOrName(userCred, backend.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("failed to find guest %s", backend.ID)
				}
				return nil, httperrors.NewGeneralError(err)
			}
			guest := _guest.(*SGuest)
			host := guest.GetHost()
			if host == nil {
				return nil, fmt.Errorf("error getting host of guest %s", guest.Name)
			}
			if host.ZoneId != lb.ZoneId {
				return nil, fmt.Errorf("zone of host %q (%s) != zone of loadbalancer %q (%s)",
					host.Name, host.ZoneId, lb.Name, lb.ZoneId)
			}
			backend.ID = guest.Id
			backend.Name = guest.Name
			backend.ExternalID = guest.ExternalId
			address, err := LoadbalancerBackendManager.getGuestAddress(guest)
			if err != nil {
				return nil, err
			}
			backend.Address = address
		case LB_BACKEND_HOST:
			if !db.IsAdminAllowCreate(userCred, man) {
				return nil, httperrors.NewForbiddenError("only sysadmin can specify host as backend")
			}
			_host, err := HostManager.FetchByIdOrName(userCred, backend.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("failed to find host %s", backend.ID)
				}
				return nil, httperrors.NewGeneralError(err)
			}
			host := _host.(*SHost)
			backend.ID = host.Id
			backend.Name = host.Name
			backend.ExternalID = host.ExternalId
			backend.Address = host.AccessIp
		default:
			return nil, httperrors.NewInputParameterError("unexpected backend type %s", backend.BackendType)
		}
		backends = append(backends, backend)
	}
	data.Set("backends", jsonutils.Marshal(backends))
	return lb.GetRegion().GetDriver().ValidateCreateLoadbalancerBackendGroupData(ctx, userCred, data, backends)
}

func (lbbg *SLoadbalancerBackendGroup) GetLoadbalancer() *SLoadbalancer {
	lb, err := LoadbalancerManager.FetchById(lbbg.LoadbalancerId)
	if err != nil {
		return nil
	}
	return lb.(*SLoadbalancer)
}

func (llbg *SLoadbalancerBackendGroup) GetRegion() *SCloudregion {
	if loadbalancer := llbg.GetLoadbalancer(); loadbalancer != nil {
		return loadbalancer.GetRegion()
	}
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) GetIRegion() (cloudprovider.ICloudRegion, error) {
	if loadbalancer := lbbg.GetLoadbalancer(); loadbalancer != nil {
		return loadbalancer.GetIRegion()
	}
	return nil, fmt.Errorf("failed to find loadbalancer for backendgroup %s", lbbg.Name)
}

func (lbbg *SLoadbalancerBackendGroup) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbbg *SLoadbalancerBackendGroup) ValidateDeleteCondition(ctx context.Context) error {
	men := []db.IModelManager{
		LoadbalancerManager,
		LoadbalancerListenerManager,
		LoadbalancerListenerRuleManager,
	}
	lbbgId := lbbg.Id
	for _, man := range men {
		t := man.TableSpec().Instance()
		pdF := t.Field("pending_deleted")
		n := t.Query().
			Equals("backend_group_id", lbbgId).
			Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
			Count()
		if n > 0 {
			return fmt.Errorf("backend group %s is still referred to by %d %s",
				lbbgId, n, man.KeywordPlural())
		}
	}
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lbbg.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	{
		lb, err := LoadbalancerManager.FetchById(lbbg.LoadbalancerId)
		if err != nil {
			log.Errorf("loadbalancer backend group %s(%s): fetch loadbalancer (%s) error: %s",
				lbbg.Name, lbbg.Id, lbbg.LoadbalancerId, err)
			return extra
		}
		extra.Set("loadbalancer", jsonutils.NewString(lb.GetName()))
	}
	return extra
}

func (lbbg *SLoadbalancerBackendGroup) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra := lbbg.GetCustomizeColumns(ctx, userCred, query)
	return extra, nil
}

func (lbbg *SLoadbalancerBackendGroup) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbbg.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	params := jsonutils.NewDict()
	backends, _ := data.Get("backends")
	if backends != nil {
		params.Add(backends, "backends")
	}
	lbbg.SetStatus(userCred, LB_STATUS_CREATING, "")
	if err := lbbg.StartLoadBalancerBackendGroupCreateTask(ctx, userCred, params, ""); err != nil {
		log.Errorf("Failed to create loadbalancer backendgroup error: %v", err)
	}
}

func (lbbg *SLoadbalancerBackendGroup) StartLoadBalancerBackendGroupCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerLoadbalancerBackendGroupCreateTask", lbbg, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

// func (lbbg *SLoadbalancerBackendGroup) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
// 	lbbg.DoPendingDelete(ctx, userCred)
// 	lbbg.PreDeleteSubs(ctx, userCred)
// }

func (lbbg *SLoadbalancerBackendGroup) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential) {
	lbbg.DoCancelPendingDelete(ctx, userCred)
	subMan := LoadbalancerBackendManager
	ownerProjId := lbbg.GetOwnerProjectId()

	lockman.LockClass(ctx, subMan, ownerProjId)
	defer lockman.ReleaseClass(ctx, subMan, ownerProjId)
	q := subMan.Query().Equals("backend_group_id", lbbg.Id)
	subMan.PreDeleteSubs(ctx, userCred, q)
}

func (lbbg *SLoadbalancerBackendGroup) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbbg, "purge")
}

func (lbbg *SLoadbalancerBackendGroup) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbbg.StartLoadBalancerBackendGroupDeleteTask(ctx, userCred, parasm, "")
}

func (lbbg *SLoadbalancerBackendGroup) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbbg.SetStatus(userCred, LB_STATUS_DELETING, "")
	return lbbg.StartLoadBalancerBackendGroupDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbbg *SLoadbalancerBackendGroup) StartLoadBalancerBackendGroupDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerBackendGroupDeleteTask", lbbg, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbbg *SLoadbalancerBackendGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (man *SLoadbalancerBackendGroupManager) getLoadbalancerBackendgroupsByLoadbalancer(lb *SLoadbalancer) ([]SLoadbalancerBackendGroup, error) {
	lbbgs := []SLoadbalancerBackendGroup{}
	q := man.Query().Equals("loadbalancer_id", lb.Id)
	if err := db.FetchModelObjects(man, q, &lbbgs); err != nil {
		log.Errorf("failed to get lbbgs for lb: %s error: %v", lb.Name, err)
		return nil, err
	}
	return lbbgs, nil
}

func (man *SLoadbalancerBackendGroupManager) SyncLoadbalancerBackendgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lb *SLoadbalancer, lbbgs []cloudprovider.ICloudLoadbalancerBackendGroup, syncRange *SSyncRange) ([]SLoadbalancerBackendGroup, []cloudprovider.ICloudLoadbalancerBackendGroup, compare.SyncResult) {
	localLbgs := []SLoadbalancerBackendGroup{}
	remoteLbbgs := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	syncResult := compare.SyncResult{}

	dbLbbgs, err := man.getLoadbalancerBackendgroupsByLoadbalancer(lb)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := []SLoadbalancerBackendGroup{}
	commondb := []SLoadbalancerBackendGroup{}
	commonext := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	added := []cloudprovider.ICloudLoadbalancerBackendGroup{}

	err = compare.CompareSets(dbLbbgs, lbbgs, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].ValidateDeleteCondition(ctx)
		if err != nil { // cannot delete
			err = removed[i].SetStatus(userCred, LB_STATUS_UNKNOWN, "sync to delete")
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		} else {
			err = removed[i].Delete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerBackendgroup(ctx, userCred, lb, commonext[i], provider.ProjectId, syncRange.ProjectSync)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localLbgs = append(localLbgs, commondb[i])
			remoteLbbgs = append(remoteLbbgs, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		new, err := man.newFromCloudLoadbalancerBackendgroup(ctx, userCred, lb, added[i], provider.ProjectId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localLbgs = append(localLbgs, *new)
			remoteLbbgs = append(remoteLbbgs, added[i])
			syncResult.Add()
		}
	}
	return localLbgs, remoteLbbgs, syncResult
}

func (lbbg *SLoadbalancerBackendGroup) constructFieldsFromCloudBackendgroup(lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup) {
	lbbg.Name = extLoadbalancerBackendgroup.GetName()
	lbbg.Type = extLoadbalancerBackendgroup.GetType()
}

func (lbbg *SLoadbalancerBackendGroup) SyncWithCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, projectId string, projectSync bool) error {
	_, err := lbbg.GetModelManager().TableSpec().Update(lbbg, func() error {

		lbbg.constructFieldsFromCloudBackendgroup(lb, extLoadbalancerBackendgroup)
		if projectSync && len(projectId) > 0 {
			lbbg.ProjectId = projectId
		}

		if extLoadbalancerBackendgroup.IsDefault() {
			_, err := lb.GetModelManager().TableSpec().Update(lb, func() error {
				lb.BackendGroupId = lbbg.Id
				return nil
			})
			if err != nil {
				log.Errorf("failed to set backendgroup id for lb %s error: %v", lb.Name, err)
			}
		}

		return nil
	})
	return err
}

func (man *SLoadbalancerBackendGroupManager) newFromCloudLoadbalancerBackendgroup(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, extLoadbalancerBackendgroup cloudprovider.ICloudLoadbalancerBackendGroup, projectId string) (*SLoadbalancerBackendGroup, error) {
	lbbg := &SLoadbalancerBackendGroup{}
	lbbg.SetModelManager(man)

	lbbg.LoadbalancerId = lb.Id
	lbbg.ExternalId = extLoadbalancerBackendgroup.GetGlobalId()

	lbbg.constructFieldsFromCloudBackendgroup(lb, extLoadbalancerBackendgroup)

	lbbg.ProjectId = userCred.GetProjectId()
	if len(projectId) > 0 {
		lbbg.ProjectId = projectId
	}
	err := man.TableSpec().Insert(lbbg)
	if err != nil {
		return nil, err
	}

	if extLoadbalancerBackendgroup.IsDefault() {
		_, err := lb.GetModelManager().TableSpec().Update(lb, func() error {
			lb.BackendGroupId = lbbg.Id
			return nil
		})
		if err != nil {
			log.Errorf("failed to set backendgroup id for lb %s error: %v", lb.Name, err)
		}
	}
	return lbbg, nil
}

func (man *SLoadbalancerBackendGroupManager) InitializeData() error {
	backendgroups := []SLoadbalancerBackendGroup{}
	q := man.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("type")))
	if err := db.FetchModelObjects(man, q, &backendgroups); err != nil {
		log.Errorf("failed fetching backendgroups with empty type error: %v", err)
		return err
	}
	for i := 0; i < len(backendgroups); i++ {
		_, err := man.TableSpec().Update(&backendgroups[i], func() error {
			backendgroups[i].Type = LB_BACKENDGROUP_TYPE_NORMAL
			return nil
		})
		if err != nil {
			log.Errorf("failed setting backendgroup %s(%s) type error: %v", backendgroups[i].Name, backendgroups[i].Id, err)
		}
	}
	return nil
}
