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
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudaccountSyncTaskManager struct {
	db.SDomainLevelResourceBaseManager
	db.SStatusResourceBaseManager
}

var CloudaccountSyncTaskManager *SCloudaccountSyncTaskManager

func init() {
	CloudaccountSyncTaskManager = &SCloudaccountSyncTaskManager{
		SDomainLevelResourceBaseManager: db.NewDomainLevelResourceBaseManager(
			SCloudaccountSyncTask{},
			"cloudaccount_sync_tasks_tbl",
			api.CloudaccountSyncTaskKeyword,
			api.CloudaccountSyncTaskKeywordPlural,
		),
	}
	CloudaccountSyncTaskManager.SetVirtualObject(CloudaccountSyncTaskManager)
}

type SCloudaccountSyncTask struct {
	db.SDomainLevelResourceBase
	db.SStatusResourceBase

	// 父任务 ID（账号级为空）
	ParentId string `width:"36" charset:"ascii" nullable:"true" list:"domain" create:"optional" index:"true"`

	// 任务层级: account / provider / region
	Level string `width:"16" charset:"ascii" nullable:"false" list:"domain" create:"required" index:"true"`

	CloudaccountId  string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"required" index:"true"`
	CloudproviderId string `width:"36" charset:"ascii" nullable:"true" list:"domain" create:"optional" index:"true"`
	CloudregionId   string `width:"36" charset:"ascii" nullable:"true" list:"domain" create:"optional" index:"true"`

	Mode string `width:"16" charset:"ascii" nullable:"false" default:"manual" list:"domain"`

	QueuedAt  time.Time `nullable:"true" list:"domain"`
	StartedAt time.Time `nullable:"true" list:"domain"`
	EndedAt   time.Time `nullable:"true" list:"domain"`

	SyncRange   jsonutils.JSONObject `nullable:"true" list:"domain"`
	SyncResults jsonutils.JSONObject `nullable:"true" list:"domain"`
}

func (manager *SCloudaccountSyncTaskManager) InitializeData() error {
	ctx := context.Background()
	tasks, err := manager.fetchTasksByFilter("", "", api.CloudaccountSyncTaskLevelAccount, api.CloudaccountSyncTaskStatusActive)
	if err != nil {
		return errors.Wrap(err, "fetchTasksByFilter")
	}
	userCred := auth.AdminCredential()
	for i := range tasks {
		err := tasks[i].cancelCascade(ctx, userCred, "cancel tasks because of service restart")
		if err != nil {
			log.Errorf("cancel sync task %s fail: %v", tasks[i].Id, err)
		}
	}
	return nil
}

// HistoryDataClean 实现 db.IHistoryDataManager：清理 ended_at 早于 timeBefore 的终态任务。
// 写法对齐 SStandaloneAnonResourceBaseManager.HistoryDataClean，供 history-data-clean API 与定时任务复用。
func (manager *SCloudaccountSyncTaskManager) HistoryDataClean(ctx context.Context, timeBefore time.Time) (int, error) {
	q := manager.Query("id").
		In("status", api.CloudaccountSyncTaskStatusTerminal).
		LE("ended_at", timeBefore)
	rows, err := q.Rows()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return 0, nil
		}
		return 0, errors.Wrap(err, "Query")
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, errors.Wrap(err, "rows.Scan")
		}
		ids = append(ids, id)
	}

	purge := func(ids []string) error {
		vars := make([]interface{}, 0, len(ids))
		placeholders := make([]string, len(ids))
		for i := range placeholders {
			placeholders[i] = "?"
			vars = append(vars, ids[i])
		}
		sqlStr := fmt.Sprintf(
			"delete from %s where id in (%s)",
			manager.TableSpec().Name(), strings.Join(placeholders, ","),
		)
		lockman.LockRawObject(ctx, manager.Keyword(), "purge")
		defer lockman.ReleaseRawObject(ctx, manager.Keyword(), "purge")
		_, err := sqlchemy.GetDB().Exec(sqlStr, vars...)
		if err != nil {
			return errors.Wrapf(err, strings.ReplaceAll(sqlStr, "?", "%s"), vars...)
		}
		return nil
	}

	splitByLen := func(data []string, splitLen int) [][]string {
		var result [][]string
		for i := 0; i < len(data); i += splitLen {
			end := i + splitLen
			if end > len(data) {
				end = len(data)
			}
			result = append(result, data[i:end])
		}
		return result
	}
	for _, batch := range splitByLen(ids, 100) {
		if err := purge(batch); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

// CleanupExpiredTasks 定时入口：按 CloudaccountSyncTaskKeepDays 调用 HistoryDataClean。
func (manager *SCloudaccountSyncTaskManager) CleanupExpiredTasks(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	keepDays := options.Options.CloudaccountSyncTaskKeepDays
	if keepDays <= 0 {
		return
	}
	cnt, err := manager.HistoryDataClean(ctx, time.Now().UTC().AddDate(0, 0, -keepDays))
	if err != nil {
		log.Errorf("CleanupExpiredTasks HistoryDataClean: %v", err)
		return
	}
	if cnt > 0 {
		log.Infof("CleanupExpiredTasks deleted %d cloudaccount sync tasks older than %d days", cnt, keepDays)
	}
}

func (manager *SCloudaccountSyncTaskManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudaccountSyncTaskListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainLevelResourceBaseManager.ListItemFilter")
	}

	if len(query.Status) == 0 {
		if !query.All {
			if query.Running {
				query.Status = []string{api.CloudaccountSyncTaskStatusRunning}
			} else if query.Active {
				query.Status = api.CloudaccountSyncTaskStatusActive
			}
		}
	}

	q, err = manager.SStatusResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusResourceBaseManager.ListItemFilter")
	}

	if query.AccountLevel || (len(query.Level) == 0 && len(query.ParentId) == 0 && len(query.CloudproviderId) == 0 && len(query.CloudregionId) == 0) {
		// default list account-level only when not drilling down
		if len(query.Level) == 0 && len(query.ParentId) == 0 {
			q = q.Equals("level", api.CloudaccountSyncTaskLevelAccount)
		}
	}
	if len(query.Level) > 0 {
		q = q.Equals("level", query.Level)
	}
	if len(query.ParentId) > 0 {
		q = q.Equals("parent_id", query.ParentId)
	}
	if len(query.CloudaccountId) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(ctx, userCred, query.CloudaccountId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudaccountManager.Keyword(), query.CloudaccountId)
			}
			return nil, errors.Wrap(err, "CloudaccountManager.FetchByIdOrName")
		}
		q = q.Equals("cloudaccount_id", account.GetId())
	}
	if len(query.CloudproviderId) > 0 {
		q = q.Equals("cloudprovider_id", query.CloudproviderId)
	}
	if len(query.CloudregionId) > 0 {
		q = q.Equals("cloudregion_id", query.CloudregionId)
	}
	return q, nil
}

func (manager *SCloudaccountSyncTaskManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudaccountSyncTaskListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SDomainLevelResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainLevelResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainLevelResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SCloudaccountSyncTaskManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SDomainLevelResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SCloudaccountSyncTaskManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudaccountSyncTaskDetails {
	rows := make([]api.CloudaccountSyncTaskDetails, len(objs))
	domainRows := manager.SDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	accountIds := stringutils2.SSortedStrings{}
	providerIds := stringutils2.SSortedStrings{}
	regionIds := stringutils2.SSortedStrings{}
	taskIds := make([]string, len(objs))
	for i := range objs {
		task := objs[i].(*SCloudaccountSyncTask)
		taskIds[i] = task.Id
		rows[i] = api.CloudaccountSyncTaskDetails{
			DomainLevelResourceDetails: domainRows[i],
			Status:                     task.Status,
			ParentId:                   task.ParentId,
			Level:                      task.Level,
			Mode:                       task.Mode,
			QueuedAt:                   task.QueuedAt,
			StartedAt:                  task.StartedAt,
			EndedAt:                    task.EndedAt,
			SyncResults:                task.SyncResults,
		}
		if len(task.CloudaccountId) > 0 {
			accountIds = append(accountIds, task.CloudaccountId)
		}
		if len(task.CloudproviderId) > 0 {
			providerIds = append(providerIds, task.CloudproviderId)
		}
		if len(task.CloudregionId) > 0 {
			regionIds = append(regionIds, task.CloudregionId)
		}
	}

	accounts := map[string]SCloudaccount{}
	if len(accountIds) > 0 {
		db.FetchStandaloneObjectsByIds(CloudaccountManager, accountIds, &accounts)
	}
	providers := map[string]SCloudprovider{}
	if len(providerIds) > 0 {
		db.FetchStandaloneObjectsByIds(CloudproviderManager, providerIds, &providers)
	}
	regions := map[string]SCloudregion{}
	if len(regionIds) > 0 {
		db.FetchStandaloneObjectsByIds(CloudregionManager, regionIds, &regions)
	}

	childStats := manager.fetchChildStats(taskIds)

	for i := range objs {
		task := objs[i].(*SCloudaccountSyncTask)
		if acc, ok := accounts[task.CloudaccountId]; ok {
			rows[i].Cloudaccount = acc.Name
		}
		if p, ok := providers[task.CloudproviderId]; ok {
			rows[i].Cloudprovider = p.Name
		}
		if r, ok := regions[task.CloudregionId]; ok {
			rows[i].Cloudregion = r.Name
		}
		if st, ok := childStats[task.Id]; ok {
			rows[i].ChildTotal = st.Total
			rows[i].ChildDone = st.Done
			rows[i].ChildFail = st.Fail
			rows[i].ChildCancel = st.Cancel
			rows[i].ChildRunning = st.Running
			rows[i].ChildQueued = st.Queued
		}
	}
	return rows
}

type sChildStat struct {
	Total   int
	Done    int
	Fail    int
	Cancel  int
	Running int
	Queued  int
}

func (manager *SCloudaccountSyncTaskManager) fetchChildStats(parentIds []string) map[string]sChildStat {
	ret := map[string]sChildStat{}
	if len(parentIds) == 0 {
		return ret
	}
	type row struct {
		ParentId string
		Status   string
		Cnt      int
	}
	q := manager.Query("parent_id", "status")
	q = q.In("parent_id", parentIds)
	q = q.AppendField(sqlchemy.COUNT("cnt"))
	q = q.GroupBy("parent_id", "status")
	rows := []row{}
	err := q.All(&rows)
	if err != nil {
		log.Errorf("fetchChildStats fail: %v", err)
		return ret
	}
	for _, r := range rows {
		st := ret[r.ParentId]
		st.Total += r.Cnt
		switch r.Status {
		case api.CloudaccountSyncTaskStatusDone:
			st.Done += r.Cnt
		case api.CloudaccountSyncTaskStatusFail:
			st.Fail += r.Cnt
		case api.CloudaccountSyncTaskStatusCancel:
			st.Cancel += r.Cnt
		case api.CloudaccountSyncTaskStatusRunning:
			st.Running += r.Cnt
		case api.CloudaccountSyncTaskStatusQueued:
			st.Queued += r.Cnt
		}
		ret[r.ParentId] = st
	}
	return ret
}

func (manager *SCloudaccountSyncTaskManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotSupportedError("creating cloudaccount sync task directly is not supported")
}

func (task *SCloudaccountSyncTask) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return httperrors.NewNotSupportedError("deleting cloudaccount sync task is not supported")
}

func (manager *SCloudaccountSyncTaskManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (manager *SCloudaccountSyncTaskManager) FetchChildren(parentId string) ([]SCloudaccountSyncTask, error) {
	return manager.fetchTasksByFilter("", parentId, "", nil)
}

func (manager *SCloudaccountSyncTaskManager) fetchTasksByFilter(accountId, parentId, level string, status []string) ([]SCloudaccountSyncTask, error) {
	q := manager.Query()
	if len(accountId) > 0 {
		q = q.Equals("cloudaccount_id", accountId)
	}
	if len(parentId) > 0 {
		q = q.Equals("parent_id", parentId)
	}
	if len(level) > 0 {
		q = q.Equals("level", level)
	}
	if len(status) > 0 {
		q = q.In("status", status)
	}
	ret := make([]SCloudaccountSyncTask, 0)
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return ret, nil
}

func (manager *SCloudaccountSyncTaskManager) cancelActiveAccountTasks(ctx context.Context, userCred mcclient.TokenCredential, accountId string, reason string) error {
	tasks, err := manager.fetchTasksByFilter(accountId, "", api.CloudaccountSyncTaskLevelAccount, api.CloudaccountSyncTaskStatusActive)
	if err != nil {
		return errors.Wrap(err, "fetchTasksByFilter")
	}
	for i := range tasks {
		err := tasks[i].cancelCascade(ctx, userCred, reason)
		if err != nil {
			log.Errorf("cancelActiveAccountTasks %s: %v", tasks[i].Id, err)
		}
	}
	return nil
}

// syncRangeTargetsSpecificRegions 表示本次同步限定了 region/zone/host，可与账号下其它 region 并行。
func syncRangeTargetsSpecificRegions(syncRange *SSyncRange) bool {
	if syncRange == nil {
		return false
	}
	return len(syncRange.Region) > 0 || len(syncRange.Zone) > 0 || len(syncRange.Host) > 0
}

func (manager *SCloudaccountSyncTaskManager) findActiveAccountTask(accountId string) (*SCloudaccountSyncTask, error) {
	tasks, err := manager.fetchTasksByFilter(accountId, "", api.CloudaccountSyncTaskLevelAccount, api.CloudaccountSyncTaskStatusActive)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	// 优先复用 running，其次任意 active（按创建时间取最后一个更接近当前同步）
	var picked *SCloudaccountSyncTask
	for i := range tasks {
		t := &tasks[i]
		if t.Status == api.CloudaccountSyncTaskStatusRunning {
			return t, nil
		}
		picked = t
	}
	return picked, nil
}

func (manager *SCloudaccountSyncTaskManager) findActiveProviderTask(parentId, providerId string) (*SCloudaccountSyncTask, error) {
	tasks, err := manager.fetchTasksByFilter("", parentId, api.CloudaccountSyncTaskLevelProvider, api.CloudaccountSyncTaskStatusActive)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		if tasks[i].CloudproviderId == providerId {
			return &tasks[i], nil
		}
	}
	return nil, nil
}

// StartAccountSyncTask 创建或复用 L1 账号同步任务。
// replaceActive=false（订阅/region 同步）：复用已有 active L1，不取消其它订阅或 region。
// replaceActive=true（账号全量同步）：取消旧 active L1 后再建新任务。
func (manager *SCloudaccountSyncTaskManager) StartAccountSyncTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	account *SCloudaccount,
	syncRange *SSyncRange,
	mode string,
	replaceActive bool,
) (*SCloudaccountSyncTask, error) {
	lockman.LockRawObject(ctx, manager.Keyword(), account.Id)
	defer lockman.ReleaseRawObject(ctx, manager.Keyword(), account.Id)

	if !replaceActive {
		existing, err := manager.findActiveAccountTask(account.Id)
		if err != nil {
			return nil, errors.Wrap(err, "findActiveAccountTask")
		}
		if existing != nil {
			return existing, nil
		}
	} else {
		err := manager.cancelActiveAccountTasks(ctx, userCred, account.Id, "cancel old tasks because of new sync")
		if err != nil {
			log.Errorf("cancelActiveAccountTasks: %v", err)
		}
	}

	now := time.Now().UTC()
	task := &SCloudaccountSyncTask{}
	task.SetModelManager(manager, task)
	task.Name = fmt.Sprintf("sync-%s-%s", account.Name, now.Format("20060102-150405"))
	task.DomainId = account.DomainId
	task.Level = api.CloudaccountSyncTaskLevelAccount
	task.CloudaccountId = account.Id
	task.Status = api.CloudaccountSyncTaskStatusQueued
	task.QueuedAt = now
	task.Mode = mode
	if syncRange != nil {
		task.SyncRange = jsonutils.Marshal(syncRange.SyncRangeInput)
	}
	err := manager.TableSpec().Insert(ctx, task)
	if err != nil {
		return nil, errors.Wrap(err, "Insert account sync task")
	}
	db.OpsLog.LogEvent(task, db.ACT_CREATE, nil, userCred)
	logclient.AddActionLogWithContext(ctx, task, logclient.ACT_CREATE, nil, userCred, true)
	return task, nil
}

func (manager *SCloudaccountSyncTaskManager) StartProviderSyncTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	parent *SCloudaccountSyncTask,
	provider *SCloudprovider,
	syncRange *SSyncRange,
) (*SCloudaccountSyncTask, error) {
	if parent == nil {
		return nil, nil
	}
	if parent.IsCanceled() {
		return nil, errors.Wrap(context.Canceled, "account sync task cancelled")
	}
	existing, err := manager.findActiveProviderTask(parent.Id, provider.Id)
	if err != nil {
		return nil, errors.Wrap(err, "findActiveProviderTask")
	}
	if existing != nil {
		return existing, nil
	}
	now := time.Now().UTC()
	task := &SCloudaccountSyncTask{}
	task.SetModelManager(manager, task)
	task.Name = fmt.Sprintf("sync-provider-%s", provider.Name)
	task.DomainId = parent.DomainId
	task.ParentId = parent.Id
	task.Level = api.CloudaccountSyncTaskLevelProvider
	task.CloudaccountId = parent.CloudaccountId
	task.CloudproviderId = provider.Id
	task.Status = api.CloudaccountSyncTaskStatusQueued
	task.QueuedAt = now
	task.Mode = parent.Mode
	if syncRange != nil {
		task.SyncRange = jsonutils.Marshal(syncRange.SyncRangeInput)
	}
	err = manager.TableSpec().Insert(ctx, task)
	if err != nil {
		return nil, errors.Wrap(err, "Insert provider sync task")
	}
	parent.markRunning(ctx, userCred)
	return task, nil
}

func (manager *SCloudaccountSyncTaskManager) StartRegionSyncTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	parent *SCloudaccountSyncTask,
	cpr *SCloudproviderregion,
) (*SCloudaccountSyncTask, error) {
	if parent == nil {
		return nil, nil
	}
	if parent.IsCanceled() {
		return nil, errors.Wrap(context.Canceled, "provider sync task cancelled")
	}
	// also refuse if account-level ancestor was cancelled
	if len(parent.ParentId) > 0 {
		if aObj, err := db.FetchById(manager, parent.ParentId); err == nil {
			if aObj.(*SCloudaccountSyncTask).IsCanceled() {
				return nil, errors.Wrap(context.Canceled, "account sync task cancelled")
			}
		}
	}
	// 同一 region 重复点击：只取消该 region 旧叶子，不影响其它 region
	siblings, err := manager.fetchTasksByFilter("", parent.Id, api.CloudaccountSyncTaskLevelRegion, api.CloudaccountSyncTaskStatusActive)
	if err != nil {
		return nil, errors.Wrap(err, "fetch active region siblings")
	}
	for i := range siblings {
		if siblings[i].CloudproviderId == cpr.CloudproviderId && siblings[i].CloudregionId == cpr.CloudregionId {
			if cerr := siblings[i].cancelCascade(ctx, userCred, "cancel old region task because of new region sync"); cerr != nil {
				log.Errorf("cancel old region sync task %s: %v", siblings[i].Id, cerr)
			}
		}
	}
	region, _ := cpr.GetRegion()
	regionName := cpr.CloudregionId
	if region != nil {
		regionName = region.Name
	}
	now := time.Now().UTC()
	task := &SCloudaccountSyncTask{}
	task.SetModelManager(manager, task)
	task.Name = fmt.Sprintf("sync-region-%s", regionName)
	task.DomainId = parent.DomainId
	task.ParentId = parent.Id
	task.Level = api.CloudaccountSyncTaskLevelRegion
	task.CloudaccountId = parent.CloudaccountId
	task.CloudproviderId = cpr.CloudproviderId
	task.CloudregionId = cpr.CloudregionId
	task.Status = api.CloudaccountSyncTaskStatusQueued
	task.QueuedAt = now
	task.Mode = parent.Mode
	task.SyncRange = parent.SyncRange
	err = manager.TableSpec().Insert(ctx, task)
	if err != nil {
		return nil, errors.Wrap(err, "Insert region sync task")
	}
	// Insert 与 IsCanceled 检查之间可能已被取消：立刻作废新叶子，避免残留 queued
	if parent.IsCanceled() {
		_, _ = db.Update(task, func() error {
			task.Status = api.CloudaccountSyncTaskStatusCancel
			task.EndedAt = time.Now().UTC()
			if task.StartedAt.IsZero() {
				task.StartedAt = task.EndedAt
			}
			return nil
		})
		return nil, errors.Wrap(context.Canceled, "provider sync task cancelled")
	}
	if len(parent.ParentId) > 0 {
		if aObj, err := db.FetchById(manager, parent.ParentId); err == nil {
			if aObj.(*SCloudaccountSyncTask).IsCanceled() {
				_, _ = db.Update(task, func() error {
					task.Status = api.CloudaccountSyncTaskStatusCancel
					task.EndedAt = time.Now().UTC()
					if task.StartedAt.IsZero() {
						task.StartedAt = task.EndedAt
					}
					return nil
				})
				return nil, errors.Wrap(context.Canceled, "account sync task cancelled")
			}
		}
	}
	parent.markRunning(ctx, userCred)
	if len(parent.ParentId) > 0 {
		if pObj, _ := manager.FetchById(parent.ParentId); pObj != nil {
			pObj.(*SCloudaccountSyncTask).markRunning(ctx, userCred)
		}
	}
	return task, nil
}

func (task *SCloudaccountSyncTask) getStatus() (string, error) {
	q := CloudaccountSyncTaskManager.Query("status").Equals("id", task.Id)
	var status string
	err := q.Row().Scan(&status)
	if err != nil {
		return "", errors.Wrap(err, "scan status")
	}
	return status, nil
}

func (task *SCloudaccountSyncTask) markRunning(ctx context.Context, userCred mcclient.TokenCredential) error {
	status, err := task.getStatus()
	if err != nil {
		return err
	}
	if status != api.CloudaccountSyncTaskStatusQueued {
		return nil
	}
	_, err = db.Update(task, func() error {
		task.Status = api.CloudaccountSyncTaskStatusRunning
		if task.StartedAt.IsZero() {
			task.StartedAt = time.Now().UTC()
		}
		return nil
	})
	return err
}

func (task *SCloudaccountSyncTask) markLeafRunning(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, task)
	defer lockman.ReleaseObject(ctx, task)

	status, err := task.getStatus()
	if err != nil {
		return err
	}
	if status == api.CloudaccountSyncTaskStatusCancel {
		return errors.Wrap(context.Canceled, "task cancelled")
	}
	if status != api.CloudaccountSyncTaskStatusQueued {
		return nil
	}
	_, err = db.Update(task, func() error {
		task.Status = api.CloudaccountSyncTaskStatusRunning
		task.StartedAt = time.Now().UTC()
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(task, db.ACT_START, nil, userCred)
	return nil
}

// completeProviderSyncTaskIfNoRegionLeaves 在订阅同步未挂上任何 region 叶子时收口 L2（并聚合 L1）。
// 典型场景：订阅下无 cloudproviderregion、均 disabled、或 StartRegionSyncTask 全部失败。
func completeProviderSyncTaskIfNoRegionLeaves(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	syncRange *SSyncRange,
) {
	if syncRange == nil || len(syncRange.ProviderSyncTaskId) == 0 {
		return
	}
	obj, err := db.FetchById(CloudaccountSyncTaskManager, syncRange.ProviderSyncTaskId)
	if err != nil {
		return
	}
	providerTask := obj.(*SCloudaccountSyncTask)
	if providerTask.IsCanceled() {
		return
	}
	children, err := CloudaccountSyncTaskManager.fetchTasksByFilter("", providerTask.Id, api.CloudaccountSyncTaskLevelRegion, nil)
	if err != nil {
		log.Errorf("completeProviderSyncTaskIfNoRegionLeaves fetch children: %v", err)
		return
	}
	if len(children) > 0 {
		return
	}
	_ = providerTask.completeLeaf(ctx, userCred, api.CloudaccountSyncTaskStatusDone, nil, "no region to sync", true)
}

func (task *SCloudaccountSyncTask) completeLeaf(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	status string,
	syncResults jsonutils.JSONObject,
	reason string,
	succ bool,
) error {
	lockman.LockObject(ctx, task)
	defer lockman.ReleaseObject(ctx, task)

	cur, err := task.getStatus()
	if err != nil {
		return err
	}
	if cur == api.CloudaccountSyncTaskStatusCancel {
		task.aggregateParent(ctx, userCred)
		return nil
	}
	_, err = db.Update(task, func() error {
		task.Status = status
		task.EndedAt = time.Now().UTC()
		if syncResults != nil {
			task.SyncResults = syncResults
		}
		return nil
	})
	if err != nil {
		return err
	}
	notes := map[string]string{"status": status, "reason": reason}
	db.OpsLog.LogEvent(task, db.ACT_DONE, notes, userCred)
	logclient.AddActionLogWithContext(ctx, task, logclient.ACT_DONE, notes, userCred, succ)
	task.aggregateParent(ctx, userCred)
	return nil
}

func (task *SCloudaccountSyncTask) aggregateParent(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(task.ParentId) == 0 {
		return
	}
	pObj, err := CloudaccountSyncTaskManager.FetchById(task.ParentId)
	if err != nil {
		log.Errorf("aggregateParent fetch %s: %v", task.ParentId, err)
		return
	}
	parent := pObj.(*SCloudaccountSyncTask)
	parent.tryAggregate(ctx, userCred)
}

func (task *SCloudaccountSyncTask) tryAggregate(ctx context.Context, userCred mcclient.TokenCredential) {
	lockman.LockObject(ctx, task)
	defer lockman.ReleaseObject(ctx, task)

	cur, err := task.getStatus()
	if err != nil {
		return
	}
	if cur == api.CloudaccountSyncTaskStatusCancel {
		task.aggregateParent(ctx, userCred)
		return
	}

	children, err := CloudaccountSyncTaskManager.fetchTasksByFilter("", task.Id, "", nil)
	if err != nil || len(children) == 0 {
		return
	}
	done, fail, cancel, active := 0, 0, 0, 0
	for i := range children {
		switch children[i].Status {
		case api.CloudaccountSyncTaskStatusDone:
			done++
		case api.CloudaccountSyncTaskStatusFail:
			fail++
		case api.CloudaccountSyncTaskStatusCancel:
			cancel++
		default:
			active++
		}
	}
	if active > 0 {
		return
	}
	newStatus := api.CloudaccountSyncTaskStatusDone
	if fail > 0 {
		newStatus = api.CloudaccountSyncTaskStatusFail
	} else if cancel > 0 && done == 0 {
		newStatus = api.CloudaccountSyncTaskStatusCancel
	} else if cancel > 0 {
		newStatus = api.CloudaccountSyncTaskStatusDone
	}
	_, err = db.Update(task, func() error {
		task.Status = newStatus
		task.EndedAt = time.Now().UTC()
		if task.StartedAt.IsZero() {
			task.StartedAt = task.EndedAt
		}
		return nil
	})
	if err != nil {
		log.Errorf("tryAggregate update %s: %v", task.Id, err)
		return
	}
	// 账号级汇总终态时兜底收回本树残留 sync_status（例如某叶子取消未进 DoSync）；勿 idle 账号下其它无关订阅
	if task.Level == api.CloudaccountSyncTaskLevelAccount && len(task.ParentId) == 0 {
		resetAccountTaskSyncStatuses(ctx, userCred, task)
	}
	task.aggregateParent(ctx, userCred)
}

func (task *SCloudaccountSyncTask) PerformCancel(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.CloudaccountSyncTaskCancelInput,
) (jsonutils.JSONObject, error) {
	reason := input.Reason
	if len(reason) == 0 {
		reason = "cancel task by admin manually"
	}
	status, err := task.getStatus()
	if err != nil {
		return nil, err
	}
	if status == api.CloudaccountSyncTaskStatusDone || status == api.CloudaccountSyncTaskStatusFail {
		return nil, httperrors.NewConflictError("task completed with status %s", status)
	}
	err = task.cancelCascade(ctx, userCred, reason)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (task *SCloudaccountSyncTask) cancelCascade(ctx context.Context, userCred mcclient.TokenCredential, reason string) error {
	// 先标自身 cancel，阻止并发再挂新叶子；仅对用户点击的这一层写操作日志
	err := task.cancelSelf(ctx, userCred, reason, true, true)
	if err != nil {
		return err
	}
	// 批量取消整棵子树（避免对每个 L3 加锁/写日志/markEndSync，账号级取消会超时）
	err = task.bulkCancelDescendants(ctx)
	if err != nil {
		return err
	}
	switch task.Level {
	case api.CloudaccountSyncTaskLevelRegion:
		if cpr := task.getCloudproviderregion(); cpr != nil {
			resetCloudproviderregionAfterCancel(ctx, userCred, cpr)
		}
	case api.CloudaccountSyncTaskLevelProvider:
		// 只收口本订阅，不影响同账号其它订阅
		resetProviderSyncStatuses(ctx, userCred, task.CloudproviderId)
	case api.CloudaccountSyncTaskLevelAccount:
		// 只收口本 L1 树下出现过的订阅，禁止 idle 账号下全部 provider
		resetAccountTaskSyncStatuses(ctx, userCred, task)
	}
	task.aggregateParent(ctx, userCred)
	return nil
}

// bulkCancelDescendants 用少量 SQL 将子孙 active 任务标为 cancel，并短循环消化建叶子竞态。
func (task *SCloudaccountSyncTask) bulkCancelDescendants(ctx context.Context) error {
	now := time.Now().UTC()
	table := CloudaccountSyncTaskManager.TableSpec().Name()

	cancelByParents := func(parentIds []string) (int64, error) {
		if len(parentIds) == 0 {
			return 0, nil
		}
		// SET status=?, ended_at=? WHERE parent_id IN (...) AND status IN (...)
		// 参数顺序必须与占位符一致：先 SET，再 parent_id，最后 status
		parentPh := make([]string, len(parentIds))
		statusPh := make([]string, len(api.CloudaccountSyncTaskStatusActive))
		args := make([]interface{}, 0, 2+len(parentIds)+len(api.CloudaccountSyncTaskStatusActive))
		args = append(args, api.CloudaccountSyncTaskStatusCancel, now)
		for i := range parentIds {
			parentPh[i] = "?"
			args = append(args, parentIds[i])
		}
		for i, s := range api.CloudaccountSyncTaskStatusActive {
			statusPh[i] = "?"
			args = append(args, s)
		}
		sqlStr := fmt.Sprintf(
			"UPDATE %s SET status=?, ended_at=? WHERE parent_id IN (%s) AND status IN (%s)",
			table, strings.Join(parentPh, ","), strings.Join(statusPh, ","),
		)
		result, err := sqlchemy.GetDB().ExecContext(ctx, sqlStr, args...)
		if err != nil {
			return 0, errors.Wrap(err, "bulk cancel by parent")
		}
		n, _ := result.RowsAffected()
		if n > 0 {
			log.Infof("bulkCancelDescendants parent=%s cancelled %d children", task.Id, n)
		}
		return n, nil
	}

	for round := 0; round < 5; round++ {
		var affected int64
		switch task.Level {
		case api.CloudaccountSyncTaskLevelAccount:
			// 取消 L2
			n, err := cancelByParents([]string{task.Id})
			if err != nil {
				return err
			}
			affected += n
			// 取消所有 L2 下的 L3（含本轮刚被取消的 L2）
			l2, err := CloudaccountSyncTaskManager.fetchTasksByFilter("", task.Id, api.CloudaccountSyncTaskLevelProvider, nil)
			if err != nil {
				return err
			}
			l2Ids := make([]string, 0, len(l2))
			for i := range l2 {
				l2Ids = append(l2Ids, l2[i].Id)
			}
			n, err = cancelByParents(l2Ids)
			if err != nil {
				return err
			}
			affected += n
		case api.CloudaccountSyncTaskLevelProvider:
			n, err := cancelByParents([]string{task.Id})
			if err != nil {
				return err
			}
			affected += n
		default:
			// region 叶子无子任务
			return nil
		}
		if affected == 0 {
			left, err := CloudaccountSyncTaskManager.countActiveDescendants(task)
			if err != nil {
				return err
			}
			if left == 0 {
				return nil
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

func (manager *SCloudaccountSyncTaskManager) countActiveDescendants(task *SCloudaccountSyncTask) (int, error) {
	q := manager.Query().In("status", api.CloudaccountSyncTaskStatusActive)
	switch task.Level {
	case api.CloudaccountSyncTaskLevelAccount:
		// parent_id = L1 or parent_id in L2
		l2q := manager.Query("id").Equals("parent_id", task.Id)
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(q.Field("parent_id"), task.Id),
			sqlchemy.In(q.Field("parent_id"), l2q.SubQuery()),
		))
	case api.CloudaccountSyncTaskLevelProvider:
		q = q.Equals("parent_id", task.Id)
	default:
		return 0, nil
	}
	return q.CountWithError()
}

func (task *SCloudaccountSyncTask) cancelSelf(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	reason string,
	allowTerminal bool,
	writeLog bool,
) error {
	lockman.LockObject(ctx, task)
	defer lockman.ReleaseObject(ctx, task)

	status, err := task.getStatus()
	if err != nil {
		return err
	}
	switch status {
	case api.CloudaccountSyncTaskStatusDone, api.CloudaccountSyncTaskStatusFail:
		if allowTerminal {
			return nil
		}
		return httperrors.NewConflictError("task completed with status %s", status)
	case api.CloudaccountSyncTaskStatusCancel:
		return nil
	}
	now := time.Now().UTC()
	_, err = db.Update(task, func() error {
		task.Status = api.CloudaccountSyncTaskStatusCancel
		if task.StartedAt.IsZero() {
			task.StartedAt = now
		}
		task.EndedAt = now
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "update cancel status")
	}
	if writeLog {
		notes := map[string]string{"status": status, "reason": reason}
		db.OpsLog.LogEvent(task, db.ACT_CANCEL, notes, userCred)
		logclient.AddActionLogWithContext(ctx, task, logclient.ACT_CANCEL, notes, userCred, true)
	}
	return nil
}

func (task *SCloudaccountSyncTask) getCloudproviderregion() *SCloudproviderregion {
	if len(task.CloudproviderId) == 0 || len(task.CloudregionId) == 0 {
		return nil
	}
	cpr, err := CloudproviderRegionManager.FetchByIds(task.CloudproviderId, task.CloudregionId)
	if err != nil {
		return nil
	}
	return cpr
}

// forceIdleCloudproviderregion 无条件将关节表置 idle，避免与 markSyncing 的 db.Update 竞态丢更新。
func forceIdleCloudproviderregion(cpr *SCloudproviderregion) {
	if cpr == nil {
		return
	}
	now := timeutils.UtcNow()
	table := CloudproviderRegionManager.TableSpec().Name()
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf("UPDATE %s SET sync_status=?, last_sync_end_at=? WHERE cloudprovider_id=? AND cloudregion_id=? AND sync_status<>?", table),
		api.CLOUD_PROVIDER_SYNC_STATUS_IDLE, now, cpr.CloudproviderId, cpr.CloudregionId, api.CLOUD_PROVIDER_SYNC_STATUS_IDLE,
	)
	if err != nil {
		log.Errorf("forceIdleCloudproviderregion %s/%s: %v", cpr.CloudproviderId, cpr.CloudregionId, err)
		return
	}
	cpr.SyncStatus = api.CLOUD_PROVIDER_SYNC_STATUS_IDLE
	cpr.LastSyncEndAt = now
}

// resetCloudproviderregionAfterCancel 将未跑完 DoSync 的关节表收回 idle，并尝试收口 provider/account。
func resetCloudproviderregionAfterCancel(ctx context.Context, userCred mcclient.TokenCredential, cpr *SCloudproviderregion) {
	if cpr == nil {
		return
	}
	forceIdleCloudproviderregion(cpr)
	provider, err := cpr.GetProvider()
	if err != nil {
		return
	}
	_ = provider.markEndSyncWithLock(ctx, userCred, false)
}

func bulkIdleCloudproviderRegions(ctx context.Context, providerId string) {
	if len(providerId) == 0 {
		return
	}
	table := CloudproviderRegionManager.TableSpec().Name()
	_, err := sqlchemy.GetDB().ExecContext(ctx,
		fmt.Sprintf("UPDATE %s SET sync_status=? WHERE cloudprovider_id=? AND sync_status<>?", table),
		api.CLOUD_PROVIDER_SYNC_STATUS_IDLE, providerId, api.CLOUD_PROVIDER_SYNC_STATUS_IDLE,
	)
	if err != nil {
		log.Errorf("bulkIdleCloudproviderRegions %s: %v", providerId, err)
	}
}

// resetProviderSyncStatuses 订阅级取消后收回该订阅下残留的 region/provider/account sync_status。
func resetProviderSyncStatuses(ctx context.Context, userCred mcclient.TokenCredential, providerId string) {
	if len(providerId) == 0 {
		return
	}
	bulkIdleCloudproviderRegions(ctx, providerId)
	pObj, err := CloudproviderManager.FetchById(providerId)
	if err != nil {
		return
	}
	provider := pObj.(*SCloudprovider)
	_ = provider.markEndSyncWithLock(ctx, userCred, false)
}

// resetAccountTaskSyncStatuses 账号级取消后，仅收回该 L1 任务树下的订阅/region 状态。
func resetAccountTaskSyncStatuses(ctx context.Context, userCred mcclient.TokenCredential, accountTask *SCloudaccountSyncTask) {
	if accountTask == nil {
		return
	}
	providers, err := CloudaccountSyncTaskManager.fetchTasksByFilter("", accountTask.Id, api.CloudaccountSyncTaskLevelProvider, nil)
	if err != nil {
		log.Errorf("resetAccountTaskSyncStatuses fetch providers: %v", err)
		return
	}
	seen := make(map[string]struct{}, len(providers))
	for i := range providers {
		pid := providers[i].CloudproviderId
		if len(pid) == 0 {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		resetProviderSyncStatuses(ctx, userCred, pid)
	}
	accountObj, err := CloudaccountManager.FetchById(accountTask.CloudaccountId)
	if err != nil {
		return
	}
	_ = accountObj.(*SCloudaccount).MarkEndSyncWithLock(ctx, userCred, false)
}

func (task *SCloudaccountSyncTask) LogProgress(ctx context.Context, act string, notes interface{}) {
	logclient.AddActionLogWithContext(ctx, task, act, notes, auth.AdminCredential(), true)
}

func (task *SCloudaccountSyncTask) IsCanceled() bool {
	status, err := task.getStatus()
	if err != nil {
		return false
	}
	return status == api.CloudaccountSyncTaskStatusCancel
}

func CheckCloudaccountSyncCanceled(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

// RunRegionSyncWithTask wraps DoSync with leaf task lifecycle and cancel watch.
func RunRegionSyncWithTask(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	leaf *SCloudaccountSyncTask,
	cpr *SCloudproviderregion,
	syncRange SSyncRange,
) {
	if leaf == nil {
		err := cpr.DoSync(ctx, userCred, syncRange)
		if err != nil {
			log.Errorf("DoSync failed %v", err)
		}
		return
	}

	// 父任务/自身已被批量取消：直接退出，勿再执行
	if leaf.IsCanceled() {
		resetCloudproviderregionAfterCancel(ctx, userCred, cpr)
		return
	}

	runCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		check := func() bool {
			if leaf.IsCanceled() {
				cancelFunc()
				return true
			}
			return false
		}
		if check() {
			return
		}
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				if check() {
					return
				}
			}
		}
	}()

	err := leaf.markLeafRunning(runCtx, userCred)
	if err != nil {
		_ = leaf.completeLeaf(ctx, userCred, api.CloudaccountSyncTaskStatusCancel, nil, err.Error(), false)
		// 未进入 DoSync，cpr 仍停在 queued，需收回
		resetCloudproviderregionAfterCancel(ctx, userCred, cpr)
		return
	}

	leaf.LogProgress(runCtx, logclient.ACT_PROGRESS, map[string]string{
		"event":    "region_sync_start",
		"provider": leaf.CloudproviderId,
		"region":   leaf.CloudregionId,
	})

	err = cpr.DoSync(runCtx, userCred, syncRange)
	if runCtx.Err() != nil || errors.Cause(err) == context.Canceled || leaf.IsCanceled() {
		_ = leaf.completeLeaf(ctx, userCred, api.CloudaccountSyncTaskStatusCancel, nil, "cancelled", false)
		// DoSync 的 defer 一般会 markEndSync；若取消发生在 markSyncing 之前仍可能残留
		resetCloudproviderregionAfterCancel(ctx, userCred, cpr)
		return
	}
	results := cpr.SyncResults
	if err != nil {
		_ = leaf.completeLeaf(ctx, userCred, api.CloudaccountSyncTaskStatusFail, results, err.Error(), false)
		return
	}
	_ = leaf.completeLeaf(ctx, userCred, api.CloudaccountSyncTaskStatusDone, results, "", true)
}
