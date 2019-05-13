package models

import (
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SPasswordManager struct {
	db.SResourceBaseManager
}

var PasswordManager *SPasswordManager

func init() {
	PasswordManager = &SPasswordManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SPassword{},
			"password",
			"password",
			"passwords",
		),
	}
}

/*
+----------------+--------------+------+-----+---------+----------------+
| Field          | Type         | Null | Key | Default | Extra          |
+----------------+--------------+------+-----+---------+----------------+
| id             | int(11)      | NO   | PRI | NULL    | auto_increment |
| local_user_id  | int(11)      | NO   | MUL | NULL    |                |
| password       | varchar(128) | YES  |     | NULL    |                |
| expires_at     | datetime     | YES  |     | NULL    |                |
| self_service   | tinyint(1)   | NO   |     | 0       |                |
| password_hash  | varchar(255) | YES  |     | NULL    |                |
| created_at_int | bigint(20)   | NO   |     | 0       |                |
| expires_at_int | bigint(20)   | YES  |     | NULL    |                |
| created_at     | datetime     | NO   |     | NULL    |                |
+----------------+--------------+------+-----+---------+----------------+
*/

type SPassword struct {
	db.SResourceBase

	Id           int       `primary:"true" auto_increment:"true"`
	LocalUserId  int       `nullable:"false" index:"true"`
	Password     string    `width:"128" charset:"ascii" nullable:"true"`
	ExpiresAt    time.Time `nullable:"true"`
	SelfService  bool      `nullable:"false" default:"false"`
	PasswordHash string    `width:"255" charset:"ascii" nullable:"true"`
	CreatedAtInt int64     `nullable:"false" default:"0"`
	ExpiresAtInt int64     `nullable:"true"`
}

func (manager *SPasswordManager) fetchByLocaluserId(localUserId int) ([]SPassword, error) {
	passes := make([]SPassword, 0)
	passwords := manager.Query().SubQuery()

	q := passwords.Query().Equals("local_user_id", localUserId).Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(passwords.Field("expires_at_int")),
		sqlchemy.Equals(passwords.Field("expires_at_int"), 0),
		sqlchemy.GE(passwords.Field("expires_at_int"), time.Now().UnixNano()/1000),
	))
	err := db.FetchModelObjects(manager, q, &passes)
	if err != nil {
		return nil, err
	}

	return passes, nil
}

func (manager *SPasswordManager) savePassword(localUserId int, password string) error {
	hash, err := seclib2.BcryptPassword(password)
	if err != nil {
		return errors.WithMessage(err, "seclib2.BcryptPassword")
	}
	rec := SPassword{}
	rec.LocalUserId = localUserId
	rec.PasswordHash = hash
	rec.CreatedAtInt = time.Now().UnixNano() / 1000
	err = manager.TableSpec().Insert(&rec)
	if err != nil {
		return errors.WithMessage(err, "Insert")
	}
	return nil
}

func (manager *SPasswordManager) delete(localUserId int) error {
	recs, err := manager.fetchByLocaluserId(localUserId)
	if err != nil {
		return errors.WithMessage(err, "manager.fetchByLocaluserId")
	}
	for i := range recs {
		_, err = db.Update(&recs[i], func() error {
			return recs[i].MarkDelete()
		})
		if err != nil {
			return errors.WithMessage(err, "recs[i].MarkDelete")
		}
	}
	return nil
}
