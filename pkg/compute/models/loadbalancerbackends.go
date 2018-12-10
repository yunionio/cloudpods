package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerBackendManager struct {
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

	BackendGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	BackendId      string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	BackendType    string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	Weight         int    `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	Address        string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	Port           int    `nullable:"false" list:"user" create:"required" update:"user"`
}

func (man *SLoadbalancerBackendManager) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerBackend{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.PreDelete(ctx, userCred)
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
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (man *SLoadbalancerBackendManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerProjId)
	backendTypeV := validators.NewStringChoicesValidator("backend_type", LB_BACKEND_TYPES)
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
	backendType := backendTypeV.Value
	var baseName string
	switch backendType {
	case LB_BACKEND_GUEST:
		backendV := validators.NewModelIdOrNameValidator("backend", "server", ownerProjId)
		err := backendV.Validate(data)
		if err != nil {
			return nil, err
		}
		guest := backendV.Model.(*SGuest)
		{
			// guest zone must match that of loadbalancer's
			host := guest.GetHost()
			if host == nil {
				return nil, fmt.Errorf("error getting host of guest %s", guest.GetId())
			}
			lb := backendGroup.GetLoadbalancer()
			if lb == nil {
				return nil, fmt.Errorf("error loadbalancer of backend group %s", backendGroup.GetId())
			}
			if host.ZoneId != lb.ZoneId {
				return nil, fmt.Errorf("zone of host %q (%s) != zone of loadbalancer %q (%s)",
					host.Name, host.ZoneId, lb.Name, lb.ZoneId)
			}
		}
		{
			// get guest intranet address
			//
			// NOTE add address hint (cidr) if needed
			gns := guest.GetNetworks()
			if len(gns) == 0 {
				return nil, fmt.Errorf("guest %s has no network attached", guest.GetId())
			}
			var address string
			for _, gn := range gns {
				if !gn.IsExit() {
					address = gn.IpAddr
					break
				}
			}
			if len(address) == 0 {
				return nil, fmt.Errorf("guest %s has no intranet address attached", guest.GetId())
			}
			data.Set("address", jsonutils.NewString(address))
		}
		baseName = guest.Name
	case LB_BACKEND_HOST:
		if !db.IsAdminAllowCreate(userCred, man) {
			return nil, fmt.Errorf("only sysadmin can specify host as backend")
		}
		backendV := validators.NewModelIdOrNameValidator("backend", "host", userCred.GetProjectId())
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
	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (lbb *SLoadbalancerBackend) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
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
	return lbb.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lbb *SLoadbalancerBackend) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	lbb.DoPendingDelete(ctx, userCred)
}

func (lbb *SLoadbalancerBackend) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (man *SLoadbalancerBackendManager) getLoadbalancerBackendsByLoadbalancerBackendgroup(lbbg *SLoadbalancerBackendGroup) ([]SLoadbalancerBackend, error) {
	lbbs := []SLoadbalancerBackend{}
	q := man.Query().Equals("backend_group_id", lbbg.Id)
	if err := db.FetchModelObjects(man, q, &lbbg); err != nil {
		return nil, err
	}
	return lbbs, nil
}

func (man *SLoadbalancerBackendManager) SyncLoadbalancerBackends(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, lbbg *SLoadbalancerBackendGroup, lbbs []cloudprovider.ICloudLoadbalancerBackend, syncRange *SSyncRange) compare.SyncResult {
	syncResult := compare.SyncResult{}

	dbLbbs, err := man.getLoadbalancerBackendsByLoadbalancerBackendgroup(lbbg)
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
		err = commondb[i].SyncWithCloudLoadbalancerBackend(ctx, userCred, commonext[i], provider.ProjectId, syncRange.ProjectSync)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		_, err := man.newFromCloudLoadbalancerBackend(ctx, userCred, lbbg, added[i], provider.ProjectId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

func (lbb *SLoadbalancerBackend) SyncWithCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, extLbb cloudprovider.ICloudLoadbalancerBackend, projectId string, projectSync bool) error {
	_, err := LoadbalancerBackendManager.TableSpec().Update(lbb, func() error {
		lbb.Name = extLbb.GetName()
		lbb.Status = extLbb.GetStatus()

		lbb.Weight = extLbb.GetWeight()
		lbb.Address = extLbb.GetAddress()
		lbb.Port = extLbb.GetPort()

		lbb.BackendType = extLbb.GetBackendType()
		lbb.BackendId = extLbb.GetBackendId()

		if projectSync && len(projectId) > 0 {
			lbb.ProjectId = projectId
		}

		return nil
	})
	return err
}

func (man *SLoadbalancerBackendManager) newFromCloudLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbbg *SLoadbalancerBackendGroup, extLbb cloudprovider.ICloudLoadbalancerBackend, projectId string) (*SLoadbalancerBackend, error) {
	lbb := SLoadbalancerBackend{}
	lbb.SetModelManager(man)

	lbb.BackendGroupId = lbbg.Id
	lbb.Name = extLbb.GetName()
	lbb.ExternalId = extLbb.GetGlobalId()
	lbb.Status = extLbb.GetStatus()

	lbb.Weight = extLbb.GetWeight()
	lbb.Address = extLbb.GetAddress()
	lbb.Port = extLbb.GetPort()
	lbb.BackendType = extLbb.GetBackendType()
	lbb.BackendId = extLbb.GetBackendId()

	lbb.ProjectId = userCred.GetProjectId()
	if len(projectId) > 0 {
		lbb.ProjectId = projectId
	}
	return &lbb, man.TableSpec().Insert(&lbb)
}
