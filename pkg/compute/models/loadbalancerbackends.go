package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
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
	{
		backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", userProjId)
		backendGroupV.Optional(true)
		q, err = backendGroupV.QueryFilter(q, data)
		if err != nil {
			return nil, err
		}
	}
	{
		// NOTE extend this when new backend_type was added
		backendV := validators.NewModelIdOrNameValidator("backend", "server", userProjId)
		backendV.Optional(true)
		q, err = backendV.QueryFilter(q, data)
		if err != nil {
			return nil, err
		}
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
				return nil, fmt.Errorf("host zone (%s) != loadbalancer %q zone (%s)",
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
		if !userCred.IsSystemAdmin() {
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
