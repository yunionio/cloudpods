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

package tasks

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

var PullContactType = []string{
	apis.DINGTALK,
	apis.FEISHU,
	apis.WORKWX,
}

var UserContactType = []string{
	apis.EMAIL,
	apis.MOBILE,
}

var allContactTypes = []string{
	apis.DINGTALK,
	apis.FEISHU,
	apis.WORKWX,
	apis.EMAIL,
	apis.MOBILE,
}

type SubcontactPullTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SubcontactPullTask{})
}

func (self *SubcontactPullTask) taskFailed(ctx context.Context, receiver *models.SReceiver, reason string) {
	log.Errorf("fail to pull subcontact of receiver %q: %s", receiver.Id, reason)
	receiver.SetStatus(ctx, self.UserCred, apis.RECEIVER_STATUS_PULL_FAILED, reason)
	logclient.AddActionLogWithContext(ctx, receiver, logclient.ACT_PULL_SUBCONTACT, reason, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

func (self *SubcontactPullTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	failedReasons := make([]string, 0)
	// pull contacts
	receiver := obj.(*models.SReceiver)
	if len(receiver.Mobile) == 0 {
		self.SetStageComplete(ctx, nil)
		return
	}
	// sync email and mobile to keystone
	s := auth.GetSession(ctx, self.UserCred, "")
	mobile := receiver.Mobile
	if strings.HasPrefix(mobile, "+86 ") {
		mobile = strings.TrimSpace(mobile[4:])
	}
	params := map[string]string{
		"email":  receiver.Email,
		"mobile": receiver.Mobile,
	}
	_, err := identity.UsersV3.Update(s, receiver.Id, jsonutils.Marshal(params))
	if err != nil {
		log.Errorf("update user email and mobile fail %s", err)
	}
	var contactTypes []string
	if self.Params.Contains("contact_types") {
		jArray, _ := self.Params.Get("contact_types")
		contactTypes = jArray.(*jsonutils.JSONArray).GetStringArray()
	}

	// 遍历所有通知渠道
	for _, contactType := range allContactTypes {
		// 若该渠道在输入渠道内，则设为enable
		if utils.IsInStringArray(contactType, contactTypes) {
			// 常规渠道
			if utils.IsInStringArray(contactType, PullContactType) {
				content := ""
				driver := models.GetDriver(contactType)
				content, err = driver.ContactByMobile(ctx, mobile, self.UserCred.GetDomainId())
				if err != nil {
					var reason string
					if errors.Cause(err) == apis.ErrNoSuchMobile {
						receiver.MarkContactTypeUnVerified(ctx, contactType, apis.ErrNoSuchMobile.Error())
						reason = fmt.Sprintf("%q: no such mobile %s", contactType, receiver.Mobile)
					} else if errors.Cause(err) == apis.ErrIncompleteConfig {
						receiver.MarkContactTypeUnVerified(ctx, contactType, apis.ErrIncompleteConfig.Error())
						reason = fmt.Sprintf("%q: %v", contactType, err)
					} else {
						receiver.MarkContactTypeUnVerified(ctx, contactType, "service exceptions")
						reason = fmt.Sprintf("%q: %v", contactType, err)
					}
					failedReasons = append(failedReasons, reason)
					continue
				}
				subcontact := []models.SSubContact{}
				q := models.SubContactManager.Query()
				cond := sqlchemy.AND(sqlchemy.Equals(q.Field("receiver_id"), receiver.Id), sqlchemy.Equals(q.Field("type"), contactType))
				q.Filter(cond)
				err = db.FetchModelObjects(models.SubContactManager, q, &subcontact)
				if err != nil {
					failedReasons = append(failedReasons, err.Error())
					continue
				}
				subid := ""
				if len(subcontact) > 0 {
					subid = subcontact[0].Id
				}
				err = models.SubContactManager.TableSpec().InsertOrUpdate(ctx, &models.SSubContact{
					SStandaloneResourceBase: db.SStandaloneResourceBase{
						SStandaloneAnonResourceBase: db.SStandaloneAnonResourceBase{Id: subid},
					},
					ReceiverID:        receiver.Id,
					Type:              contactType,
					Contact:           content,
					ParentContactType: "mobile",
					Enabled:           tristate.True,
				})
				if err != nil {
					failedReasons = append(failedReasons, err.Error())
					continue
				}
				receiver.SetContact(contactType, content)
				receiver.MarkContactTypeVerified(ctx, contactType)
			} else {
				_, err := db.Update(receiver, func() error {
					if contactType == apis.MOBILE {
						receiver.EnabledMobile = tristate.True
					}
					if contactType == apis.EMAIL {
						receiver.EnabledEmail = tristate.True
					}
					return nil
				})
				if err != nil {
					failedReasons = append(failedReasons, err.Error())
					continue
				}
			}
		} else {
			// 若该渠道在输入渠道内，则设为disable
			if utils.IsInStringArray(contactType, PullContactType) {
				subcontact := []models.SSubContact{}
				q := models.SubContactManager.Query()
				cond := sqlchemy.AND(sqlchemy.Equals(q.Field("receiver_id"), receiver.Id), sqlchemy.Equals(q.Field("type"), contactType))
				q.Filter(cond)
				err = db.FetchModelObjects(models.SubContactManager, q, &subcontact)
				if err != nil {
					failedReasons = append(failedReasons, err.Error())
					continue
				}
				subid := ""
				if len(subcontact) > 0 {
					subid = subcontact[0].Id
				}
				err = models.SubContactManager.TableSpec().InsertOrUpdate(ctx, &models.SSubContact{
					SStandaloneResourceBase: db.SStandaloneResourceBase{
						SStandaloneAnonResourceBase: db.SStandaloneAnonResourceBase{Id: subid},
					},
					ReceiverID:        receiver.Id,
					Type:              contactType,
					ParentContactType: "mobile",
					Enabled:           tristate.False,
				})
				if err != nil {
					failedReasons = append(failedReasons, err.Error())
					continue
				}
			} else {
				_, err := db.Update(receiver, func() error {
					if contactType == apis.MOBILE {
						receiver.EnabledMobile = tristate.False
					}
					if contactType == apis.EMAIL {
						receiver.EnabledEmail = tristate.False
					}
					return nil
				})
				if err != nil {
					failedReasons = append(failedReasons, err.Error())
					continue
				}
			}
		}
	}

	if len(failedReasons) > 0 {
		reason := strings.Join(failedReasons, "; ")
		self.taskFailed(ctx, receiver, reason)
		return
	}
	// success
	receiver.SetStatus(ctx, self.UserCred, apis.RECEIVER_STATUS_READY, "")
	logclient.AddActionLogWithContext(ctx, receiver, logclient.ACT_PULL_SUBCONTACT, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
