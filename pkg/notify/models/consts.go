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

	NOTIFY_RECEIVED = "received"  // Received a task about sending a notification
	NOTIFY_SENT     = "sent"      // Nofity module has sent notification, but result unkown
	NOTIFY_OK       = "sent_ok"   // Notification was sent successfully
	NOTIFY_FAIL     = "sent_fail" // That sent a notification is failed

	CONTACT_INIT      = "init"      // Contact's status is init which means no verifying
	CONTACT_VERIFYING = "verifying" // Contact's status is verifying
	CONTACT_VERIFIED  = "verified"  // Contact's status is verified

	VERIFICATION_SENT          = "sent"      // Verification was sent
	VERIFICATION_SENT_FAIL     = "sent_fail" // Verification was sent failed
	VERIFICATION_VERIFIED      = "verified"  // Verification was verified
	VERIFICATION_TOKEN_EXPIRED = "Verification code expired"
	VERIFICATION_TOKEN_INVALID = "Incorrect verification code"
)

// Dingtalk account will be update automatically as mobile number change so that update dingtalk is not allowed
// In webconsole, uid is the same as contact.
var UpdateNotAllow = map[string]struct{}{
	DINGTALK:   {},
	WEBCONSOLE: {},
}
