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

package quotas

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type IQuotaKeys interface {
	Fields() []string
	Values() []string
	Compare(IQuotaKeys) int

	OwnerId() mcclient.IIdentityProvider

	Scope() rbacutils.TRbacScope
}

type IQuota interface {
	db.IUsage

	GetKeys() IQuotaKeys
	SetKeys(IQuotaKeys)

	FetchSystemQuota()
	// FetchUsage(ctx context.Context) error
	Update(quota IQuota)
	Add(quota IQuota)
	Sub(quota IQuota)
	Allocable(quota IQuota) int
	ResetNegative()
	Exceed(request IQuota, quota IQuota) error
	// IsEmpty() bool
	ToJSON(prefix string) jsonutils.JSONObject
}

type IQuotaStore interface {
	object.IObject

	GetQuota(ctx context.Context, keys IQuotaKeys, quota IQuota) error
	GetChildrenQuotas(ctx context.Context, keys IQuotaKeys) ([]IQuota, error)
	GetParentQuotas(ctx context.Context, keys IQuotaKeys) ([]IQuota, error)

	SetQuota(ctx context.Context, userCred mcclient.TokenCredential, quota IQuota) error
	AddQuota(ctx context.Context, userCred mcclient.TokenCredential, diff IQuota) error
	SubQuota(ctx context.Context, userCred mcclient.TokenCredential, diff IQuota) error

	DeleteQuota(ctx context.Context, userCred mcclient.TokenCredential, keys IQuotaKeys) error
	DeleteAllQuotas(ctx context.Context, userCred mcclient.TokenCredential, keys IQuotaKeys) error
}

type IQuotaManager interface {
	db.IResourceModelManager

	checkSetPendingQuota(ctx context.Context, userCred mcclient.TokenCredential, quota IQuota) error
	cancelPendingUsage(ctx context.Context, userCred mcclient.TokenCredential, localUsage IQuota, cancelUsage IQuota, save bool) error
	cancelUsage(ctx context.Context, userCred mcclient.TokenCredential, usage IQuota) error
	addUsage(ctx context.Context, userCred mcclient.TokenCredential, usage IQuota) error
	getQuotaCount(ctx context.Context, request IQuota, pendingKey IQuotaKeys) (int, error)

	FetchIdNames(ctx context.Context, idMap map[string]map[string]string) (map[string]map[string]string, error)
}
