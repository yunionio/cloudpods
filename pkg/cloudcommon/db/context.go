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

	"yunion.io/x/pkg/appctx"
)

const (
	APP_CONTEXT_KEY_DB_METADATA_MANAGER     = appctx.AppContextKey("db_metadata_manager")
	APP_CONTEXT_KEY_DB_TENANT_CACHE_MANAGER = appctx.AppContextKey("db_tenant_cache_manager")
)

func GetMetadaManagerInContext(ctx context.Context) *SMetadataManager {
	val := ctx.Value(APP_CONTEXT_KEY_DB_METADATA_MANAGER)
	if val != nil {
		return val.(*SMetadataManager)
	} else {
		return Metadata
	}
}

func SaveMetadaManagerInContext(ctx context.Context, manager *SMetadataManager) context.Context {
	return context.WithValue(ctx, APP_CONTEXT_KEY_DB_METADATA_MANAGER, manager)
}

func GetTenantCacheManagerInContext(ctx context.Context) *STenantCacheManager {
	val := ctx.Value(APP_CONTEXT_KEY_DB_TENANT_CACHE_MANAGER)
	if val != nil {
		return val.(*STenantCacheManager)
	} else {
		return TenantCacheManager
	}
}

func SaveTenantCacheManagerInContext(ctx context.Context, manager *STenantCacheManager) context.Context {
	return context.WithValue(ctx, APP_CONTEXT_KEY_DB_TENANT_CACHE_MANAGER, manager)
}
