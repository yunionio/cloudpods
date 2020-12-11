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
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

func (t *SSplitTableSpec) Purge() error {
	if t.maxSegments <= 0 {
		return nil
	}
	metas, err := t.GetTableMetas()
	if err != nil {
		return errors.Wrap(err, "GetTableMetas")
	}
	if t.maxSegments >= len(metas) {
		return nil
	}
	for i := 0; i < len(metas)-t.maxSegments; i += 1 {
		dropSQL := fmt.Sprintf("DROP TABLE `%s`", metas[i].Table)
		log.Infof("Ready to drop table: %s", dropSQL)
		_, err := sqlchemy.Exec(dropSQL)
		if err != nil {
			return errors.Wrap(err, "sqlchemy.Exec")
		}
		_, err = t.metaSpec.Update(&metas[i], func() error {
			metas[i].DeleteAt = time.Now()
			metas[i].Deleted = true
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "metaSpec.Update")
		}
	}
	return nil
}
