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

package oldmodels

import (
	"time"
)

type SContactManager struct {
	SStatusStandaloneResourceBaseManager
}

var ContactManager *SContactManager

func init() {
	ContactManager = &SContactManager{
		SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(
			SContact{},
			"notify_t_contacts",
			"contact",
			"contacts",
		),
	}
	ContactManager.SetVirtualObject(ContactManager)
}

const (
	CONTACT_INIT      = "init"      // Contact's status is init which means no verifying
	CONTACT_VERIFYING = "verifying" // Contact's status is verifying
	CONTACT_VERIFIED  = "verified"  // Contact's status is verified
)

type SContact struct {
	SStatusStandaloneResourceBase

	UID         string    `width:"128" nullable:"false" create:"required" update:"user" list:"user" get:"user"`
	ContactType string    `width:"16" nullable:"false" create:"required" update:"user"`
	Contact     string    `width:"64" nullable:"false" create:"required" update:"user"`
	Enabled     string    `width:"5" nullable:"false" default:"1" create:"optional" update:"user"`
	VerifiedAt  time.Time `update:"user"`
}
