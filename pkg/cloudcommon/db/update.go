package db

import (
	"context"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

func Update(model IModel, updateFunc func() error) (sqlchemy.UpdateDiffs, error) {
	return model.GetModelManager().TableSpec().Update(model, updateFunc)
}

func UpdateWithLock(ctx context.Context, model IModel, updateFunc func() error) (sqlchemy.UpdateDiffs, error) {
	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	return Update(model, updateFunc)
}
