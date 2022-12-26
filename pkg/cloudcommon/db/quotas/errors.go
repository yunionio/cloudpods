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

package quotas

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/httperrors"
)

type SOutOfQuotaError struct {
	scope   rbacscope.TRbacScope
	name    string
	limit   int
	used    int
	request int
}

type SOutOfQuotaErrors struct {
	errors []SOutOfQuotaError
}

func (e *SOutOfQuotaError) Cause() error {
	return httperrors.ErrOutOfQuota
}

func (e *SOutOfQuotaError) Error() string {
	return fmt.Sprintf("[%s.%s] limit %d used %d request %d", e.scope, e.name, e.limit, e.used, e.request)
}

func (es *SOutOfQuotaErrors) Error() string {
	qs := make([]string, len(es.errors))
	for i := range es.errors {
		e := es.errors[i]
		qs[i] = e.Error()
	}
	return fmt.Sprintf("Out of quota: %s", strings.Join(qs, ", "))
}

func (es *SOutOfQuotaErrors) IsError() bool {
	if len(es.errors) == 0 {
		return false
	} else {
		return true
	}
}

func NewOutOfQuotaError() *SOutOfQuotaErrors {
	return &SOutOfQuotaErrors{
		errors: make([]SOutOfQuotaError, 0),
	}
}

func (es *SOutOfQuotaErrors) Add(quota IQuota, name string, limit int, used int, request int) {
	scope := quota.GetKeys().Scope()
	e := SOutOfQuotaError{
		scope:   scope,
		name:    name,
		limit:   limit,
		used:    used,
		request: request,
	}
	es.errors = append(es.errors, e)
}

func (es *SOutOfQuotaErrors) Cause() error {
	return httperrors.ErrOutOfQuota
}
