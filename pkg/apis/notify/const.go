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

package notify

import "yunion.io/x/onecloud/pkg/apis"

const (
	SERVICE_TYPE    = apis.SERVICE_TYPE_NOTIFY
	SERVICE_VERSION = ""

	EMAIL          = "email"
	MOBILE         = "mobile"
	DINGTALK       = "dingtalk"
	FEISHU         = "feishu"
	WEBCONSOLE     = "webconsole"
	WORKWX         = "workwx"
	FEISHU_ROBOT   = "feishu-robot"
	DINGTALK_ROBOT = "dingtalk-robot"
	WORKWX_ROBOT   = "workwx-robot"
	WEBHOOK        = "webhook"

	ROBOT = "robot"

	RECEIVER_NOTIFICATION_RECEIVED = "received"  // Received a task about sending a notification
	RECEIVER_NOTIFICATION_SENT     = "sending"   // Nofity module has sent notification, but result unkown
	RECEIVER_NOTIFICATION_OK       = "sent_ok"   // Notification was sent successfully
	RECEIVER_NOTIFICATION_FAIL     = "sent_fail" // That sent a notification is failed

	VERIFICATION_SENT          = "sent"      // Verification was sent
	VERIFICATION_SENT_FAIL     = "sent_fail" // Verification was sent failed
	VERIFICATION_VERIFIED      = "verified"  // Verification was verified
	VERIFICATION_TOKEN_EXPIRED = "Verification code expired"
	VERIFICATION_TOKEN_INVALID = "Incorrect verification code"

	RECEIVER_STATUS_READY       = "ready"
	RECEIVER_STATUS_PULLING     = "pulling"
	RECEIVER_STATUS_PULL_FAILED = "pull_failed"

	NOTIFICATION_PRIORITY_IMPORTANT = "important"
	NOTIFICATION_PRIORITY_CRITICAL  = "fatal"
	NOTIFICATION_PRIORITY_NORMAL    = "normal"

	NOTIFICATION_STATUS_RECEIVED = "received"
	NOTIFICATION_STATUS_SENDING  = "sending"
	NOTIFICATION_STATUS_FAILED   = "failed"
	NOTIFICATION_STATUS_OK       = "ok"
	NOTIFICATION_STATUS_PART_OK  = "part_ok"

	NOTIFICATION_TAG_ALERT = "alert"

	TEMPLATE_TYPE_TITLE   = "title"
	TEMPLATE_TYPE_CONTENT = "content"
	TEMPLATE_TYPE_REMOTE  = "remote"

	TEMPLATE_LANG_EN = "en"
	TEMPLATE_LANG_CN = "cn"

	CTYPE_ROBOT_YES  = "yes"
	CTYPE_ROBOT_ONLY = "only"

	SUBSCRIPTION_RESOURCE_CREATE_DELETE      = "resource create delete"
	SUBSCRIPTION_RESOURCE_CHANGECONFIG       = "resource change config"
	SUBSCRIPTION_RESOURCE_UPDATE             = "resource udpate"
	SUBSCRIPTION_RESOURCE_EXPIRED_RELEASE    = "resource expired release"
	SUBSCRIPTION_AUTOMATED_PROCESS_EXECUTION = "automated process execution"

	SUBSCRIPTION_TYPE_RESOURCE          = "resource"
	SUBSCRIPTION_TYPE_AUTOMATED_PROCESS = "automated_process"

	SUBSCRIPTION_RESOURCE_SERVER                  = "server"
	SUBSCRIPTION_RESOURCE_SCALINGGROUP            = "scalinggroup"
	SUBSCRIPTION_RESOURCE_SCALINGPOLICY           = "scalingpolicy"
	SUBSCRIPTION_RESOURCE_IMAGE                   = "image"
	SUBSCRIPTION_RESOURCE_DISK                    = "disk"
	SUBSCRIPTION_RESOURCE_SNAPSHOT                = "snapshot"
	SUBSCRIPTION_RESOURCE_INSTANCESNAPSHOT        = "instance_snapshot"
	SUBSCRIPTION_RESOURCE_SNAPSHOTPOLICY          = "snapshotpolicy"
	SUBSCRIPTION_RESOURCE_NETWORK                 = "network"
	SUBSCRIPTION_RESOURCE_EIP                     = "eip"
	SUBSCRIPTION_RESOURCE_SECGROUP                = "secgroup"
	SUBSCRIPTION_RESOURCE_LOADBALANCER            = "loadbalancer"
	SUBSCRIPTION_RESOURCE_LOADBALANCERACL         = "loadbalanceracl"
	SUBSCRIPTION_RESOURCE_LOADBALANCERCERTIFICATE = "loadbalancercertificate"
	SUBSCRIPTION_RESOURCE_BUCKET                  = "bucket"
	SUBSCRIPTION_RESOURCE_DBINSTANCE              = "dbinstance"
	SUBSCRIPTION_RESOURCE_ELASTICCACHE            = "elasticcache"
	SUBSCRIPTION_RESOURCE_SCHEDULEDTASK           = "scheduledtask"
)
