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

	"yunion.io/x/pkg/errors"
)

type Session struct {
	PlaybookSessionBase

	playbook     string
	requirements string
	files        map[string][]byte
}

type trySession struct {
	*Session

	privateKeyIndex int
}

func (trySession *trySession) GetPrivateKey() string {
	return trySession.GetPrivateKeys()[trySession.privateKeyIndex]
}

func NewSession() *Session {
	sess := &Session{
		PlaybookSessionBase: NewPlaybookSessionBase(),
		files:               map[string][]byte{},
	}
	return sess
}

func (sess *Session) PrivateKeys(s []string) *Session {
	sess.privateKeys = s
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

func (sess *Session) Run(ctx context.Context) error {
	privateKeys := sess.GetPrivateKeys()
	errs := make([]error, 0, len(privateKeys))
	for i := range privateKeys {
		trySession := &trySession{
			Session:         sess,
			privateKeyIndex: i,
		}
		err := runnable{trySession}.Run(ctx)
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}
