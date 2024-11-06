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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

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
	GroupKeys   *api.STopicGroupKeys `nullable:"true" list:"user"  update:"user"`
	AdvanceDays []int                `nullable:"true" charset:"utf8" list:"user" update:"user" create:"optional"`

	WebconsoleDisable tristate.TriState
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
		if isNew {
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
	for i := range rows {
		rows[i].StandaloneResourceDetails = sRows[i]
		ss := objs[i].(*STopic)
		rows[i].Resources = ss.getResources()
	}
	return rows
}

func (sm *STopicManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewForbiddenError("prohibit creation")
}

func (ss *STopic) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.TopicUpdateInput) (api.TopicUpdateInput, error) {
	return input, nil
}

func (ss *STopic) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return httperrors.NewForbiddenError("prohibit deletion")
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

func (s *STopic) getResources() []string {
	temp := []SNotifyResource{}
	q := NotifyResourceManager.Query()
	sq := TopicResourceManager.Query().Equals("topic_id", s.Id).SubQuery()
	q = q.Join(sq, sqlchemy.Equals(q.Field("id"), sq.Field("resource_id")))
	q.All(&temp)
	resources := make([]string, 0, len(temp))
	for i := range temp {
		resources = append(resources, temp[i].Id)
	}
	return resources
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
	q = q.Equals("results", isFailed)
	q = q.Equals("enabled", true)
	actionQ := NotifyActionManager.Query().Equals("name", action).IsTrue("enabled").SubQuery()
	topicActionQ := TopicActionManager.Query()
	topicActionQ = topicActionQ.Join(actionQ, sqlchemy.Equals(topicActionQ.Field("action_id"), actionQ.Field("id")))
	resourceQ := NotifyResourceManager.Query().Equals("name", resourceType).IsTrue("enabled").SubQuery()
	topicResourceQ := TopicResourceManager.Query()
	topicResourceSq := topicResourceQ.Join(resourceQ, sqlchemy.Equals(topicResourceQ.Field("resource_id"), resourceQ.Field("id"))).SubQuery()
	tempSQ := topicActionQ.LeftJoin(topicResourceSq, sqlchemy.Equals(topicActionQ.Field("topic_id"), topicResourceSq.Field("topic_id"))).SubQuery()
	q = q.Join(tempSQ, sqlchemy.Equals(tempSQ.Field("topic_id"), q.Field("id")))
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
	q := manager.Query()
	if event.Result() == api.ResultSucceed {
		q = q.Equals("results", true)
	} else {
		q = q.Equals("results", false)
	}
	actionQ := NotifyActionManager.Query().Equals("name", event.Action()).IsTrue("enabled").SubQuery()
	topicActionQ := TopicActionManager.Query()
	topicActionQ = topicActionQ.Join(actionQ, sqlchemy.Equals(topicActionQ.Field("action_id"), actionQ.Field("id")))
	resourceQ := NotifyResourceManager.Query().Equals("name", event.ResourceType()).IsTrue("enabled").SubQuery()
	topicResourceQ := TopicResourceManager.Query()
	topicResourceSq := topicResourceQ.Join(resourceQ, sqlchemy.Equals(topicResourceQ.Field("resource_id"), resourceQ.Field("id"))).SubQuery()
	tempSQ := topicActionQ.LeftJoin(topicResourceSq, sqlchemy.Equals(topicActionQ.Field("topic_id"), topicResourceSq.Field("topic_id"))).SubQuery()
	q = q.Join(tempSQ, sqlchemy.Equals(tempSQ.Field("topic_id"), q.Field("id")))
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

func (topic *STopic) CreateEvent(ctx context.Context, resType, action, message string) (*SEvent, error) {
	eve := &SEvent{
		Message:      message,
		ResourceType: resType,
		Action:       action,
		TopicId:      topic.Id,
	}
	return eve, EventManager.TableSpec().Insert(ctx, eve)
}

func (s *STopic) PerformAddResource(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.STopicResourceInput) (jsonutils.JSONObject, error) {
	count, err := NotifyResourceManager.Query().Equals("id", input.ResourceId).CountWithError()
	if err != nil {
		return nil, errors.Wrap(err, "fetch count")
	}
	if count == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "resource_id")
	}
	count, err = TopicResourceManager.Query().Equals("resource_id", input.ResourceId).Equals("topic_id", s.Id).CountWithError()
	if err != nil {
		return nil, errors.Wrap(err, "fetch count")
	}
	if count > 0 {
		return nil, errors.Wrapf(httperrors.ErrDuplicateResource, "topic:%s,resource:%s has been exist", s.Id, input.ResourceId)
	}
	topicResource := STopicResource{
		TopicId:    s.Id,
		ResourceId: input.ResourceId,
	}
	return jsonutils.Marshal(topicResource), TopicResourceManager.TableSpec().InsertOrUpdate(ctx, &topicResource)
}

func (s *STopic) PerformRemoveResource(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.STopicResourceInput) (jsonutils.JSONObject, error) {
	topicResource := STopicResource{}
	err := TopicResourceManager.Query().Equals("resource_id", input.ResourceId).Equals("topic_id", s.Id).First(&topicResource)
	if err != nil {
		return nil, errors.Wrap(err, "fetch topic_resource")
	}
	return jsonutils.Marshal(topicResource), topicResource.Delete(ctx, userCred)
}

func (s *STopic) PerformAddAction(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.STopicActionInput) (jsonutils.JSONObject, error) {
	count, err := NotifyActionManager.Query().Equals("id", input.ActionId).CountWithError()
	if err != nil {
		return nil, errors.Wrap(err, "fetch count")
	}
	if count == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "action_id")
	}
	count, err = TopicActionManager.Query().Equals("action_id", input.ActionId).Equals("topic_id", s.Id).CountWithError()
	if err != nil {
		return nil, errors.Wrap(err, "fetch count")
	}
	if count > 0 {
		return nil, errors.Wrapf(httperrors.ErrDuplicateResource, "topic:%s,action:%s has been exist", s.Id, input.ActionId)
	}
	topicAction := STopicAction{
		TopicId:  s.Id,
		ActionId: input.ActionId,
	}
	return jsonutils.Marshal(topicAction), TopicActionManager.TableSpec().InsertOrUpdate(ctx, &topicAction)
}

func (s *STopic) PerformRemoveAction(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.STopicActionInput) (jsonutils.JSONObject, error) {
	topicAction := STopicAction{}
	err := TopicActionManager.Query().Equals("action_id", input.ActionId).Equals("topic_id", s.Id).First(&topicAction)
	if err != nil {
		return nil, errors.Wrap(err, "fetch topic_action")
	}
	return jsonutils.Marshal(topicAction), topicAction.Delete(ctx, userCred)
}
