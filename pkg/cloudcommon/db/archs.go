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

package db

import (
	"context"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:model-api-gen
type SMultiArchResourceBase struct {
	// 操作系统 CPU 架构
	// example: x86 arm
	OsArch string `width:"16" charset:"ascii" nullable:"true" list:"user" get:"user" create:"optional" update:"domain"`
}

type SMultiArchResourceBaseManager struct{}

func (manager *SMultiArchResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query apis.MultiArchResourceBaseListInput,
) (*sqlchemy.SQuery, error) {
	return ListQueryByArchitecture(q, "os_arch", query.OsArch), nil
}

func ListQueryByArchitecture(q *sqlchemy.SQuery, fieldKey string, archs []string) *sqlchemy.SQuery {
	if len(archs) == 0 {
		return q
	}
	conditions := []sqlchemy.ICondition{}
	for _, arch := range archs {
		if len(arch) == 0 {
			continue
		}
		if arch == apis.OS_ARCH_X86 {
			conditions = append(conditions, sqlchemy.OR(
				sqlchemy.Startswith(q.Field(fieldKey), arch),
				sqlchemy.Equals(q.Field(fieldKey), apis.OS_ARCH_I386),
				sqlchemy.IsNullOrEmpty(q.Field(fieldKey)),
			))
		} else if arch == apis.OS_ARCH_ARM {
			conditions = append(conditions, sqlchemy.OR(
				sqlchemy.Startswith(q.Field(fieldKey), arch),
				sqlchemy.Equals(q.Field(fieldKey), apis.OS_ARCH_AARCH32),
				sqlchemy.Equals(q.Field(fieldKey), apis.OS_ARCH_AARCH64),
			))
		} else {
			conditions = append(conditions, sqlchemy.Startswith(q.Field(fieldKey), arch))
		}
	}
	if len(conditions) > 0 {
		q = q.Filter(sqlchemy.OR(conditions...))
	}
	return q
}
