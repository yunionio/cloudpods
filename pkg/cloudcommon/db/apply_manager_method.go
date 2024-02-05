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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type ListItemExportKeysProvider interface {
	ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error)
}

func ApplyListItemExportKeys(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
	managers ...ListItemExportKeysProvider,
) (*sqlchemy.SQuery, error) {
	var err error
	for _, manager := range managers {
		q, err = manager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrapf(err, "%T.ListItemExportKeys", manager)
		}
	}
	return q, nil
}

type QueryDistinctExtraFieldProvider interface {
	QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error)
}

func ApplyQueryDistinctExtraField(
	q *sqlchemy.SQuery,
	field string,
	managers ...QueryDistinctExtraFieldProvider,
) (*sqlchemy.SQuery, error) {
	var err error
	for _, manager := range managers {
		q, err = manager.QueryDistinctExtraField(q, field)
		if err != nil {
			return nil, errors.Wrapf(err, "%T.QueryDistinctExtraField", manager)
		}
	}
	return q, nil
}

type FilterByOwnerProvider interface {
	Keyword() string
	KeywordPlural() string
	FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery
}

func ApplyFilterByOwner(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	owner mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	managers ...FilterByOwnerProvider,
) *sqlchemy.SQuery {
	for _, manager := range managers {
		q = manager.FilterByOwner(ctx, q, manager, userCred, owner, scope)
	}
	return q
}
