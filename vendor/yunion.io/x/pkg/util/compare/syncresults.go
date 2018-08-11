package compare

import (
	"fmt"
	"strings"
)

type SyncResult struct {
	AddCnt       int
	AddErrCnt    int
	UpdateCnt    int
	UpdateErrCnt int
	DelCnt       int
	DelErrCnt    int
	errors       []error
}

func (self *SyncResult) Error(msg error) {
	if self.errors == nil {
		self.errors = make([]error, 0)
	}
	self.errors = append(self.errors, msg)
}

func (self *SyncResult) Add() {
	self.AddCnt += 1
}

func (self *SyncResult) AddError(msg error) {
	self.AddErrCnt += 1
	self.Error(msg)
}

func (self *SyncResult) Update() {
	self.UpdateCnt += 1
}

func (self *SyncResult) UpdateError(msg error) {
	self.UpdateErrCnt += 1
	self.Error(msg)
}

func (self *SyncResult) Delete() {
	self.DelCnt += 1
}

func (self *SyncResult) DeleteError(msg error) {
	self.DelErrCnt += 1
	self.Error(msg)
}

func (self *SyncResult) AllError() error {
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

func (self *SyncResult) IsError() bool {
	return self.errors != nil && len(self.errors) > 0
}

func (self *SyncResult) Result() string {
	msg := fmt.Sprintf("removed %d failed %d updated %d failed %d added %d failed %d", self.DelCnt, self.DelErrCnt, self.UpdateCnt, self.UpdateErrCnt, self.AddCnt, self.AddErrCnt)
	if self.IsError() {
		msg = fmt.Sprintf("%s;%s", msg, self.AllError())
	}
	return msg
}
