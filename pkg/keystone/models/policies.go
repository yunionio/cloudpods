package models

import (
	"database/sql"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/pkg/tristate"
)

type SPolicyManager struct {
	db.SStandaloneResourceBaseManager
}

var PolicyManager *SPolicyManager

func init() {
	PolicyManager = &SPolicyManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SPolicy{},
			"policy",
			"policy",
			"policies",
		),
	}
}

/*
+-------+--------------+------+-----+---------+-------+
| Field | Type         | Null | Key | Default | Extra |
+-------+--------------+------+-----+---------+-------+
| id    | varchar(64)  | NO   | PRI | NULL    |       |
| type  | varchar(255) | NO   |     | NULL    |       |
| blob  | text         | NO   |     | NULL    |       |
| extra | text         | YES  |     | NULL    |       |
+-------+--------------+------+-----+---------+-------+
*/

type SPolicy struct {
	db.SStandaloneResourceBase

	Type string `width:"255" charset:"utf8" nullable:"false" list:"user" update:"admin"`
	Blob string `nullable:"false" list:"user" update:"admin"`

	Extra *jsonutils.JSONDict `nullable:"true" list:"user"`

	Enabled tristate.TriState `nullable:"false" default:"false" list:"user" update:"admin" create:"admin_optional"`
}

func (manager *SPolicyManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNullOrEmpty("name")
	policies := make([]SPolicy, 0)
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil {
		return err
	}
	for i := range policies {
		db.Update(&policies[i], func() error {
			policies[i].Name = policies[i].Type
			policies[i].Description, _ = policies[i].Extra.GetString("description")
			return nil
		})
	}
	return nil
}

func (manager *SPolicyManager) FetchEnabledPolicies() ([]SPolicy, error) {
	q := manager.Query().IsTrue("enabled")

	policies := make([]SPolicy, 0)
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return policies, nil
}
