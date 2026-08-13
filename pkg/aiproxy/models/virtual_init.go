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
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func backfillEmptyTenantId(manager db.IModelManager) error {
	cred := auth.AdminCredential()
	if cred == nil {
		return nil
	}
	projectId := strings.TrimSpace(cred.GetProjectId())
	domainId := strings.TrimSpace(cred.GetProjectDomainId())
	if projectId == "" {
		return nil
	}
	cnt, err := manager.Query().IsNullOrEmpty("tenant_id").CountWithError()
	if err != nil {
		return errors.Wrapf(err, "count %s empty tenant_id", manager.KeywordPlural())
	}
	if cnt == 0 {
		return nil
	}
	_, err = sqlchemy.GetDB().Exec(
		fmt.Sprintf("update %s set tenant_id = ?, domain_id = ? where tenant_id is null or tenant_id = ''", manager.TableSpec().Name()),
		projectId,
		domainId,
	)
	if err != nil {
		return errors.Wrapf(err, "backfill %s tenant_id", manager.KeywordPlural())
	}
	return nil
}
