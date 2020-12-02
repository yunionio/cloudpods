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

package splitable

import (
	"database/sql"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

type STableMetadata struct {
	Id        int64     `primary:"true" auto_increment:"true"`
	Table     string    `width:"64" charset:"ascii"`
	Start     int64     `nullable:"true"`
	End       int64     `nullable:"true"`
	StartDate time.Time `nullable:"true"`
	EndDate   time.Time `nullable:"true"`
	Deleted   bool      `nullable:"false"`
	DeleteAt  time.Time `nullable:"true"`
	CreatedAt time.Time `nullable:"false" created_at:"true"`
}

func (spec *SSplitTableSpec) GetTableMetas() ([]STableMetadata, error) {
	q := spec.metaSpec.Query().Asc("id").IsFalse("deleted")
	metas := make([]STableMetadata, 0)
	err := q.All(&metas)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query metadata")
	}
	return metas, nil
}

func (spec *SSplitTableSpec) GetTableSpec(meta STableMetadata) *sqlchemy.STableSpec {
	tbSpec := *spec.tableSpec
	return tbSpec.Clone(meta.Table, meta.Start)
}
