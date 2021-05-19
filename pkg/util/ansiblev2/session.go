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

package ansiblev2

import (
	"context"
	"io"
)

type Session struct {
	PlaybookSessionBase

	playbook     string
	requirements string
	files        map[string][]byte
}

func NewSession() *Session {
	sess := &Session{
		PlaybookSessionBase: NewPlaybookSessionBase(),
		files:               map[string][]byte{},
	}
	return sess
}

func (sess *Session) PrivateKey(s string) *Session {
	sess.privateKey = s
	return sess
}

func (sess *Session) Playbook(s string) *Session {
	sess.playbook = s
	return sess
}

func (sess *Session) Inventory(s string) *Session {
	sess.inventory = s
	return sess
}

func (sess *Session) Requirements(s string) *Session {
	sess.requirements = s
	return sess
}

func (sess *Session) AddFile(path string, data []byte) *Session {
	sess.files[path] = data
	return sess
}

func (sess *Session) RolePublic(public bool) *Session {
	sess.rolePublic = public
	return sess
}

func (sess *Session) Timeout(timeout int) *Session {
	sess.timeout = timeout
	return sess
}

func (sess *Session) RemoveFile(path string) []byte {
	data := sess.files[path]
	delete(sess.files, path)
	return data
}

func (sess *Session) Files(files map[string][]byte) *Session {
	sess.files = files
	return sess
}

func (sess *Session) OutputWriter(w io.Writer) *Session {
	sess.outputWriter = w
	return sess
}

func (sess *Session) KeepTmpdir(keep bool) *Session {
	sess.keepTmpdir = keep
	return sess
}

func (sess *Session) GetPlaybook() string {
	return sess.playbook
}

func (sess *Session) GetRequirements() string {
	return sess.requirements
}

func (sess *Session) GetFile() map[string][]byte {
	return sess.files
}

func (sess *Session) Run(ctx context.Context) (err error) {
	return runnable{sess}.Run(ctx)
}
