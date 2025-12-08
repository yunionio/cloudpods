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
	"fmt"
	"strings"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/notify"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func parseEvent(es string) (api.SNotifyEvent, error) {
	es = strings.ToLower(es)
	ess := strings.Split(es, api.DelimiterInEvent)
	if len(ess) != 2 && len(ess) != 3 {
		return api.SNotifyEvent{}, fmt.Errorf("invalid event string %q", es)
	}
	event := api.Event.WithResourceType(ess[0]).WithAction(api.SAction(ess[1]))
	if len(ess) == 3 {
		event = event.WithResult(api.SResult(ess[2]))
	}
	return event, nil
}

type STopicManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var TopicManager *STopicManager

func init() {
	TopicManager = &STopicManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			STopic{},
			"topic_tbl",
			"topic",
			"topics",
		),
	}
	TopicManager.SetVirtualObject(TopicManager)
}

// 消息订阅
type STopic struct {
	db.SEnabledStatusStandaloneResourceBase

	Type        string               `width:"20" nullable:"false" create:"required" update:"user" list:"user"`
	Results     tristate.TriState    `default:"true"`
	TitleCn     string               `length:"medium" nullable:"true" charset:"utf8" list:"user" update:"user" create:"optional"`
	TitleEn     string               `length:"medium" nullable:"true" charset:"utf8" list:"user" update:"user" create:"optional"`
	ContentCn   string               `length:"medium" nullable:"true" charset:"utf8" list:"user" update:"user" create:"optional"`
	ContentEn   string               `length:"medium" nullable:"true" charset:"utf8" list:"user" update:"user" create:"optional"`
	GroupKeys   *api.STopicGroupKeys `nullable:"true" list:"user" update:"user" create:"optional"`
	AdvanceDays []int                `nullable:"true" charset:"utf8" list:"user" update:"user" create:"optional"`

	WebconsoleDisable tristate.TriState `default:"false" list:"user" update:"user" create:"optional"`
}

const (
	DefaultResourceCreateDelete       = "resource create or delete"
	DefaultResourceChangeConfig       = "resource change config"
	DefaultResourceUpdate             = "resource update"
	DefaultResourceReleaseDue1Day     = "resource release due 1 day"
	DefaultResourceReleaseDue3Day     = "resource release due 3 day"
	DefaultResourceReleaseDue30Day    = "resource release due 30 day"
	DefaultResourceRelease            = "resource release"
	DefaultScheduledTaskExecute       = "scheduled task execute"
	DefaultScalingPolicyExecute       = "scaling policy execute"
	DefaultSnapshotPolicyExecute      = "snapshot policy execute"
	DefaultResourceOperationFailed    = "resource operation failed"
	DefaultResourceOperationSuccessed = "resource operation successed"
	DefaultResourceSync               = "resource sync"
	DefaultSystemExceptionEvent       = "system exception event"
	DefaultChecksumTestFailed         = "checksum test failed"
	DefaultUserLock                   = "user lock"
	DefaultActionLogExceedCount       = "action log exceed count"
	DefaultSyncAccountStatus          = "cloud account sync status"
	DefaultPasswordExpireDue1Day      = "password expire due 1 day"
	DefaultPasswordExpireDue7Day      = "password expire due 7 day"
	DefaultPasswordExpire             = "password expire"
	DefaultNetOutOfSync               = "net out of sync"
	DefaultMysqlOutOfSync             = "mysql out of sync"
	DefaultServiceAbnormal            = "service abnormal"
	DefaultServerPanicked             = "server panicked"
	DefaultAttachOrDetach             = "resource attach or detach"
	DefaultIsolatedDeviceChanged      = "isolated device changed"
	DefaultStatusChanged              = "resource status changed"
)

func (sm *STopicManager) InitializeData() error {
	initSNames := sets.NewString(
		DefaultResourceCreateDelete,
		DefaultResourceChangeConfig,
		DefaultResourceUpdate,
		DefaultScheduledTaskExecute,
		DefaultScalingPolicyExecute,
		DefaultSnapshotPolicyExecute,
		DefaultResourceOperationFailed,
		DefaultResourceSync,
		DefaultSystemExceptionEvent,
		DefaultChecksumTestFailed,
		DefaultUserLock,
		DefaultActionLogExceedCount,
		DefaultSyncAccountStatus,
		DefaultNetOutOfSync,
		DefaultMysqlOutOfSync,
		DefaultServiceAbnormal,
		DefaultServerPanicked,
		DefaultPasswordExpire,
		DefaultResourceRelease,
		DefaultResourceOperationSuccessed,
		DefaultAttachOrDetach,
		DefaultIsolatedDeviceChanged,
		DefaultStatusChanged,
	)
	q := sm.Query()
	topics := make([]STopic, 0, initSNames.Len())
	err := db.FetchModelObjects(sm, q, &topics)
	if err != nil {
		return errors.Wrap(err, "unable to FetchModelObjects")
	}
	nameTopicMap := make(map[string]*STopic, len(topics))
	for i := range topics {
		t := &topics[i]
		initSNames.Delete(t.Name)
		nameTopicMap[t.Name] = t
	}
	for _, name := range initSNames.UnsortedList() {
		nameTopicMap[name] = nil
	}
	ctx := context.Background()
	for name, topic := range nameTopicMap {
		isNew := false
		t := new(STopic)
		if topic == nil {
			t.Id = db.DefaultUUIDGenerator()
			isNew = true
		} else {
			t.Id = topic.Id
		}
		t.Name = name
		t.Enabled = tristate.True
		switch name {
		case DefaultResourceCreateDelete:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
			t.ContentCn = api.COMMON_CONTENT_CN
			t.ContentEn = api.COMMON_CONTENT_EN
			t.TitleCn = api.COMMON_TITLE_CN
			t.TitleEn = api.COMMON_TITLE_EN
		case DefaultResourceChangeConfig:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
			t.ContentCn = api.COMMON_CONTENT_CN
			t.ContentEn = api.COMMON_CONTENT_EN
			t.TitleCn = api.COMMON_TITLE_CN
			t.TitleEn = api.COMMON_TITLE_EN
		case DefaultResourceUpdate:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
			t.ContentCn = api.UPDATE_CONTENT_CN
			t.ContentEn = api.UPDATE_CONTENT_EN
			t.TitleCn = api.UPDATE_TITLE_CN
			t.TitleEn = api.UPDATE_TITLE_EN
		case DefaultScheduledTaskExecute:
			t.Type = api.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.SCHEDULEDTASK_EXECUTE_CONTENT_CN
			t.ContentEn = api.SCHEDULEDTASK_EXECUTE_CONTENT_EN
			t.TitleCn = api.SCHEDULEDTASK_EXECUTE_TITLE_CN
			t.TitleEn = api.SCHEDULEDTASK_EXECUTE_TITLE_EN
		case DefaultScalingPolicyExecute:
			t.Type = api.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.SCALINGPOLICY_EXECUTE_CONTENT_CN
			t.ContentEn = api.SCALINGPOLICY_EXECUTE_CONTENT_EN
			t.TitleCn = api.SCALINGPOLICY_EXECUTE_TITLE_CN
			t.TitleEn = api.SCALINGPOLICY_EXECUTE_TITLE_EN
		case DefaultSnapshotPolicyExecute:
			t.Type = api.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.SNAPSHOTPOLICY_EXECUTE_CONTENT_CN
			t.ContentEn = api.SNAPSHOTPOLICY_EXECUTE_CONTENT_EN
			t.TitleCn = api.SNAPSHOTPOLICY_EXECUTE_TITLE_CN
			t.TitleEn = api.SNAPSHOTPOLICY_EXECUTE_TITLE_EN
		case DefaultResourceOperationFailed:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.Results = tristate.False
		case DefaultResourceOperationSuccessed:
			t.Results = tristate.True
			t.Type = api.TOPIC_TYPE_RESOURCE
		case DefaultResourceSync:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.WebconsoleDisable = tristate.True
			t.Results = tristate.True
			t.ContentCn = api.COMMON_CONTENT_CN
			t.ContentEn = api.COMMON_CONTENT_EN
			t.TitleCn = api.COMMON_TITLE_CN
			t.TitleEn = api.COMMON_TITLE_EN
			groupKeys := []string{"action_display"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultSystemExceptionEvent:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.Results = tristate.False
			t.ContentCn = api.EXCEPTION_CONTENT_CN
			t.ContentEn = api.EXCEPTION_CONTENT_EN
			t.TitleCn = api.EXCEPTION_TITLE_CN
			t.TitleEn = api.EXCEPTION_TITLE_EN
		case DefaultChecksumTestFailed:
			t.Type = api.TOPIC_TYPE_SECURITY
			t.Results = tristate.False
			t.ContentCn = api.CHECKSUM_TEST_FAILED_CONTENT_CN
			t.ContentEn = api.CHECKSUM_TEST_FAILED_CONTENT_EN
			t.TitleCn = api.CHECKSUM_TEST_FAILED_TITLE_CN
			t.TitleEn = api.CHECKSUM_TEST_FAILED_TITLE_EN
		case DefaultUserLock:
			t.Type = api.TOPIC_TYPE_SECURITY
			t.Results = tristate.True
			t.ContentCn = api.USER_LOCK_CONTENT_CN
			t.ContentEn = api.USER_LOCK_CONTENT_EN
			t.TitleCn = api.USER_LOCK_TITLE_CN
			t.TitleEn = api.USER_LOCK_TITLE_EN
		case DefaultActionLogExceedCount:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
			t.ContentCn = api.ACTION_LOG_EXCEED_COUNT_CONTENT_CN
			t.ContentEn = api.ACTION_LOG_EXCEED_COUNT_CONTENT_EN
			t.TitleCn = api.ACTION_LOG_EXCEED_COUNT_TITLE_CN
			t.TitleEn = api.ACTION_LOG_EXCEED_COUNT_TITLE_EN
		case DefaultSyncAccountStatus:
			t.Type = api.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.SYNC_ACCOUNT_STATUS_CONTENT_CN
			t.ContentEn = api.SYNC_ACCOUNT_STATUS_CONTENT_EN
			t.TitleCn = api.SYNC_ACCOUNT_STATUS_TITLE_CN
			t.TitleEn = api.SYNC_ACCOUNT_STATUS_TITLE_EN
			groupKeys := []string{"name"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultNetOutOfSync:
			t.Type = api.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.NET_OUT_OF_SYNC_CONTENT_CN
			t.ContentEn = api.NET_OUT_OF_SYNC_CONTENT_EN
			t.TitleCn = api.NET_OUT_OF_SYNC_TITLE_CN
			t.TitleEn = api.NET_OUT_OF_SYNC_TITLE_EN
			groupKeys := []string{"service_name"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultMysqlOutOfSync:
			t.Type = api.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.MYSQL_OUT_OF_SYNC_CONTENT_CN
			t.ContentEn = api.MYSQL_OUT_OF_SYNC_CONTENT_EN
			t.TitleCn = api.MYSQL_OUT_OF_SYNC_TITLE_CN
			t.TitleEn = api.MYSQL_OUT_OF_SYNC_TITLE_EN
			groupKeys := []string{"ip"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultServiceAbnormal:
			t.Results = tristate.True
			t.Type = api.TOPIC_TYPE_AUTOMATED_PROCESS
			t.ContentCn = api.SERVICE_ABNORMAL_CONTENT_CN
			t.ContentEn = api.SERVICE_ABNORMAL_CONTENT_EN
			t.TitleCn = api.SERVICE_ABNORMAL_TITLE_CN
			t.TitleEn = api.SERVICE_ABNORMAL_TITLE_EN
			groupKeys := []string{"service_name"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultServerPanicked:
			t.Results = tristate.False
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.ContentCn = api.SERVER_PANICKED_CONTENT_CN
			t.ContentEn = api.SERVER_PANICKED_CONTENT_EN
			t.TitleCn = api.SERVER_PANICKED_TITLE_CN
			t.TitleEn = api.SERVER_PANICKED_TITLE_EN
			groupKeys := []string{"name"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultPasswordExpire:
			t.AdvanceDays = []int{1, 7}
			t.Type = api.TOPIC_TYPE_SECURITY
			t.Results = tristate.True
			t.ContentCn = api.PWD_EXPIRE_SOON_CONTENT_CN
			t.ContentEn = api.PWD_EXPIRE_SOON_CONTENT_EN
			t.TitleCn = api.PWD_EXPIRE_SOON_TITLE_CN
			t.TitleEn = api.PWD_EXPIRE_SOON_TITLE_EN
		case DefaultResourceRelease:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.AdvanceDays = []int{1, 7, 30}
			t.Results = tristate.True
			t.ContentCn = api.EXPIRED_RELEASE_CONTENT_CN
			t.ContentEn = api.EXPIRED_RELEASE_CONTENT_EN
			t.TitleCn = api.EXPIRED_RELEASE_TITLE_CN
			t.TitleEn = api.EXPIRED_RELEASE_TITLE_EN
		case DefaultAttachOrDetach:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
		case DefaultIsolatedDeviceChanged:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
		case DefaultStatusChanged:
			t.Type = api.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
			t.ContentCn = api.STATUS_CHANGED_CONTENT_CN
			t.ContentEn = api.STATUS_CHANGED_CONTENT_EN
			t.TitleCn = api.STATUS_CHANGED_TITLE_CN
			t.TitleEn = api.STATUS_CHANGED_TITLE_EN
		}
		if topic == nil {
			err := sm.TableSpec().Insert(ctx, t)
			if err != nil {
				return errors.Wrapf(err, "unable to insert %s", name)
			}
		} else {
			if t.Name == DefaultResourceReleaseDue3Day || t.Name == DefaultResourceReleaseDue30Day || t.Name == DefaultResourceReleaseDue1Day {
				err = topic.Delete(ctx, auth.AdminCredential())
				if err != nil {
					log.Errorf("delete %s err %s", topic.Name, err.Error())
				}
				continue
			}
			if t.Name == DefaultPasswordExpireDue7Day || t.Name == DefaultPasswordExpireDue1Day {
				err = topic.Delete(ctx, auth.AdminCredential())
				if err != nil {
					log.Errorf("delete %s err %s", topic.Name, err.Error())
				}
				continue
			}

			_, err := db.Update(topic, func() error {
				topic.Name = t.Name
				// topic.Resources = t.Resources
				// topic.Actions = t.Actions
				topic.Type = t.Type
				topic.Results = t.Results
				topic.WebconsoleDisable = t.WebconsoleDisable
				topic.GroupKeys = t.GroupKeys
				if len(topic.AdvanceDays) == 0 {
					topic.AdvanceDays = t.AdvanceDays
				}
				if len(topic.ContentCn) == 0 || topic.Name == DefaultPasswordExpire || topic.Name == DefaultResourceRelease {
					if len(t.ContentCn) == 0 {
						t.ContentCn = api.COMMON_CONTENT_CN
					}
					topic.ContentCn = t.ContentCn
				}
				if len(topic.ContentEn) == 0 || topic.Name == DefaultPasswordExpire || topic.Name == DefaultResourceRelease {
					if len(t.ContentEn) == 0 {
						t.ContentEn = api.COMMON_CONTENT_EN
					}
					topic.ContentEn = t.ContentEn
				}
				if len(topic.TitleCn) == 0 || topic.Name == DefaultPasswordExpire || topic.Name == DefaultResourceRelease {
					if len(t.TitleCn) == 0 {
						t.TitleCn = api.COMMON_TITLE_CN
					}
					topic.TitleCn = t.TitleCn
				}
				if len(topic.TitleEn) == 0 || topic.Name == DefaultPasswordExpire || topic.Name == DefaultResourceRelease {
					if len(t.TitleEn) == 0 {
						t.TitleEn = api.COMMON_TITLE_EN
					}
					topic.TitleEn = t.TitleEn
				}
				return nil
			})
			if err != nil {
				return errors.Wrapf(err, "unable to update topic %s", topic.Name)
			}
		}
		acnt, rcnt := 0, 0
		if !gotypes.IsNil(topic) {
			acnt = TopicActionManager.Query().Equals("topic_id", topic.Id).Count()
			rcnt = TopicResourceManager.Query().Equals("topic_id", topic.Id).Count()
		}
		if isNew || acnt == 0 || rcnt == 0 {
			initTopicElement(name, t)
		}
	}
	return nil
}

// 新建关联关系
func initTopicElement(name string, t *STopic) {
	switch name {
	case DefaultResourceCreateDelete:
		t.addResources(
			api.TOPIC_RESOURCE_HOST,
			api.TOPIC_RESOURCE_SERVER,
			api.TOPIC_RESOURCE_SCALINGGROUP,
			api.TOPIC_RESOURCE_IMAGE,
			api.TOPIC_RESOURCE_DISK,
			api.TOPIC_RESOURCE_SNAPSHOT,
			api.TOPIC_RESOURCE_INSTANCESNAPSHOT,
			api.TOPIC_RESOURCE_SNAPSHOTPOLICY,
			api.TOPIC_RESOURCE_NETWORK,
			api.TOPIC_RESOURCE_EIP,
			api.TOPIC_RESOURCE_LOADBALANCER,
			api.TOPIC_RESOURCE_LOADBALANCERACL,
			api.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE,
			api.TOPIC_RESOURCE_BUCKET,
			api.TOPIC_RESOURCE_DBINSTANCE,
			api.TOPIC_RESOURCE_ELASTICCACHE,
			api.TOPIC_RESOURCE_BAREMETAL,
			api.TOPIC_RESOURCE_SECGROUP,
			api.TOPIC_RESOURCE_FILESYSTEM,
			api.TOPIC_RESOURCE_NATGATEWAY,
			api.TOPIC_RESOURCE_VPC,
			api.TOPIC_RESOURCE_CDNDOMAIN,
			api.TOPIC_RESOURCE_WAF,
			api.TOPIC_RESOURCE_KAFKA,
			api.TOPIC_RESOURCE_ELASTICSEARCH,
			api.TOPIC_RESOURCE_MONGODB,
			api.TOPIC_RESOURCE_DNSZONE,
			api.TOPIC_RESOURCE_DNSRECORDSET,
			api.TOPIC_RESOURCE_LOADBALANCERLISTENER,
			api.TOPIC_RESOURCE_LOADBALANCERBACKEDNGROUP,
			api.TOPIC_RESOURCE_PROJECT,
		)
		t.addAction(
			api.ActionCreate,
			api.ActionDelete,
			api.ActionPendingDelete,
		)
	case DefaultResourceChangeConfig:
		t.addResources(
			api.TOPIC_RESOURCE_HOST,
			api.TOPIC_RESOURCE_SERVER,
			api.TOPIC_RESOURCE_DBINSTANCE,
			api.TOPIC_RESOURCE_ELASTICCACHE,
		)
		t.addAction(api.ActionChangeConfig)
	case DefaultResourceUpdate:
		t.addResources(
			api.TOPIC_RESOURCE_SERVER,
			api.TOPIC_RESOURCE_DBINSTANCE,
			api.TOPIC_RESOURCE_ELASTICCACHE,
			api.TOPIC_RESOURCE_USER,
			api.TOPIC_RESOURCE_HOST,
			api.TOPIC_RESOURCE_PROJECT,
		)
		t.addAction(api.ActionUpdate)
		t.addAction(api.ActionRebuildRoot)
		t.addAction(api.ActionResetPassword)
		t.addAction(api.ActionChangeIpaddr)
	case DefaultScheduledTaskExecute:
		t.addResources(api.TOPIC_RESOURCE_SCHEDULEDTASK)
		t.addAction(api.ActionExecute)
	case DefaultScalingPolicyExecute:
		t.addResources(api.TOPIC_RESOURCE_SCALINGPOLICY)
		t.addAction(api.ActionExecute)
	case DefaultSnapshotPolicyExecute:
		t.addResources(api.TOPIC_RESOURCE_SNAPSHOTPOLICY)
		t.addAction(api.ActionExecute)
	case DefaultResourceOperationFailed:
		t.addResources(
			api.TOPIC_RESOURCE_SERVER,
			api.TOPIC_RESOURCE_EIP,
			api.TOPIC_RESOURCE_LOADBALANCER,
			api.TOPIC_RESOURCE_DBINSTANCE,
			api.TOPIC_RESOURCE_ELASTICCACHE,
			api.TOPIC_RESOURCE_CLOUDPHONE,
		)
		t.addAction(
			api.ActionStart,
			api.ActionStop,
			api.ActionRestart,
			api.ActionReset,
			api.ActionAttach,
			api.ActionDetach,
			api.ActionCreate,
			api.ActionSyncStatus,
			api.ActionRebuildRoot,
			api.ActionChangeConfig,
			api.ActionCreateBackupServer,
			api.ActionDelBackupServer,
			api.ActionMigrate,
		)
	case DefaultResourceOperationSuccessed:
		t.addResources(
			api.TOPIC_RESOURCE_SERVER,
			api.TOPIC_RESOURCE_CLOUDPHONE,
		)
		t.addAction(
			api.ActionStart,
			api.ActionStop,
			api.ActionRestart,
			api.ActionReset,
			api.ActionCreateBackupServer,
		)
	case DefaultResourceSync:
		t.addResources(
			api.TOPIC_RESOURCE_SERVER,
			api.TOPIC_RESOURCE_DISK,
			api.TOPIC_RESOURCE_DBINSTANCE,
			api.TOPIC_RESOURCE_ELASTICCACHE,
			api.TOPIC_RESOURCE_LOADBALANCER,
			api.TOPIC_RESOURCE_EIP,
			api.TOPIC_RESOURCE_VPC,
			api.TOPIC_RESOURCE_NETWORK,
			api.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE,
			api.TOPIC_RESOURCE_DNSZONE,
			api.TOPIC_RESOURCE_NATGATEWAY,
			api.TOPIC_RESOURCE_BUCKET,
			api.TOPIC_RESOURCE_FILESYSTEM,
			api.TOPIC_RESOURCE_WEBAPP,
			api.TOPIC_RESOURCE_CDNDOMAIN,
			api.TOPIC_RESOURCE_WAF,
			api.TOPIC_RESOURCE_KAFKA,
			api.TOPIC_RESOURCE_ELASTICSEARCH,
			api.TOPIC_RESOURCE_MONGODB,
			api.TOPIC_RESOURCE_DNSRECORDSET,
			api.TOPIC_RESOURCE_LOADBALANCERLISTENER,
			api.TOPIC_RESOURCE_LOADBALANCERBACKEDNGROUP,
		)
		t.addAction(
			api.ActionSyncCreate,
			api.ActionSyncUpdate,
			api.ActionSyncDelete,
		)
	case DefaultSystemExceptionEvent:
		t.addResources(
			api.TOPIC_RESOURCE_HOST,
			api.TOPIC_RESOURCE_TASK,
		)
		t.addAction(
			api.ActionSystemPanic,
			api.ActionSystemException,
			api.ActionOffline,
		)
	case DefaultChecksumTestFailed:
		t.addResources(
			api.TOPIC_RESOURCE_DB_TABLE_RECORD,
			api.TOPIC_RESOURCE_VM_INTEGRITY_CHECK,
			api.TOPIC_RESOURCE_CLOUDPODS_COMPONENT,
			api.TOPIC_RESOURCE_SNAPSHOT,
			api.TOPIC_RESOURCE_IMAGE,
		)
		t.addAction(
			api.ActionChecksumTest,
		)
	case DefaultUserLock:
		t.addResources(
			api.TOPIC_RESOURCE_USER,
		)
		t.addAction(
			api.ActionLock,
		)
	case DefaultActionLogExceedCount:
		t.addResources(
			api.TOPIC_RESOURCE_ACTION_LOG,
		)
		t.addAction(
			api.ActionExceedCount,
		)
	case DefaultSyncAccountStatus:
		t.addResources(
			api.TOPIC_RESOURCE_ACCOUNT_STATUS,
		)
		t.addAction(
			api.ActionSyncAccountStatus,
		)
	case DefaultNetOutOfSync:
		t.addResources(
			api.TOPIC_RESOURCE_NET,
		)
		t.addAction(
			api.ActionNetOutOfSync,
		)
	case DefaultMysqlOutOfSync:
		t.addResources(
			api.TOPIC_RESOURCE_DBINSTANCE,
		)
		t.addAction(
			api.ActionMysqlOutOfSync,
		)
	case DefaultServiceAbnormal:
		t.addResources(
			api.TOPIC_RESOURCE_SERVICE,
		)
		t.addAction(
			api.ActionServiceAbnormal,
		)
	case DefaultServerPanicked:
		t.addResources(
			api.TOPIC_RESOURCE_SERVER,
		)
		t.addAction(
			api.ActionServerPanicked,
		)
	case DefaultPasswordExpire:
		t.addResources(
			api.TOPIC_RESOURCE_USER,
		)
		t.addAction(
			api.ActionPasswordExpireSoon,
		)
	case DefaultResourceRelease:
		t.addResources(
			api.TOPIC_RESOURCE_SERVER,
			api.TOPIC_RESOURCE_DISK,
			api.TOPIC_RESOURCE_EIP,
			api.TOPIC_RESOURCE_LOADBALANCER,
			api.TOPIC_RESOURCE_DBINSTANCE,
			api.TOPIC_RESOURCE_ELASTICCACHE,
		)
		t.addAction(api.ActionExpiredRelease)
	case DefaultAttachOrDetach:
		t.addResources(
			api.TOPIC_RESOURCE_HOST,
			api.TOPIC_RESOURCE_CLOUDPHONE,
		)
		t.addAction(
			api.ActionAttach,
			api.ActionDetach,
		)
	case DefaultIsolatedDeviceChanged:
		t.addResources(
			api.TOPIC_RESOURCE_HOST,
		)
		t.addAction(
			api.ActionIsolatedDeviceCreate,
			api.ActionIsolatedDeviceUpdate,
			api.ActionIsolatedDeviceDelete,
		)
	case DefaultStatusChanged:
		t.addResources(
			api.TOPIC_RESOURCE_SERVER,
			api.TOPIC_RESOURCE_HOST,
		)
		t.addAction(
			api.ActionStatusChanged,
		)
	}
}

func (sm *STopicManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.TopicListInput) (*sqlchemy.SQuery, error) {
	q, err := sm.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = sm.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (sm *STopicManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.TopicDetails {
	sRows := sm.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]api.TopicDetails, len(objs))
	topicIds := make([]string, len(objs))
	for i := range rows {
		rows[i].StandaloneResourceDetails = sRows[i]
		ss := objs[i].(*STopic)
		topicIds[i] = ss.Id
	}
	resources, resourceMap := []STopicResource{}, map[string][]string{}
	err := TopicResourceManager.Query().In("topic_id", topicIds).All(&resources)
	if err != nil {
		log.Errorf("query resources error: %v", err)
		return rows
	}
	for _, r := range resources {
		_, ok := resourceMap[r.TopicId]
		if !ok {
			resourceMap[r.TopicId] = []string{}
		}
		resourceMap[r.TopicId] = append(resourceMap[r.TopicId], r.ResourceId)
	}
	actions, actionMap := []STopicAction{}, map[string][]string{}
	err = TopicActionManager.Query().In("topic_id", topicIds).All(&actions)
	if err != nil {
		log.Errorf("query actions error: %v", err)
		return rows
	}
	for _, a := range actions {
		_, ok := actionMap[a.TopicId]
		if !ok {
			actionMap[a.TopicId] = []string{}
		}
		actionMap[a.TopicId] = append(actionMap[a.TopicId], a.ActionId)
	}
	for i := range rows {
		rows[i].Resources, _ = resourceMap[topicIds[i]]
		rows[i].Actions, _ = actionMap[topicIds[i]]
	}

	return rows
}

func (sm *STopicManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.STopicCreateInput,
) (*api.STopicCreateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = sm.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}
	if !utils.IsInStringArray(input.Type, []string{
		api.TOPIC_TYPE_RESOURCE,
		api.TOPIC_TYPE_AUTOMATED_PROCESS,
		api.TOPIC_TYPE_SECURITY,
	}) {
		return nil, httperrors.NewInputParameterError("invalid type %s", input.Type)
	}
	input.Status = apis.STATUS_AVAILABLE
	return input, nil
}

func (tp *STopic) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	tp.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := &api.STopicCreateInput{}
	data.Unmarshal(input)
	for _, resource := range input.Resources {
		r := &STopicResource{
			ResourceId: resource,
			TopicId:    tp.Id,
		}
		r.SetModelManager(TopicResourceManager, r)
		TopicResourceManager.TableSpec().Insert(ctx, r)
	}
	for _, action := range input.Actions {
		a := &STopicAction{
			ActionId: action,
			TopicId:  tp.Id,
		}
		a.SetModelManager(TopicActionManager, a)
		TopicActionManager.TableSpec().Insert(ctx, a)
	}
}

func (ss *STopic) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.TopicUpdateInput) (*api.TopicUpdateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = ss.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (tp *STopic) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	tp.SEnabledStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	input := api.TopicUpdateInput{}
	jsonutils.Update(&input, data)
	if len(input.Resources) > 0 {
		tp.cleanResources()
		for _, res := range input.Resources {
			r := &STopicResource{
				ResourceId: res,
				TopicId:    tp.Id,
			}
			r.SetModelManager(TopicResourceManager, r)
			TopicResourceManager.TableSpec().Insert(ctx, r)
		}
	}
	if len(input.Actions) > 0 {
		tp.cleanActions()
		for _, action := range input.Actions {
			a := &STopicAction{
				ActionId: action,
				TopicId:  tp.Id,
			}
			a.SetModelManager(TopicActionManager, a)
			TopicActionManager.TableSpec().Insert(ctx, a)
		}
	}
}

func (ss *STopic) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return ss.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, info)
}

func (tp *STopic) cleanResources() error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where topic_id = ?",
			TopicResourceManager.TableSpec().Name(),
		), tp.Id,
	)
	return err
}

func (tp *STopic) cleanActions() error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where topic_id = ?",
			TopicActionManager.TableSpec().Name(),
		), tp.Id,
	)
	return err
}

func (tp *STopic) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := tp.cleanResources()
	if err != nil {
		return errors.Wrapf(err, "cleanResources")
	}
	err = tp.cleanActions()
	if err != nil {
		return errors.Wrapf(err, "cleanActions")
	}
	return tp.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (s *STopic) addResources(resources ...string) {
	for i := range resources {
		if TopicResourceManager.Query().Equals("topic_id", s.Id).Equals("resource_id", resources[i]).Count() == 0 {
			TopicResourceManager.TableSpec().InsertOrUpdate(context.Background(), &STopicResource{
				ResourceId: resources[i],
				TopicId:    s.Id,
			})
		}
	}
}

func (s *STopic) addAction(actions ...api.SAction) {
	for i := range actions {
		if TopicActionManager.Query().Equals("topic_id", s.Id).Equals("action_id", actions[i]).Count() == 0 {
			TopicActionManager.TableSpec().InsertOrUpdate(context.Background(), &STopicAction{
				ActionId: string(actions[i]),
				TopicId:  s.Id,
			})
		}
	}
}

func (s *STopic) GetResources() ([]STopicResource, error) {
	ret := []STopicResource{}
	q := TopicResourceManager.Query().Equals("topic_id", s.Id)
	err := db.FetchModelObjects(TopicResourceManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *STopic) GetActions() ([]STopicAction, error) {
	ret := []STopicAction{}
	q := TopicActionManager.Query().Equals("topic_id", s.Id)
	err := db.FetchModelObjects(TopicActionManager, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (sm *STopicManager) GetTopicByEvent(resourceType string, action api.SAction, isFailed api.SResult) (*STopic, error) {
	topics, err := sm.GetTopicsByEvent(resourceType, action, isFailed)
	if err != nil {
		return nil, errors.Wrapf(err, "GetTopicsByEvent")
	}
	if len(topics) == 0 {
		return nil, httperrors.NewResourceNotFoundError("no available topic found by %s %s", action, resourceType)
	}
	// free memory in time
	if len(topics) > 1 {
		return nil, httperrors.NewResourceNotFoundError("duplicates %d topics found by %s %s", len(topics), action, resourceType)
	}
	return &topics[0], nil
}

func (sm *STopicManager) GetTopicsByEvent(resourceType string, action api.SAction, isFailed api.SResult) ([]STopic, error) {
	q := sm.Query()
	q = q.Equals("results", isFailed).IsTrue("enabled")
	actionQ := TopicActionManager.Query("topic_id").Equals("action_id", action).SubQuery()
	q = q.In("id", actionQ)
	resourceQ := TopicResourceManager.Query("topic_id").Equals("resource_id", resourceType).SubQuery()
	q = q.In("id", resourceQ)
	var topics []STopic
	err := db.FetchModelObjects(sm, q, &topics)
	if err != nil {
		return nil, errors.Wrap(err, "unable to FetchModelObjects")
	}
	return topics, err
}

func (manager *STopicManager) TopicByEvent(eventStr string) (*STopic, error) {
	event, err := parseEvent(eventStr)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse event %q", event)
	}
	q := manager.Query().Equals("results", event.Result() == api.ResultSucceed)
	actionQ := TopicActionManager.Query("topic_id").Equals("action_id", event.Action()).SubQuery()
	q = q.In("id", actionQ)
	resourceQ := TopicResourceManager.Query("topic_id").Equals("resource_id", event.ResourceType()).SubQuery()
	q = q.In("id", resourceQ)
	var topics []STopic
	err = db.FetchModelObjects(manager, q, &topics)
	if err != nil {
		return nil, errors.Wrap(err, "unable to FetchModelObjects")
	}
	for i := range topics {
		if topics[i].Enabled.IsFalse() {
			return nil, errors.Wrapf(errors.ErrInvalidStatus, "topic %s disabled", eventStr)
		}
		return &topics[i], nil
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "topic %s", eventStr)
}

func (t *STopic) PreCheckPerformAction(
	ctx context.Context, userCred mcclient.TokenCredential,
	action string, query jsonutils.JSONObject, data jsonutils.JSONObject,
) error {
	if err := t.SStandaloneResourceBase.PreCheckPerformAction(ctx, userCred, action, query, data); err != nil {
		return err
	}
	if action == "enable" || action == "disable" {
		if !db.IsAdminAllowPerform(ctx, userCred, t, action) {
			return httperrors.NewForbiddenError("only allow admin to perform enable operations")
		}
	}
	return nil
}

func init() {
	converter = &sConverter{
		resource2Value: &sync.Map{},
		value2Resource: &sync.Map{},
		action2Value:   &sync.Map{},
		value2Action:   &sync.Map{},
	}
	converter.registerResource(
		map[string]int{
			notify.TOPIC_RESOURCE_SERVER:                   0,
			notify.TOPIC_RESOURCE_SCALINGGROUP:             1,
			notify.TOPIC_RESOURCE_SCALINGPOLICY:            2,
			notify.TOPIC_RESOURCE_IMAGE:                    3,
			notify.TOPIC_RESOURCE_DISK:                     4,
			notify.TOPIC_RESOURCE_SNAPSHOT:                 5,
			notify.TOPIC_RESOURCE_INSTANCESNAPSHOT:         6,
			notify.TOPIC_RESOURCE_SNAPSHOTPOLICY:           7,
			notify.TOPIC_RESOURCE_NETWORK:                  8,
			notify.TOPIC_RESOURCE_EIP:                      9,
			notify.TOPIC_RESOURCE_SECGROUP:                 10,
			notify.TOPIC_RESOURCE_LOADBALANCER:             11,
			notify.TOPIC_RESOURCE_LOADBALANCERACL:          12,
			notify.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE:  13,
			notify.TOPIC_RESOURCE_BUCKET:                   14,
			notify.TOPIC_RESOURCE_DBINSTANCE:               15,
			notify.TOPIC_RESOURCE_ELASTICCACHE:             16,
			notify.TOPIC_RESOURCE_SCHEDULEDTASK:            17,
			notify.TOPIC_RESOURCE_BAREMETAL:                18,
			notify.TOPIC_RESOURCE_VPC:                      19,
			notify.TOPIC_RESOURCE_DNSZONE:                  20,
			notify.TOPIC_RESOURCE_NATGATEWAY:               21,
			notify.TOPIC_RESOURCE_WEBAPP:                   22,
			notify.TOPIC_RESOURCE_CDNDOMAIN:                23,
			notify.TOPIC_RESOURCE_FILESYSTEM:               24,
			notify.TOPIC_RESOURCE_WAF:                      25,
			notify.TOPIC_RESOURCE_KAFKA:                    26,
			notify.TOPIC_RESOURCE_ELASTICSEARCH:            27,
			notify.TOPIC_RESOURCE_MONGODB:                  28,
			notify.TOPIC_RESOURCE_DNSRECORDSET:             29,
			notify.TOPIC_RESOURCE_LOADBALANCERLISTENER:     30,
			notify.TOPIC_RESOURCE_LOADBALANCERBACKEDNGROUP: 31,
			notify.TOPIC_RESOURCE_HOST:                     32,
			notify.TOPIC_RESOURCE_TASK:                     33,
			notify.TOPIC_RESOURCE_CLOUDPODS_COMPONENT:      34,
			notify.TOPIC_RESOURCE_DB_TABLE_RECORD:          35,
			notify.TOPIC_RESOURCE_USER:                     36,
			notify.TOPIC_RESOURCE_ACTION_LOG:               37,
			notify.TOPIC_RESOURCE_ACCOUNT_STATUS:           38,
			notify.TOPIC_RESOURCE_NET:                      39,
			notify.TOPIC_RESOURCE_SERVICE:                  40,
			notify.TOPIC_RESOURCE_VM_INTEGRITY_CHECK:       41,
			notify.TOPIC_RESOURCE_PROJECT:                  42,
		},
	)
	converter.registerAction(
		map[notify.SAction]int{
			notify.ActionCreate:               0,
			notify.ActionDelete:               1,
			notify.ActionPendingDelete:        2,
			notify.ActionUpdate:               3,
			notify.ActionRebuildRoot:          4,
			notify.ActionResetPassword:        5,
			notify.ActionChangeConfig:         6,
			notify.ActionExpiredRelease:       7,
			notify.ActionExecute:              8,
			notify.ActionChangeIpaddr:         9,
			notify.ActionSyncStatus:           10,
			notify.ActionCleanData:            11,
			notify.ActionMigrate:              12,
			notify.ActionCreateBackupServer:   13,
			notify.ActionDelBackupServer:      14,
			notify.ActionSyncCreate:           15,
			notify.ActionSyncUpdate:           16,
			notify.ActionSyncDelete:           17,
			notify.ActionOffline:              18,
			notify.ActionSystemPanic:          19,
			notify.ActionSystemException:      20,
			notify.ActionChecksumTest:         21,
			notify.ActionLock:                 22,
			notify.ActionExceedCount:          23,
			notify.ActionSyncAccountStatus:    24,
			notify.ActionPasswordExpireSoon:   25,
			notify.ActionNetOutOfSync:         26,
			notify.ActionMysqlOutOfSync:       27,
			notify.ActionServiceAbnormal:      28,
			notify.ActionServerPanicked:       29,
			notify.ActionAttach:               30,
			notify.ActionDetach:               31,
			notify.ActionIsolatedDeviceCreate: 32,
			notify.ActionIsolatedDeviceUpdate: 33,
			notify.ActionIsolatedDeviceDelete: 34,
			notify.ActionStatusChanged:        35,
		},
	)
}

var converter *sConverter

type sConverter struct {
	resource2Value *sync.Map
	value2Resource *sync.Map
	action2Value   *sync.Map
	value2Action   *sync.Map
}

type sResourceValue struct {
	resource string
	value    int
}

type sActionValue struct {
	action string
	value  int
}

func (rc *sConverter) registerResource(resourceValues map[string]int) {
	for resource, value := range resourceValues {
		if v, ok := rc.resource2Value.Load(resource); ok && v.(int) != value {
			log.Fatalf("resource '%s' has been mapped to value '%d', and it is not allowed to map to another value '%d'", resource, v, value)
		}
		if r, ok := rc.value2Resource.Load(value); ok && r.(string) != resource {
			log.Fatalf("value '%d' has been mapped to resource '%s', and it is not allowed to map to another resource '%s'", value, r, resource)
		}
		rc.resource2Value.Store(resource, value)
		rc.value2Resource.Store(value, resource)
	}
}

func (rc *sConverter) registerAction(actionValues map[notify.SAction]int) {
	for action, value := range actionValues {
		if v, ok := rc.action2Value.Load(action); ok && v.(int) != value {
			log.Fatalf("action '%s' has been mapped to value '%d', and it is not allowed to map to another value '%d'", action, v, value)
		}
		if a, ok := rc.value2Action.Load(value); ok && a.(notify.SAction) != action {
			log.Fatalf("value '%d' has been mapped to action '%s', and it is not allowed to map to another action '%s'", value, a, action)
		}
		rc.action2Value.Store(action, value)
		rc.value2Action.Store(value, action)
	}
}

func (rc *sConverter) resourceValue(resource string) int {
	v, ok := rc.resource2Value.Load(resource)
	if !ok {
		return -1
	}
	return v.(int)
}

func (rc *sConverter) resource(resourceValue int) string {
	r, ok := rc.value2Resource.Load(resourceValue)
	if !ok {
		return ""
	}
	return r.(string)
}

func (rc *sConverter) actionValue(action notify.SAction) int {
	v, ok := rc.action2Value.Load(action)
	if !ok {
		return -1
	}
	return v.(int)
}

func (rc *sConverter) action(actionValue int) notify.SAction {
	a, ok := rc.value2Action.Load(actionValue)
	if !ok {
		return notify.SAction("")
	}
	return a.(notify.SAction)
}

func (self *STopic) GetEnabledSubscribers(domainId, projectId string) ([]SSubscriber, error) {
	q := SubscriberManager.Query().Equals("topic_id", self.Id).IsTrue("enabled")
	q = q.Filter(sqlchemy.OR(
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("resource_scope"), api.SUBSCRIBER_SCOPE_PROJECT),
			sqlchemy.Equals(q.Field("resource_attribution_id"), projectId),
		),
		sqlchemy.AND(
			sqlchemy.Equals(q.Field("resource_scope"), api.SUBSCRIBER_SCOPE_DOMAIN),
			sqlchemy.Equals(q.Field("resource_attribution_id"), domainId),
		),
		sqlchemy.Equals(q.Field("resource_scope"), api.SUBSCRIBER_SCOPE_SYSTEM),
	))
	ret := []SSubscriber{}
	err := db.FetchModelObjects(SubscriberManager, q, &ret)
	return ret, err
}
