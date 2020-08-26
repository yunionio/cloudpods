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
	"time"

	"github.com/golang-plus/uuid"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

// +onecloud:swagger-gen-ignore
type SIdmappingManager struct {
	db.SResourceBaseManager
}

var IdmappingManager *SIdmappingManager

func init() {
	IdmappingManager = &SIdmappingManager{
		SResourceBaseManager: db.NewResourceBaseManager(
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
	db.SResourceBase

	PublicId    string `width:"64" charset:"ascii" nullable:"false" primary:"false"`
	IdpId       string `name:"domain_id" width:"64" charset:"ascii" nullable:"false" primary:"true"`
	IdpEntityId string `name:"local_id" width:"128" charset:"utf8" nullable:"false" primary:"true"`
	EntityType  string `width:"10" charset:"ascii" nullable:"false"`
}

func getIdmapKey(idpId string, entityId string, entityType string) string {
	return fmt.Sprintf("%s-%s-%s", entityType, idpId, entityId)
}

func filterByIdpAndEntityId(q *sqlchemy.SQuery, idpId string, entityId string, entityType string) *sqlchemy.SQuery {
	return q.Equals("domain_id", idpId).Equals("local_id", entityId).Equals("entity_type", entityType)
}

func (manager *SIdmappingManager) RegisterIdMap(ctx context.Context, idpId string, entityId string, entityType string) (string, error) {
	return manager.RegisterIdMapWithId(ctx, idpId, entityId, entityType, "")
}

func (manager *SIdmappingManager) RegisterIdMapWithId(ctx context.Context, idpId string, entityId string, entityType string, publicId string) (string, error) {
	key := getIdmapKey(idpId, entityId, entityType)
	lockman.LockRawObject(ctx, manager.Keyword(), key)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), key)

	q := filterByIdpAndEntityId(manager.RawQuery(), idpId, entityId, entityType)

	mapping := SIdmapping{}
	mapping.SetModelManager(manager, &mapping)
	err := q.First(&mapping)
	if err != nil && err != sql.ErrNoRows {
		return "", errors.Wrap(err, "Query")
	}
	if err == sql.ErrNoRows {
		if len(publicId) == 0 {
			u1, _ := uuid.NewV4()
			u2, _ := uuid.NewV4()
			publicId = u1.Format(uuid.StyleWithoutDash) + u2.Format(uuid.StyleWithoutDash)
		}
		mapping.PublicId = publicId
		mapping.IdpId = idpId
		mapping.IdpEntityId = entityId
		mapping.EntityType = entityType

		err = manager.TableSpec().InsertOrUpdate(ctx, &mapping)
		if err != nil {
			return "", errors.Wrap(err, "Insert")
		}
	} else {
		if mapping.Deleted {
			if len(publicId) == 0 {
				u1, _ := uuid.NewV4()
				u2, _ := uuid.NewV4()
				publicId = u1.Format(uuid.StyleWithoutDash) + u2.Format(uuid.StyleWithoutDash)
			}
			_, err = db.Update(&mapping, func() error {
				mapping.PublicId = publicId
				mapping.Deleted = false
				mapping.DeletedAt = time.Time{}
				return nil
			})
			if err != nil {
				return "", errors.Wrap(err, "undelete")
			}
		}
	}

	return mapping.PublicId, nil
}

func (manager *SIdmappingManager) FetchByIdpAndEntityId(ctx context.Context, idpId string, entityId string, entityType string) (string, error) {
	key := getIdmapKey(idpId, entityId, entityType)
	lockman.LockRawObject(ctx, manager.Keyword(), key)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), key)

	q := filterByIdpAndEntityId(manager.Query(), idpId, entityId, entityType)

	mapping := SIdmapping{}
	mapping.SetModelManager(manager, &mapping)
	err := q.First(&mapping)
	if err != nil {
		return "", err
	} else {
		return mapping.PublicId, nil
	}
}

func (manager *SIdmappingManager) FetchEntities(idStr string, entType string) ([]SIdmapping, error) {
	q := manager.Query().Equals("public_id", idStr).Equals("entity_type", entType)
	idMaps := make([]SIdmapping, 0)
	err := db.FetchModelObjects(manager, q, &idMaps)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	} else {
		return idMaps, nil
	}
}

func (manager *SIdmappingManager) FetchFirstEntity(idStr string, entType string) (*SIdmapping, error) {
	q := manager.Query().Equals("public_id", idStr).Equals("entity_type", entType)
	idMap := SIdmapping{}
	idMap.SetModelManager(manager, &idMap)
	err := q.First(&idMap)
	if err != nil {
		return nil, err
	}
	return &idMap, nil
}

func (manager *SIdmappingManager) deleteByIdpId(idpId string) error {
	return manager.deleteAny(idpId, "", "")
}

func (manager *SIdmappingManager) deleteAny(idpId string, entityType string, publicId string) error {
	q := manager.Query().Equals("domain_id", idpId)
	if len(entityType) > 0 {
		q = q.Equals("entity_type", entityType)
	}
	if len(publicId) > 0 {
		q = q.Equals("public_id", publicId)
	}
	idmappings := make([]SIdmapping, 0)
	err := db.FetchModelObjects(manager, q, &idmappings)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "FetchModelObjects")
	}
	for i := range idmappings {
		_, err = db.Update(&idmappings[i], func() error {
			return idmappings[i].MarkDelete()
		})
		if err != nil {
			return errors.Wrap(err, "markdelete")
		}
	}
	return nil
}

func (manager *SIdmappingManager) FetchPublicIdsExcludesQuery(idpId string, entityType string, excludes []string) *sqlchemy.SQuery {
	q := manager.Query("public_id").Equals("domain_id", idpId)
	q = q.Equals("entity_type", entityType)
	if len(excludes) > 0 {
		q = q.NotIn("public_id", excludes)
	}
	return q
}

func (manager *SIdmappingManager) FetchPublicIdsExcludes(idpId string, entityType string, excludes []string) ([]string, error) {
	q := manager.FetchPublicIdsExcludesQuery(idpId, entityType, excludes)
	rows, err := q.Rows()
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "q.Rows")
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	ret := make([]string, 0)
	for rows.Next() {
		var idStr string
		err = rows.Scan(&idStr)
		if err != nil {
			return nil, errors.Wrap(err, "rows.Scan")
		}
		ret = append(ret, idStr)
	}
	return ret, nil
}

func (manager *SIdmappingManager) deleteByPublicId(publicId string, entityType string) error {
	idmappings, err := manager.FetchEntities(publicId, entityType)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil
		} else {
			return errors.Wrap(err, "manager.FetchEntities")
		}
	}
	for i := range idmappings {
		_, err = db.Update(&idmappings[i], func() error {
			return idmappings[i].MarkDelete()
		})
		if err != nil {
			return errors.Wrap(err, "markdelete")
		}
	}
	return nil
}
