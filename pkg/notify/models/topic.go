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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/notify"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/bitmap"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func parseEvent(es string) (notify.SNotifyEvent, error) {
	es = strings.ToLower(es)
	ess := strings.Split(es, notify.DelimiterInEvent)
	if len(ess) != 2 && len(ess) != 3 {
		return notify.SNotifyEvent{}, fmt.Errorf("invalid event string %q", es)
	}
	event := notify.Event.WithResourceType(ess[0]).WithAction(notify.SAction(ess[1]))
	if len(ess) == 3 {
		event = event.WithResult(notify.SResult(ess[2]))
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
	Resources   uint64               `nullable:"false"`
	Actions     uint32               `nullable:"false"`
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
		t := new(STopic)
		t.Name = name
		t.Enabled = tristate.True
		switch name {
		case DefaultResourceCreateDelete:
			t.addResources(
				notify.TOPIC_RESOURCE_HOST,
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_SCALINGGROUP,
				notify.TOPIC_RESOURCE_IMAGE,
				notify.TOPIC_RESOURCE_DISK,
				notify.TOPIC_RESOURCE_SNAPSHOT,
				notify.TOPIC_RESOURCE_INSTANCESNAPSHOT,
				notify.TOPIC_RESOURCE_SNAPSHOTPOLICY,
				notify.TOPIC_RESOURCE_NETWORK,
				notify.TOPIC_RESOURCE_EIP,
				notify.TOPIC_RESOURCE_LOADBALANCER,
				notify.TOPIC_RESOURCE_LOADBALANCERACL,
				notify.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE,
				notify.TOPIC_RESOURCE_BUCKET,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
				notify.TOPIC_RESOURCE_BAREMETAL,
				notify.TOPIC_RESOURCE_SECGROUP,
				notify.TOPIC_RESOURCE_FILESYSTEM,
				notify.TOPIC_RESOURCE_NATGATEWAY,
				notify.TOPIC_RESOURCE_VPC,
				notify.TOPIC_RESOURCE_CDNDOMAIN,
				notify.TOPIC_RESOURCE_WAF,
				notify.TOPIC_RESOURCE_KAFKA,
				notify.TOPIC_RESOURCE_ELASTICSEARCH,
				notify.TOPIC_RESOURCE_MONGODB,
				notify.TOPIC_RESOURCE_DNSZONE,
				notify.TOPIC_RESOURCE_DNSRECORDSET,
				notify.TOPIC_RESOURCE_LOADBALANCERLISTENER,
				notify.TOPIC_RESOURCE_LOADBALANCERBACKEDNGROUP,
			)
			t.addAction(
				notify.ActionCreate,
				notify.ActionDelete,
				notify.ActionPendingDelete,
			)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
			t.ContentCn = api.COMMON_CONTENT_CN
			t.ContentEn = api.COMMON_CONTENT_EN
			t.TitleCn = api.COMMON_TITLE_CN
			t.TitleEn = api.COMMON_TITLE_EN
		case DefaultResourceChangeConfig:
			t.addResources(
				notify.TOPIC_RESOURCE_HOST,
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
			)
			t.addAction(notify.ActionChangeConfig)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
			t.ContentCn = api.COMMON_CONTENT_CN
			t.ContentEn = api.COMMON_CONTENT_EN
			t.TitleCn = api.COMMON_TITLE_CN
			t.TitleEn = api.COMMON_TITLE_EN
		case DefaultResourceUpdate:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
				notify.TOPIC_RESOURCE_USER,
				notify.TOPIC_RESOURCE_HOST,
			)
			t.addAction(notify.ActionUpdate)
			t.addAction(notify.ActionRebuildRoot)
			t.addAction(notify.ActionResetPassword)
			t.addAction(notify.ActionChangeIpaddr)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
			t.ContentCn = api.UPDATE_CONTENT_CN
			t.ContentEn = api.UPDATE_CONTENT_EN
			t.TitleCn = api.UPDATE_TITLE_CN
			t.TitleEn = api.UPDATE_TITLE_EN
		case DefaultScheduledTaskExecute:
			t.addResources(notify.TOPIC_RESOURCE_SCHEDULEDTASK)
			t.addAction(notify.ActionExecute)
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.SCHEDULEDTASK_EXECUTE_CONTENT_CN
			t.ContentEn = api.SCHEDULEDTASK_EXECUTE_CONTENT_EN
			t.TitleCn = api.SCHEDULEDTASK_EXECUTE_TITLE_CN
			t.TitleEn = api.SCHEDULEDTASK_EXECUTE_TITLE_EN
		case DefaultScalingPolicyExecute:
			t.addResources(notify.TOPIC_RESOURCE_SCALINGPOLICY)
			t.addAction(notify.ActionExecute)
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.SCALINGPOLICY_EXECUTE_CONTENT_CN
			t.ContentEn = api.SCALINGPOLICY_EXECUTE_CONTENT_EN
			t.TitleCn = api.SCALINGPOLICY_EXECUTE_TITLE_CN
			t.TitleEn = api.SCALINGPOLICY_EXECUTE_TITLE_EN
		case DefaultSnapshotPolicyExecute:
			t.addResources(notify.TOPIC_RESOURCE_SNAPSHOTPOLICY)
			t.addAction(notify.ActionExecute)
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.SNAPSHOTPOLICY_EXECUTE_CONTENT_CN
			t.ContentEn = api.SNAPSHOTPOLICY_EXECUTE_CONTENT_EN
			t.TitleCn = api.SNAPSHOTPOLICY_EXECUTE_TITLE_CN
			t.TitleEn = api.SNAPSHOTPOLICY_EXECUTE_TITLE_EN
		case DefaultResourceOperationFailed:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_EIP,
				notify.TOPIC_RESOURCE_LOADBALANCER,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
			)
			t.addAction(
				notify.ActionCreate,
				notify.ActionSyncStatus,
				notify.ActionRebuildRoot,
				notify.ActionChangeConfig,
				notify.ActionCreateBackupServer,
				notify.ActionDelBackupServer,
				notify.ActionMigrate,
			)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.Results = tristate.False
		case DefaultResourceOperationSuccessed:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
			)
			t.addAction(
				notify.ActionCreateBackupServer,
			)
			t.Type = notify.TOPIC_TYPE_RESOURCE
		case DefaultResourceSync:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_DISK,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
				notify.TOPIC_RESOURCE_LOADBALANCER,
				notify.TOPIC_RESOURCE_EIP,
				notify.TOPIC_RESOURCE_VPC,
				notify.TOPIC_RESOURCE_NETWORK,
				notify.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE,
				notify.TOPIC_RESOURCE_DNSZONE,
				notify.TOPIC_RESOURCE_NATGATEWAY,
				notify.TOPIC_RESOURCE_BUCKET,
				notify.TOPIC_RESOURCE_FILESYSTEM,
				notify.TOPIC_RESOURCE_WEBAPP,
				notify.TOPIC_RESOURCE_CDNDOMAIN,
				notify.TOPIC_RESOURCE_WAF,
				notify.TOPIC_RESOURCE_KAFKA,
				notify.TOPIC_RESOURCE_ELASTICSEARCH,
				notify.TOPIC_RESOURCE_MONGODB,
				notify.TOPIC_RESOURCE_DNSRECORDSET,
				notify.TOPIC_RESOURCE_LOADBALANCERLISTENER,
				notify.TOPIC_RESOURCE_LOADBALANCERBACKEDNGROUP,
			)
			t.addAction(
				notify.ActionSyncCreate,
				notify.ActionSyncUpdate,
				notify.ActionSyncDelete,
			)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.WebconsoleDisable = tristate.True
			t.Results = tristate.True
			t.ContentCn = api.COMMON_CONTENT_CN
			t.ContentEn = api.COMMON_CONTENT_EN
			t.TitleCn = api.COMMON_TITLE_CN
			t.TitleEn = api.COMMON_TITLE_EN
			groupKeys := []string{"action_display"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultSystemExceptionEvent:
			t.addResources(
				notify.TOPIC_RESOURCE_HOST,
				notify.TOPIC_RESOURCE_TASK,
			)
			t.addAction(
				notify.ActionSystemPanic,
				notify.ActionSystemException,
				notify.ActionOffline,
			)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.Results = tristate.False
			t.ContentCn = api.EXCEPTION_CONTENT_CN
			t.ContentEn = api.EXCEPTION_CONTENT_EN
			t.TitleCn = api.EXCEPTION_TITLE_CN
			t.TitleEn = api.EXCEPTION_TITLE_EN
		case DefaultChecksumTestFailed:
			t.addResources(
				notify.TOPIC_RESOURCE_DB_TABLE_RECORD,
				notify.TOPIC_RESOURCE_VM_INTEGRITY_CHECK,
				notify.TOPIC_RESOURCE_CLOUDPODS_COMPONENT,
				notify.TOPIC_RESOURCE_SNAPSHOT,
				notify.TOPIC_RESOURCE_IMAGE,
			)
			t.addAction(
				notify.ActionChecksumTest,
			)
			t.Type = notify.TOPIC_TYPE_SECURITY
			t.Results = tristate.False
			t.ContentCn = api.CHECKSUM_TEST_FAILED_CONTENT_CN
			t.ContentEn = api.CHECKSUM_TEST_FAILED_CONTENT_EN
			t.TitleCn = api.CHECKSUM_TEST_FAILED_TITLE_CN
			t.TitleEn = api.CHECKSUM_TEST_FAILED_TITLE_EN
		case DefaultUserLock:
			t.addResources(
				notify.TOPIC_RESOURCE_USER,
			)
			t.addAction(
				notify.ActionLock,
			)
			t.Type = notify.TOPIC_TYPE_SECURITY
			t.Results = tristate.True
			t.ContentCn = api.USER_LOCK_CONTENT_CN
			t.ContentEn = api.USER_LOCK_CONTENT_EN
			t.TitleCn = api.USER_LOCK_TITLE_CN
			t.TitleEn = api.USER_LOCK_TITLE_EN
		case DefaultActionLogExceedCount:
			t.addResources(
				notify.TOPIC_RESOURCE_ACTION_LOG,
			)
			t.addAction(
				notify.ActionExceedCount,
			)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
			t.ContentCn = api.ACTION_LOG_EXCEED_COUNT_CONTENT_CN
			t.ContentEn = api.ACTION_LOG_EXCEED_COUNT_CONTENT_EN
			t.TitleCn = api.ACTION_LOG_EXCEED_COUNT_TITLE_CN
			t.TitleEn = api.ACTION_LOG_EXCEED_COUNT_TITLE_EN
		case DefaultSyncAccountStatus:
			t.addResources(
				notify.TOPIC_RESOURCE_ACCOUNT_STATUS,
			)
			t.addAction(
				notify.ActionSyncAccountStatus,
			)
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.SYNC_ACCOUNT_STATUS_CONTENT_CN
			t.ContentEn = api.SYNC_ACCOUNT_STATUS_CONTENT_EN
			t.TitleCn = api.SYNC_ACCOUNT_STATUS_TITLE_CN
			t.TitleEn = api.SYNC_ACCOUNT_STATUS_TITLE_EN
			groupKeys := []string{"name"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultNetOutOfSync:
			t.addResources(
				notify.TOPIC_RESOURCE_NET,
			)
			t.addAction(
				notify.ActionNetOutOfSync,
			)
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.NET_OUT_OF_SYNC_CONTENT_CN
			t.ContentEn = api.NET_OUT_OF_SYNC_CONTENT_EN
			t.TitleCn = api.NET_OUT_OF_SYNC_TITLE_CN
			t.TitleEn = api.NET_OUT_OF_SYNC_TITLE_EN
			groupKeys := []string{"service_name"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultMysqlOutOfSync:
			t.addResources(
				notify.TOPIC_RESOURCE_DBINSTANCE,
			)
			t.addAction(
				notify.ActionMysqlOutOfSync,
			)
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
			t.Results = tristate.True
			t.ContentCn = api.MYSQL_OUT_OF_SYNC_CONTENT_CN
			t.ContentEn = api.MYSQL_OUT_OF_SYNC_CONTENT_EN
			t.TitleCn = api.MYSQL_OUT_OF_SYNC_TITLE_CN
			t.TitleEn = api.MYSQL_OUT_OF_SYNC_TITLE_EN
			groupKeys := []string{"ip"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultServiceAbnormal:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVICE,
			)
			t.addAction(
				notify.ActionServiceAbnormal,
			)
			t.Results = tristate.True
			t.Type = notify.TOPIC_TYPE_AUTOMATED_PROCESS
			t.ContentCn = api.SERVICE_ABNORMAL_CONTENT_CN
			t.ContentEn = api.SERVICE_ABNORMAL_CONTENT_EN
			t.TitleCn = api.SERVICE_ABNORMAL_TITLE_CN
			t.TitleEn = api.SERVICE_ABNORMAL_TITLE_EN
			groupKeys := []string{"service_name"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultServerPanicked:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
			)
			t.addAction(
				notify.ActionServerPanicked,
			)
			t.Results = tristate.False
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.ContentCn = api.SERVER_PANICKED_CONTENT_CN
			t.ContentEn = api.SERVER_PANICKED_CONTENT_EN
			t.TitleCn = api.SERVER_PANICKED_TITLE_CN
			t.TitleEn = api.SERVER_PANICKED_TITLE_EN
			groupKeys := []string{"name"}
			t.GroupKeys = (*api.STopicGroupKeys)(&groupKeys)
		case DefaultPasswordExpire:
			t.addResources(
				notify.TOPIC_RESOURCE_USER,
			)
			t.addAction(
				notify.ActionPasswordExpireSoon,
			)
			t.AdvanceDays = []int{1, 7}
			t.Type = notify.TOPIC_TYPE_SECURITY
			t.Results = tristate.True
			t.ContentCn = api.PWD_EXPIRE_SOON_CONTENT_CN
			t.ContentEn = api.PWD_EXPIRE_SOON_CONTENT_EN
			t.TitleCn = api.PWD_EXPIRE_SOON_TITLE_CN
			t.TitleEn = api.PWD_EXPIRE_SOON_TITLE_EN
		case DefaultResourceRelease:
			t.addResources(
				notify.TOPIC_RESOURCE_SERVER,
				notify.TOPIC_RESOURCE_DISK,
				notify.TOPIC_RESOURCE_EIP,
				notify.TOPIC_RESOURCE_LOADBALANCER,
				notify.TOPIC_RESOURCE_DBINSTANCE,
				notify.TOPIC_RESOURCE_ELASTICCACHE,
			)
			t.addAction(notify.ActionExpiredRelease)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.AdvanceDays = []int{1, 7, 30}
			t.Results = tristate.True
			t.ContentCn = api.EXPIRED_RELEASE_CONTENT_CN
			t.ContentEn = api.EXPIRED_RELEASE_CONTENT_EN
			t.TitleCn = api.EXPIRED_RELEASE_TITLE_CN
			t.TitleEn = api.EXPIRED_RELEASE_TITLE_EN
		case DefaultAttachOrDetach:
			t.addResources(
				notify.TOPIC_RESOURCE_HOST,
			)
			t.addAction(
				notify.ActionAttach,
				notify.ActionDetach,
			)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
		case DefaultIsolatedDeviceChanged:
			t.addResources(
				notify.TOPIC_RESOURCE_HOST,
			)
			t.addAction(
				notify.ActionIsolatedDeviceCreate,
				notify.ActionIsolatedDeviceUpdate,
				notify.ActionIsolatedDeviceDelete,
			)
			t.Type = notify.TOPIC_TYPE_RESOURCE
			t.Results = tristate.True
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
				topic.Resources = t.Resources
				topic.Actions = t.Actions
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
	}
	return nil
}

func (sm *STopicManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input notify.TopicListInput) (*sqlchemy.SQuery, error) {
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

func (sm *STopicManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []notify.TopicDetails {
	sRows := sm.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]notify.TopicDetails, len(objs))
	for i := range rows {
		rows[i].StandaloneResourceDetails = sRows[i]
		ss := objs[i].(*STopic)
		rows[i].Resources = ss.getResources()
	}
	return rows
}

type SSubscriberDis struct {
	SSubscriber
	ReceiverName string `json:"receiver_name"`
	RoleName     string `json:"role_name"`
}

func (s *STopic) subscriptionReceiverDiss() ([]SSubscriberDis, error) {
	q := SubscriberManager.Query().Equals("subscription_id", s.Id)
	rq := ReceiverManager.Query("id", "name").SubQuery()
	roq := db.RoleCacheManager.Query("id", "name").SubQuery()
	q = q.LeftJoin(rq, sqlchemy.Equals(q.Field("receiver"), rq.Field("id")))
	q = q.LeftJoin(roq, sqlchemy.Equals(q.Field("receiver"), roq.Field("id")))
	// It looks strange, but the order of append cannot be changed
	q.AppendField(q.QueryFields()...)
	q.AppendField(rq.Field("name", "receiver_name"))
	q.AppendField(roq.Field("name", "role_name"))
	srs := make([]SSubscriberDis, 0)
	err := q.All(&srs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch All")
	}
	return srs, nil
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
	for _, resource := range resources {
		v := converter.resourceValue(resource)
		if v < 0 {
			continue
		}
		s.Resources += 1 << v
	}
}

func (s *STopic) addAction(actions ...notify.SAction) {
	for _, action := range actions {
		v := converter.actionValue(action)
		if v < 0 {
			continue
		}
		s.Actions += 1 << v
	}
}

func (s *STopic) getResources() []string {
	vs := bitmap.Uint64ToIntArray(s.Resources)
	resources := make([]string, 0, len(vs))
	for _, v := range vs {
		resources = append(resources, converter.resource(v))
	}
	return resources
}

func (s *STopic) getActions() []notify.SAction {
	vs := bitmap.Uint2IntArray(s.Actions)
	actions := make([]notify.SAction, 0, len(vs))
	for _, v := range vs {
		actions = append(actions, converter.action(v))
	}
	return actions
}

func (sm *STopicManager) GetTopicByEvent(resourceType string, action notify.SAction, isFailed notify.SResult) (*STopic, error) {
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

func (sm *STopicManager) GetTopicsByEvent(resourceType string, action notify.SAction, isFailed notify.SResult) ([]STopic, error) {
	resourceV := converter.resourceValue(resourceType)
	if resourceV < 0 {
		return nil, fmt.Errorf("unknow resource type %s", resourceType)
	}
	actionV := converter.actionValue(action)
	if actionV < 0 {
		return nil, fmt.Errorf("unkonwn action %s", action)
	}
	q := sm.Query()
	if isFailed == api.ResultSucceed {
		q = q.Equals("results", true)
	} else {
		q = q.Equals("results", false)
	}
	q = q.Equals("enabled", true)
	q = q.Filter(sqlchemy.GT(sqlchemy.AND_Val("", q.Field("resources"), 1<<resourceV), 0))
	q = q.Filter(sqlchemy.GT(sqlchemy.AND_Val("", q.Field("actions"), 1<<actionV), 0))
	topics := []STopic{}
	err := db.FetchModelObjects(sm, q, &topics)
	return topics, err
}

func (sm *STopicManager) TopicsByEvent(eventStr string) ([]STopic, error) {
	event, err := parseEvent(eventStr)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse event %q", event)
	}
	resourceV := converter.resourceValue(event.ResourceType())
	if resourceV < 0 {
		log.Warningf("unknown resource type: %s", event.ResourceType())
		return nil, nil
	}
	actionV := converter.actionValue(event.Action())
	if actionV < 0 {
		log.Warningf("unknown action type: %s", event.Action())
		return nil, nil
	}
	q := sm.Query()
	if event.Result() == api.ResultSucceed {
		q = q.Equals("results", true)
	} else {
		q = q.Equals("results", false)
	}
	q = q.Equals("enabled", true)
	q = q.Filter(sqlchemy.GT(sqlchemy.AND_Val("", q.Field("resources"), 1<<resourceV), 0))
	q = q.Filter(sqlchemy.GT(sqlchemy.AND_Val("", q.Field("actions"), 1<<actionV), 0))
	var topics []STopic
	err = db.FetchModelObjects(sm, q, &topics)
	if err != nil {
		return nil, errors.Wrap(err, "unable to FetchModelObjects")
	}
	return topics, nil
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
		},
	)
	converter.registerAction(
		map[notify.SAction]int{
			notify.ActionCreate:             0,
			notify.ActionDelete:             1,
			notify.ActionPendingDelete:      2,
			notify.ActionUpdate:             3,
			notify.ActionRebuildRoot:        4,
			notify.ActionResetPassword:      5,
			notify.ActionChangeConfig:       6,
			notify.ActionExpiredRelease:     7,
			notify.ActionExecute:            8,
			notify.ActionChangeIpaddr:       9,
			notify.ActionSyncStatus:         10,
			notify.ActionCleanData:          11,
			notify.ActionMigrate:            12,
			notify.ActionCreateBackupServer: 13,
			notify.ActionDelBackupServer:    14,
			notify.ActionSyncCreate:         15,
			notify.ActionSyncUpdate:         16,
			notify.ActionSyncDelete:         17,
			notify.ActionOffline:            18,
			notify.ActionSystemPanic:        19,
			notify.ActionSystemException:    20,
			notify.ActionChecksumTest:       21,
			notify.ActionLock:               22,
			notify.ActionExceedCount:        23,
			notify.ActionSyncAccountStatus:  24,
			notify.ActionPasswordExpireSoon: 25,
			notify.ActionNetOutOfSync:       26,
			notify.ActionMysqlOutOfSync:     27,
			notify.ActionServiceAbnormal:    28,
			notify.ActionServerPanicked:     29,
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

func (self *STopic) CreateEvent(ctx context.Context, resType, action, message string) (*SEvent, error) {
	eve := &SEvent{
		Message:      message,
		ResourceType: resType,
		Action:       action,
		TopicId:      self.Id,
	}
	return eve, EventManager.TableSpec().Insert(ctx, eve)
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

func (sm *STopicManager) TopicByEvent(eventStr string) (*STopic, error) {
	topics, err := sm.TopicsByEvent(eventStr)
	if err != nil {
		return nil, err
	}
	if len(topics) == 1 {
		return &topics[0], nil
	}
	if len(topics) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "eventStr:%s", eventStr)
	}
	return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, "eventStr:%s", eventStr)
}
