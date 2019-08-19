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

const (
	EMAIL      = "email"
	MOBILE     = "mobile"
	DINGTALK   = "dingtalk"
	WEBCONSOLE = "webconsole"
	// Received a task about sending a notification
	NOTIFY_RECEIVED = "received"
	// Nofity module hasn't sent the notification
	NOTIFY_UNSENT = "unsent"
	// Nofity module has sent notification, but result unkown
	NOTIFY_SENT = "sent"
	// Notification was sent successfully
	NOTIFY_OK = "sent_ok"
	// That sent a notification is failed
	NOTIFY_FAIL = "sent_fail"
	// Contact's status is init which means no verifying
	CONTACT_INIT = "init"
	// Contact's status is verifying
	CONTACT_VERIFYING = "verifying"
	// Contact's status is verified
	CONTACT_VERIFIED = "verified"
	// Verification was sent
	VERIFICATION_SENT = "sent"
	// Verification was verified
	VERIFICATION_VERIFIED = "verified"

	VERIFICATION_TOKEN_EXPIRED = "Verification code expired"

	VERIFICATION_TOKEN_INVALID = "Incorrect verification code"
)
