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
	Count     uint64    `nullable:"true"`
	StartDate time.Time `nullable:"true"`
	EndDate   time.Time `nullable:"true"`
	Deleted   bool      `nullable:"false"`
	DeleteAt  time.Time `nullable:"true"`
	CreatedAt time.Time `nullable:"false" created_at:"true"`
}

func (spec *SSplitTableSpec) getTableLastMeta() (*STableMetadata, error) {
	return spec.GetTableMetaByTime(time.Time{})
}

func (spec *SSplitTableSpec) GetTableMetaByTime(recordTime time.Time) (*STableMetadata, error) {
	q := spec.metaSpec.Query().Desc("id").IsFalse("deleted")
	if !recordTime.IsZero() {
		q = q.LE("start_date", recordTime)
	}
	meta := new(STableMetadata)
	err := q.First(meta)
	return meta, err
}

type ISplitTableObject interface {
	GetRecordTime() time.Time
}

func (spec *SSplitTableSpec) GetTableMetaByObject(obj ISplitTableObject) (*STableMetadata, error) {
	return spec.GetTableMetaByTime(obj.GetRecordTime())
}

func (spec *SSplitTableSpec) getTableMetasForInit() ([]STableMetadata, error) {
	q := spec.metaSpec.Query().Asc("id").IsFalse("deleted")
	q = q.AppendField(q.Field("table"))
	q = q.AppendField(q.Field("start"))
	metas := make([]STableMetadata, 0)
	err := q.All(&metas)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query metadata")
	}
	return metas, nil
}

func (spec *SSplitTableSpec) GetTableMetas() ([]STableMetadata, error) {
	q := spec.metaSpec.Query().Asc("id").IsFalse("deleted")
	metas := make([]STableMetadata, 0)
	err := q.All(&metas)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "query metadata")
	}
	for i := 0; i < len(metas); i++ {
		if metas[i].Count <= 0 {
			// fix count
			tbl := spec.GetTableSpec(metas[i]).Instance()
			cnt, err := tbl.Query().CountWithError()
			if err != nil {
				return nil, errors.Wrap(err, "CountWithError")
			}
			spec.metaSpec.Update(&metas[i], func() error {
				metas[i].Count = uint64(cnt)
				return nil
			})
		}
	}
	return metas, nil
}

func (spec *SSplitTableSpec) GetTableSpec(meta STableMetadata) *sqlchemy.STableSpec {
	tbSpec := *spec.tableSpec
	return tbSpec.Clone(meta.Table, meta.Start)
}
