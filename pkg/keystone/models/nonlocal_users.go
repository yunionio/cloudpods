package models

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

type SNonlocalUserManager struct {
	db.SModelBaseManager
}

var NonlocalUserManager *SNonlocalUserManager

func init() {
	NonlocalUserManager = &SNonlocalUserManager{
		SModelBaseManager: db.NewModelBaseManager(
			SNonlocalUser{},
			"nonlocal_user",
			"nonlocal_user",
			"nonlocal_users",
		),
	}
}

/*
+-----------+--------------+------+-----+---------+-------+
| Field     | Type         | Null | Key | Default | Extra |
+-----------+--------------+------+-----+---------+-------+
| domain_id | varchar(64)  | NO   | PRI | NULL    |       |
| name      | varchar(255) | NO   | PRI | NULL    |       |
| user_id   | varchar(64)  | NO   | UNI | NULL    |       |
+-----------+--------------+------+-----+---------+-------+
*/

type SNonlocalUser struct {
	db.SModelBase

	DomainId string `width:"64" charset:"ascii" primary:"true"`
	Name     string `width:"255" charset:"utf8" primary:"true"`
	UserId   string `width:"64" charset:"ascii" nullable:"false" index:"true"`
}

func (manager *SNonlocalUserManager) Register(ctx context.Context, domainId string, name string) (*SNonlocalUser, error) {
	key := fmt.Sprintf("%s-%s", domainId, name)
	lockman.LockRawObject(ctx, manager.Keyword(), key)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), key)

	obj, err := db.NewModelObject(manager)
	if err != nil {
		return nil, errors.WithMessage(err, "NewModelObject")
	}
	nonlocalUser := obj.(*SNonlocalUser)
	q := manager.Query().Equals("domain_id", domainId).Equals("name", name)
	err = q.First(nonlocalUser)
	if err == nil {
		return nonlocalUser, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.WithMessage(err, "Query")
	}

	pubId, err := IdmappingManager.registerIdMap(ctx, domainId, name, api.IdMappingEntityUser)
	if err != nil {
		return nil, errors.WithMessage(err, "IdmappingManager.registerIdMap")
	}

	nonlocalUser.UserId = pubId
	nonlocalUser.Name = name
	nonlocalUser.DomainId = domainId

	err = manager.TableSpec().Insert(nonlocalUser)
	if err != nil {
		return nil, errors.WithMessage(err, "Insert")
	}

	return nonlocalUser, nil
}
