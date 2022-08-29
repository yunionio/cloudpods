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
	"math/rand"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/notify/options"
)

type SVerificationManager struct {
	db.SStandaloneResourceBaseManager
}

var VerificationManager *SVerificationManager

func init() {
	VerificationManager = &SVerificationManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SVerification{},
			"verification_tbl",
			"verification",
			"verifications",
		),
	}
	VerificationManager.SetVirtualObject(VerificationManager)
}

// +onecloud:swagger-gen-ignore
type SVerification struct {
	db.SStandaloneResourceBase

	ReceiverId  string `width:"128" nullable:"false"`
	ContactType string `width:"16" nullable:"false"`
	Token       string `width:"200" nullable:"false"`
}

var ErrVerifyFrequently = errors.Wrap(httperrors.ErrTooManyRequests, "Send validation messages too frequently")

func (vm *SVerificationManager) generateVerifyToken() string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	token := fmt.Sprintf("%06v", rnd.Int31n(1000000))
	return token
}

func (vm *SVerificationManager) Create(ctx context.Context, receiverId, contactType string) (*SVerification, error) {
	// try to reuse
	ret, err := vm.Get(receiverId, contactType)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, err
	}
	if ret == nil {
		ret = &SVerification{
			ReceiverId:  receiverId,
			ContactType: contactType,
			Token:       vm.generateVerifyToken(),
		}
		err := vm.TableSpec().Insert(ctx, ret)
		if err != nil {
			return nil, err
		}
	} else {
		now := time.Now()
		if now.Before(ret.CreatedAt.Add(time.Duration(options.Options.VerifyExpireInterval) * time.Minute)) {
			return nil, ErrVerifyFrequently
		}
		_, err := db.Update(ret, func() error {
			ret.Token = vm.generateVerifyToken()
			ret.CreatedAt = now
			ret.UpdatedAt = now
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (vm *SVerificationManager) Get(receiverId, contactType string) (*SVerification, error) {
	q := vm.Query().Equals("receiver_id", receiverId).Equals("contact_type", contactType)
	var verification SVerification
	err := q.First(&verification)
	if err != nil {
		return nil, err
	}
	verification.SetModelManager(vm, &verification)
	return &verification, nil
}
