package db

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func FetchJointByIds(manager IJointModelManager, masterId, slaveId string, query jsonutils.JSONObject) (IJointModel, error) {
	obj, err := NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	jointObj, ok := obj.(IJointModel)
	if !ok {
		return nil, fmt.Errorf("FetchByIds not a IJointModel")
	}
	q := manager.Query()
	masterField := queryField(q, manager.GetMasterManager())
	if masterField == nil {
		return nil, fmt.Errorf("cannot find master id")
	}
	slaveField := queryField(q, manager.GetSlaveManager())
	if slaveField == nil {
		return nil, fmt.Errorf("cannot find slave id")
	}
	cond := sqlchemy.AND(sqlchemy.Equals(masterField, masterId), sqlchemy.Equals(slaveField, slaveId))
	q = q.Filter(cond)
	q = manager.FilterByParams(q, query)
	count := q.Count()
	if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else if count == 0 {
		return nil, sql.ErrNoRows
	}
	err = q.First(jointObj)
	if err != nil {
		return nil, err
	}
	return jointObj, nil
}

func FetchById(manager IModelManager, idStr string) (IModel, error) {
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

func FetchByName(manager IModelManager, userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	var owner string
	if userCred != nil {
		owner = manager.GetOwnerId(userCred)
	}
	q := manager.Query()
	q = manager.FilterByName(q, idStr)
	count := q.Count()
	if count > 1 {
		q = manager.FilterByOwner(q, owner)
		count = q.Count()
	}
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

func FetchByIdOrName(manager IModelManager, userCred mcclient.IIdentityProvider, idStr string) (IModel, error) {
	obj, err := FetchById(manager, idStr)
	if err == sql.ErrNoRows {
		return FetchByName(manager, userCred, idStr)
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
	count := q.Count()
	if count > 1 {
		q = manager.FilterByOwner(q, manager.GetOwnerId(userCred))
		count = q.Count()
	}
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
