// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang-plus/uuid"
	"github.com/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

type SIdmappingManager struct {
	db.SModelBaseManager
}

var IdmappingManager *SIdmappingManager

func init() {
	IdmappingManager = &SIdmappingManager{
		SModelBaseManager: db.NewModelBaseManager(
			SIdmapping{},
			"id_mapping",
			"id_mapping",
			"id_mappings",
		),
	}
	IdmappingManager.SetVirtualObject(IdmappingManager)
}

/*
+-------------+----------------------+------+-----+---------+-------+
| Field       | Type                 | Null | Key | Default | Extra |
+-------------+----------------------+------+-----+---------+-------+
| public_id   | varchar(64)          | NO   | PRI | NULL    |       |
| domain_id   | varchar(64)          | NO   | MUL | NULL    |       |
| local_id    | varchar(64)          | NO   |     | NULL    |       |
| entity_type | enum('user','group') | NO   |     | NULL    |       |
+-------------+----------------------+------+-----+---------+-------+
*/

type SIdmapping struct {
	db.SModelBase

	PublicId   string `width:"64" charset:"ascii" nullable:"false" primary:"true"`
	DomainId   string `width:"64" charset:"ascii" nullable:"false" index:"true"`
	LocalId    string `width:"64" charset:"ascii" nullable:"false"`
	EntityType string `width:"10" charset:"ascii" nullable:"false"`
}

func (manager *SIdmappingManager) registerIdMap(ctx context.Context, domainId string, localId string, entityType string) (string, error) {
	key := fmt.Sprintf("%s-%s-%s", entityType, domainId, localId)
	lockman.LockRawObject(ctx, manager.Keyword(), key)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), key)

	q := manager.Query().Equals("domain_id", domainId).Equals("local_id", localId).Equals("entity_type", entityType)

	mapping := SIdmapping{}
	err := q.First(&mapping)
	if err != nil && err != sql.ErrNoRows {
		return "", errors.Wrap(err, "Query")
	}
	if err == sql.ErrNoRows {
		u1, _ := uuid.NewV4()
		u2, _ := uuid.NewV4()
		mapping.PublicId = u1.Format(uuid.StyleWithoutDash) + u2.Format(uuid.StyleWithoutDash)
		mapping.DomainId = domainId
		mapping.LocalId = localId
		mapping.EntityType = entityType

		err = manager.TableSpec().Insert(&mapping)
		if err != nil {
			return "", errors.Wrap(err, "Insert")
		}
	}

	return mapping.PublicId, nil
}
