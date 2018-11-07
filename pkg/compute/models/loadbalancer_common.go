package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerStatus struct {
	RuntimeStatus string `width:"36" charset:"ascii" nullable:"false" default:"init" list:"user"`
}

type ILoadbalancerSubResourceManager interface {
	db.IModelManager

	// PreDeleteSubs is to be called by upper manager to PreDelete models managed by this one
	PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery)
}

// TODO
// notify on post create/update/delete
type SLoadbalancerNotifier struct{}

func (n *SLoadbalancerNotifier) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	return
}
