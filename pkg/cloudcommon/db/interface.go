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
	"yunion.io/x/pkg/object"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/splitable"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type IModelManager interface {
	lockman.ILockedClass
	object.IObject

	IsStandaloneManager() bool
	GetContextManagers() [][]IModelManager

	GetIModelManager() IModelManager

	GetMutableInstance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) IModelManager
	GetImmutableInstance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) IModelManager

	// Table() *sqlchemy.STable
	TableSpec() ITableSpec

	// Keyword() string
	// KeywordPlural() string
	Alias() string
	AliasPlural() string
	SetAlias(alias string, aliasPlural string)

	HasName() bool
	ValidateName(name string) error
	EnableGenerateName() bool

	// list hooks
	// AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool
	ValidateListConditions(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	// ListItemFilter dynamic called by dispatcher
	// ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error)
	CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*CustomizeListFilters, error)
	ExtraSearchConditions(ctx context.Context, q *sqlchemy.SQuery, like string) []sqlchemy.ICondition
	GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict
	ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error)
	// OrderByExtraFields dynmically called by dispatcher
	// OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error)

	// fetch hook
	Query(val ...string) *sqlchemy.SQuery
	RawQuery(val ...string) *sqlchemy.SQuery

	FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery
	FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery
	FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery

	FilterByOwnerProvider
	//FilterByOwner(q *sqlchemy.SQuery, man FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery

	FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery
	FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery
	FilterByUniqValues(q *sqlchemy.SQuery, uniqValues jsonutils.JSONObject) *sqlchemy.SQuery

	// GetOwnerId(userCred mcclient.IIdentityProvider) mcclient.IIdentityProvider

	// RawFetchById(idStr string) (IModel, error)
	FetchById(idStr string) (IModel, error)
	FetchByName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error)
	FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (IModel, error)

	// create hooks
	// AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	// BatchCreateValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	// ValidateCreateData dynamic called by dispatcher
	// ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	OnCreateComplete(ctx context.Context, items []IModel, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject)
	BatchPreValidate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider,
		query jsonutils.JSONObject, data *jsonutils.JSONDict, count int) error

	OnCreateFailed(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error

	// allow perform action
	// AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	// AllowPerformCheckCreateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
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
	// FetchCustomizeColumns dynamically called by dispatcher
	// FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []*jsonutils.JSONDict

	// fetch owner Id from query when create resource
	FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error)
	FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject

	/* name uniqueness scope, system/domain/project, default is system */
	NamespaceScope() rbacscope.TRbacScope
	ResourceScope() rbacscope.TRbacScope

	// 如果error为非空，说明没有匹配的field，如果为空，说明匹配上了
	QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error)

	GetPagingConfig() *SPagingConfig

	GetI18N(ctx context.Context, idstr string, resObj jsonutils.JSONObject) *jsonutils.JSONDict

	GetSplitTable() *splitable.SSplitTableSpec

	CreateByInsertOrUpdate() bool

	CustomizedTotalCount(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, totalQ *sqlchemy.SQuery) (int, jsonutils.JSONObject, error)
}

type IModel interface {
	lockman.ILockedObject
	object.IObject

	GetName() string
	GetUpdateVersion() int
	GetUpdatedAt() time.Time
	GetDeleted() bool

	KeywordPlural() string

	GetModelManager() IModelManager
	SetModelManager(IModelManager, IModel)

	GetIModel() IModel

	GetShortDesc(ctx context.Context) *jsonutils.JSONDict

	// list hooks
	//GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict

	// get hooks
	// AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool
	GetExtraDetailsHeaders(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) map[string]string

	// before create hooks
	CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error
	// after create hooks
	PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject)

	// allow perform action
	// AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PreCheckPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) error

	// update hooks
	ValidateUpdateCondition(ctx context.Context) error

	// AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool
	// ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)
	PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)

	UpdateInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	// delete hooks
	// AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	// ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error
	// CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error
	PreDelete(ctx context.Context, userCred mcclient.TokenCredential)
	MarkDelete() error
	Delete(ctx context.Context, userCred mcclient.TokenCredential) error
	PostDelete(ctx context.Context, userCred mcclient.TokenCredential)

	DeleteInContext(ctx context.Context, userCred mcclient.TokenCredential, ctxObjs []IModel, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	GetOwnerId() mcclient.IIdentityProvider
	GetUniqValues() jsonutils.JSONObject

	IsSharable(reqCred mcclient.IIdentityProvider) bool

	CustomizedGetDetailsBody(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error)
	MarkDeletePreventionOn()
	MarkDeletePreventionOff()

	GetUsages() []IUsage
	GetI18N(ctx context.Context) *jsonutils.JSONDict
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

	// AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model IStandaloneModel, query jsonutils.JSONObject) bool
	// AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master IStandaloneModel, slave IStandaloneModel) bool
}

type IJointModel interface {
	IResourceModel

	GetJointModelManager() IJointModelManager

	GetIJointModel() IJointModel

	// AllowDetach(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool

	Detach(ctx context.Context, userCred mcclient.TokenCredential) error
	// AllowGetJointDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, item IJointModel) bool
	// AllowUpdateJointItem(ctx context.Context, userCred mcclient.TokenCredential, item IJointModel) bool
}

type IMetadataBaseModelManager interface {
	GetMetadataHiddenKeys() []string
}

type IMetadataBaseModel interface {
	OnMetadataUpdated(ctx context.Context, userCred mcclient.TokenCredential)
}

type IStandaloneModelManager interface {
	IResourceModelManager

	GetIStandaloneModelManager() IStandaloneModelManager

	// GenerateName(ownerProjId string, hint string) string
	// ValidateName(name string) error
	// IsNewNameUnique(name string, projectId string) bool

	// FetchByExternalId(idStr string) (IStandaloneModel, error)

	IMetadataBaseModelManager
}

type IStandaloneModel interface {
	IResourceModel
	// IsAlterNameUnique(name string, projectId string) bool
	// GetExternalId() string

	StandaloneModelManager() IStandaloneModelManager

	GetIStandaloneModel() IStandaloneModel
	ClearSchedDescCache() error

	GetMetadata(ctx context.Context, key string, userCred mcclient.TokenCredential) string
	GetMetadataJson(ctx context.Context, key string, userCred mcclient.TokenCredential) jsonutils.JSONObject
	SetMetadata(ctx context.Context, key string, value interface{}, userCred mcclient.TokenCredential) error
	SetAllMetadata(ctx context.Context, dictstore map[string]interface{}, userCred mcclient.TokenCredential) error

	SetUserMetadataValues(ctx context.Context, dictstore map[string]string, userCred mcclient.TokenCredential) error
	SetUserMetadataAll(ctx context.Context, dictstore map[string]string, userCred mcclient.TokenCredential) error
	SetCloudMetadataAll(ctx context.Context, dictstore map[string]string, userCred mcclient.TokenCredential) error
	SetOrganizationMetadataAll(ctx context.Context, dictstore map[string]string, userCred mcclient.TokenCredential) error
	SetSysCloudMetadataAll(ctx context.Context, dictstore map[string]string, userCred mcclient.TokenCredential) error

	RemoveMetadata(ctx context.Context, key string, userCred mcclient.TokenCredential) error
	RemoveAllMetadata(ctx context.Context, userCred mcclient.TokenCredential) error
	GetAllMetadata(ctx context.Context, userCred mcclient.TokenCredential) (map[string]string, error)
	GetAllClassMetadata() (map[string]string, error)

	IsShared() bool

	IMetadataBaseModel
}

type IDomainLevelModelManager interface {
	IStandaloneModelManager

	GetIDomainLevelModelManager() IDomainLevelModelManager
	GetResourceCount() ([]SScopeResourceCount, error)
}

type IDomainLevelModel interface {
	IStandaloneModel

	IsOwner(userCred mcclient.TokenCredential) bool

	SyncCloudDomainId(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider)

	GetIDomainLevelModel() IDomainLevelModel

	IOwnerResourceBaseModel
}

type IInfrasModelManager interface {
	IDomainLevelModelManager

	GetIInfrasModelManager() IInfrasModelManager
}

type IInfrasModel interface {
	IDomainLevelModel
	ISharableBase

	GetIInfrasModel() IInfrasModel
}

type IVirtualModelManager interface {
	IStandaloneModelManager

	GetIVirtualModelManager() IVirtualModelManager
	GetResourceCount() ([]SScopeResourceCount, error)
}

type IUserModelManager interface {
	IStandaloneModelManager

	GetIUserModelManager() IUserModelManager
	GetResourceCount() ([]SScopeResourceCount, error)
}

type IVirtualModel interface {
	IStandaloneModel
	IPendingDeletable

	IsOwner(userCred mcclient.TokenCredential) bool
	// IsAdmin(userCred mcclient.TokenCredential) bool

	SyncCloudProjectId(userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider)

	GetIVirtualModel() IVirtualModel

	IOwnerResourceBaseModel
}

type ISharableVirtualModelManager interface {
	IVirtualModelManager

	GetISharableVirtualModelManager() ISharableVirtualModelManager
}

type ISharableVirtualModel interface {
	IVirtualModel
	ISharableBase

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

type IStatusStandaloneModel interface {
	IStandaloneModel
	IStatusBase
}

type IStatusDomainLevelModel interface {
	IDomainLevelModel
	IStatusBase
}

type IStatusInfrasModel interface {
	IInfrasModel
	IStatusBase
}
