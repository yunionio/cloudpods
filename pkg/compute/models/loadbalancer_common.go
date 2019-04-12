package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerStatus struct {
	RuntimeStatus string `width:"36" charset:"ascii" nullable:"false" default:"init" list:"user"`
}

type ILoadbalancerSubResourceManager interface {
	db.IModelManager

	// pendingDeleteSubs applies pending delete to sub resources
	pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery)
}

// TODO
// notify on post create/update/delete
type SLoadbalancerNotifier struct{}

func (n *SLoadbalancerNotifier) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	return
}

type SLoadbalancerLogSkipper struct{}

func (lls SLoadbalancerLogSkipper) skipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	data, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return false
	}
	if val, _ := data.GetString(consts.LBAGENT_QUERY_ORIG_KEY); val != consts.LBAGENT_QUERY_ORIG_VAL {
		return false
	}
	return true
}

func (lls SLoadbalancerLogSkipper) ListSkipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return lls.skipLog(ctx, userCred, query)
}

func (lls SLoadbalancerLogSkipper) GetSkipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return lls.skipLog(ctx, userCred, query)
}
