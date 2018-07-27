package db

import (
	"context"
	"database/sql"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/sqlchemy"
)

func fetchById(manager IModelManager, idStr string) (IModel, error) {
	q := manager.Query()
	q = manager.FilterById(q, idStr)
	count := q.Count()
	if count == 1 {
		obj, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func fetchByName(manager IModelManager, ownerProjId string, idStr string) (IModel, error) {
	q := manager.Query()
	q = manager.FilterByName(q, idStr)
	q = manager.FilterByOwner(q, ownerProjId)
	count := q.Count()
	if count == 1 {
		obj, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func fetchByIdOrName(manager IModelManager, ownerProjId string, idStr string) (IModel, error) {
	obj, err := fetchById(manager, idStr)
	if err == sql.ErrNoRows {
		return fetchByName(manager, ownerProjId, idStr)
	} else {
		return obj, err
	}
}

func fetchItemById(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, idStr string, query jsonutils.JSONObject) (IModel, error) {
	q := manager.Query()
	var err error
	if query != nil && !query.IsZero() {
		q, err = listItemQueryFilters(manager, ctx, q, userCred, query)
		if err != nil {
			return nil, err
		}
	}
	q = manager.FilterById(q, idStr)
	count := q.Count()
	if count == 1 {
		item, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(item)
		if err != nil {
			return nil, err
		}
		return item, nil
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func fetchItemByName(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, idStr string, query jsonutils.JSONObject) (IModel, error) {
	q := manager.Query()
	var err error
	if query != nil && !query.IsZero() {
		q, err = listItemQueryFilters(manager, ctx, q, userCred, query)
		if err != nil {
			return nil, err
		}
	}
	q = manager.FilterByName(q, idStr)
	q = manager.FilterByOwner(q, userCred.GetProjectId())
	count := q.Count()
	if count == 1 {
		item, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(item)
		if err != nil {
			return nil, err
		}
		return item, nil
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func fetchItem(manager IModelManager, ctx context.Context, userCred mcclient.TokenCredential, idStr string, query jsonutils.JSONObject) (IModel, error) {
	item, err := fetchItemById(manager, ctx, userCred, idStr, query)
	if err != nil {
		item, err = fetchItemByName(manager, ctx, userCred, idStr, query)
	}
	return item, err
}
