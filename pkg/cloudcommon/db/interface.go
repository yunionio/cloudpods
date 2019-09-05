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
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type IModelManager interface {
	lockman.ILockedClass
	object.IObject

	GetContextManagers() [][]IModelManager

	GetIModelManager() IModelManager

	// Table() *sqlchemy.STable
	TableSpec() *sqlchemy.STableSpec

	// Keyword() string
	KeywordPlural() string
	Alias() string
	AliasPlural() string
	SetAlias(alias string, aliasPlural string)

	ValidateName(name string) error

	// list hooks
	AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool
	ValidateListConditions(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error)
	CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*CustomizeListFilters, error)
	ExtraSearchConditions(ctx context.Context, q *sqlchemy.SQuery, like string) []sqlchemy.ICondition
	GetExportExtraKeys(ctx context.Context, query jsonutils.JSONObject, rowMap map[string]string) *jsonutils.JSONDict
	ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error)
	OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error)

	// fetch hook
	Query(val ...string) *sqlchemy.SQuery

	FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery
	FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery
	FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery
	FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery
	FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery
	FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery

	// GetOwnerId(userCred mcclient.IIdentityProvider) mcclient.IIdentityProvider

	// RawFetchById(idStr string) (IModel, error)
	FetchById(idStr string) (IModel, error)
	FetchByName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error)
	FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error)

	// create hooks
	AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	OnCreateComplete(ctx context.Context, items []IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)

	// allow perform action
	AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	AllowPerformCheckCreateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	// DoCreate(ctx context.Context, userCred mcclient.TokenCredential, kwargs jsonutils.JSONObject, data jsonutils.JSONObject, realManager IModelManager) (IModel, error)

	InitializeData() error

	CustomizeHandlerInfo(info *appsrv.SHandlerInfo)
	SetHandlerProcessTimeout(info *appsrv.SHandlerInfo, r *http.Request) time.Duration

	FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error)
	FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error)
	IsCustomizedGetDetailsBody() bool
	ListSkipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool
	GetSkipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool

	// list extend colums hook
	FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []IModel, fields stringutils2.SSortedStrings) []*jsonutils.JSONDict

	// fetch owner Id from query when create resource
	FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error)

	/* name uniqueness scope, system/domain/project, default is system */
	NamespaceScope() rbacutils.TRbacScope
	ResourceScope() rbacutils.TRbacScope

	QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error)

	GetPagingConfig() *SPagingConfig
}

type IModel interface {
	lockman.ILockedObject
	object.IObject

	GetName() string

	KeywordPlural() string

	GetModelManager() IModelManager
	SetModelManager(IModelManager, IModel)

	GetIModel() IModel

	GetShortDesc(ctx context.Context) *jsonutils.JSONDict

	// list hooks
	GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict

	// get hooks
	AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool
	GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error)
	GetExtraDetailsHeaders(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) map[string]string

	// create hooks
	CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error
	PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject)

	// allow perform action
	AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	// update hooks
	ValidateUpdateCondition(ctx context.Context) error

	AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool
	ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)
	PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)

	UpdateInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	// delete hooks
	AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	ValidateDeleteCondition(ctx context.Context) error
	CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error
	PreDelete(ctx context.Context, userCred mcclient.TokenCredential)
	MarkDelete() error
	Delete(ctx context.Context, userCred mcclient.TokenCredential) error
	PostDelete(ctx context.Context, userCred mcclient.TokenCredential)

	DeleteInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	GetOwnerId() mcclient.IIdentityProvider

	IsSharable(reqCred mcclient.IIdentityProvider) bool

	CustomizedGetDetailsBody(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error)
}

type IResourceModelManager interface {
	IModelManager

	GetIResourceModelManager() IResourceModelManager
}

type IResourceModel interface {
	IModel

	GetIResourceModel() IResourceModel
}

type IJointModelManager interface {
	IResourceModelManager

	GetIJointModelManager() IJointModelManager

	GetMasterManager() IStandaloneModelManager
	GetSlaveManager() IStandaloneModelManager

	GetMasterFieldName() string
	GetSlaveFieldName() string
	// FetchByIds(masterId string, slaveId string, query jsonutils.JSONObject) (IJointModel, error)
	FilterByParams(q *sqlchemy.SQuery, params jsonutils.JSONObject) *sqlchemy.SQuery

	AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model IStandaloneModel, query jsonutils.JSONObject) bool
	AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master IStandaloneModel, slave IStandaloneModel) bool
}

type IJointModel interface {
	IResourceModel

	GetJointModelManager() IJointModelManager

	GetIJointModel() IJointModel

	Master() IStandaloneModel
	Slave() IStandaloneModel

	AllowDetach(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool

	Detach(ctx context.Context, userCred mcclient.TokenCredential) error
	AllowGetJointDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, item IJointModel) bool
	AllowUpdateJointItem(ctx context.Context, userCred mcclient.TokenCredential, item IJointModel) bool
}

type IStandaloneModelManager interface {
	IResourceModelManager

	GetIStandaloneModelManager() IStandaloneModelManager

	// GenerateName(ownerProjId string, hint string) string
	// ValidateName(name string) error
	// IsNewNameUnique(name string, projectId string) bool

	// FetchByExternalId(idStr string) (IStandaloneModel, error)
}

type IStandaloneModel interface {
	IResourceModel
	// IsAlterNameUnique(name string, projectId string) bool
	// GetExternalId() string

	GetIStandaloneModel() IStandaloneModel
	ClearSchedDescCache() error
}

type IMetadataModel interface {
	IStandaloneModel

	GetAllMetadata(userCred mcclient.TokenCredential) (map[string]string, error)
	GetMetadataHideKeys() []string
}

type IVirtualModelManager interface {
	IStandaloneModelManager

	GetIVirtualModelManager() IVirtualModelManager
	GetResourceCount() ([]SProjectResourceCount, error)
}

type IVirtualModel interface {
	IStandaloneModel

	IsOwner(userCred mcclient.TokenCredential) bool
	// IsAdmin(userCred mcclient.TokenCredential) bool

	SyncCloudProjectId(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider)

	GetIVirtualModel() IVirtualModel
}

type ISharableVirtualModelManager interface {
	IVirtualModelManager

	GetISharableVirtualModelManager() ISharableVirtualModelManager
}

type ISharableVirtualModel interface {
	IVirtualModel

	GetISharableVirtualModel() ISharableVirtualModel
	GetSharedProjects() []string
}

type IAdminSharableVirtualModelManager interface {
	ISharableVirtualModelManager

	GetIAdminSharableVirtualModelManager() IAdminSharableVirtualModelManager

	GetRecordsSeparator() string
	GetRecordsLimit() int
	ParseInputInfo(data *jsonutils.JSONDict) ([]string, error)
}

type IAdminSharableVirtualModel interface {
	ISharableVirtualModel
	GetInfo() []string

	GetIAdminSharableVirtualModel() IAdminSharableVirtualModel
}
