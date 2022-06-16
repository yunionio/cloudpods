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
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SSubContactManager struct {
	db.SStandaloneResourceBaseManager
}

// +onecloud:swagger-gen-ignore
type SSubContact struct {
	db.SStandaloneResourceBase

	// id of receiver user
	ReceiverID        string            `width:"128" nullable:"false" index:"true"`
	Type              string            `width:"16" nullable:"false" index:"true"`
	Contact           string            `width:"128" nullable:"false"`
	ParentContactType string            `width:"16" nullable:"false"`
	Enabled           tristate.TriState `default:"false"`
	Verified          tristate.TriState `default:"false"`
	VerifiedNote      string            `width:"1024"`
}

var SubContactManager *SSubContactManager

var (
	vTrue  = true
	vFalse = false

	pTrue  = &vTrue
	pFalse = &vFalse
)

func init() {
	SubContactManager = &SSubContactManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SSubContact{},
			"subcontacts_tbl",
			"subcontact",
			"subcontacts",
		),
	}
	SubContactManager.SetVirtualObject(SubContactManager)
}

func (sc *SSubContact) Enable() error {
	return sc.Update(nil, pTrue, nil)
}

func (sc *SSubContact) Disable() error {
	return sc.Update(nil, pFalse, nil)
}

func (sc *SSubContact) Verify() error {
	return sc.Update(nil, nil, pTrue)
}

func (sc *SSubContact) Disverify() error {
	return sc.Update(nil, nil, pFalse)
}

func (sc *SSubContact) Update(contact *string, enabled *bool, verified *bool) error {
	_, err := db.Update(sc, func() error {
		if contact != nil {
			sc.Contact = *contact
		}
		if enabled != nil {
			sc.Enabled = tristate.NewFromBool(*enabled)
		}
		if verified != nil {
			sc.Verified = tristate.NewFromBool(*verified)
		}
		return nil
	})
	return err
}
