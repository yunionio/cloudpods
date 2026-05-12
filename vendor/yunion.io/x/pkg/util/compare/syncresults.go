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

package compare

import (
	"fmt"
	"strings"
	"sync"

	"yunion.io/x/pkg/errors"
)

type SyncResult struct {
	mu sync.Mutex `json:"-"`

	AddCnt       int `json:"add_cnt,omitzero"`
	AddErrCnt    int `json:"add_err_cnt,omitzero"`
	UpdateCnt    int `json:"update_cnt,omitzero"`
	UpdateErrCnt int `json:"update_err_cnt,omitzero"`
	DelCnt       int `json:"del_cnt,omitzero"`
	DelErrCnt    int `json:"del_err_cnt,omitzero"`
	errors       []error
}

func (self *SyncResult) appendError(msg error) {
	if self.errors == nil {
		self.errors = make([]error, 0)
	}
	self.errors = append(self.errors, msg)
}

func (self *SyncResult) Error(msg error) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.appendError(msg)
}

func (self *SyncResult) Add() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.AddCnt += 1
}

func (self *SyncResult) AddError(msg error) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.AddErrCnt += 1
	self.appendError(msg)
}

func (self *SyncResult) Update() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.UpdateCnt += 1
}

func (self *SyncResult) UpdateError(msg error) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.UpdateErrCnt += 1
	self.appendError(msg)
}

func (self *SyncResult) Delete() {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.DelCnt += 1
}

func (self *SyncResult) DeleteError(msg error) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.DelErrCnt += 1
	self.appendError(msg)
}

func (self *SyncResult) allErrorUnlocked() error {
	msgs := make(map[string]bool)
	for _, e := range self.errors {
		msg := e.Error()
		msgs[msg] = true
	}
	ret := make([]string, len(msgs))
	i := 0
	for m := range msgs {
		ret[i] = m
		i += 1
	}
	return fmt.Errorf(strings.Join(ret, ";"))
}

func (self *SyncResult) AllError() error {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.allErrorUnlocked()
}

func (self *SyncResult) IsError() bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	return len(self.errors) > 0
}

func (self *SyncResult) IsGenerateError() bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	for _, err := range self.errors {
		if e := errors.Cause(err); e != errors.ErrNotImplemented && e != errors.ErrNotSupported && e != errors.ErrNotEmpty {
			return true
		}
	}
	return false
}

func (self *SyncResult) Result() string {
	self.mu.Lock()
	defer self.mu.Unlock()
	msg := fmt.Sprintf("removed %d failed %d updated %d failed %d added %d failed %d", self.DelCnt, self.DelErrCnt, self.UpdateCnt, self.UpdateErrCnt, self.AddCnt, self.AddErrCnt)
	if len(self.errors) > 0 {
		msg = fmt.Sprintf("%s;%s", msg, self.allErrorUnlocked())
	}
	return msg
}
