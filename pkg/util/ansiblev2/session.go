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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	yerrors "yunion.io/x/pkg/util/errors"
)

type Session struct {
	privateKey string
	playbook   string
	inventory  string
	files      map[string][]byte

	outputWriter io.Writer
	stateMux     *sync.Mutex
	isRunning    bool
	keepTmpdir   bool
}

func NewSession() *Session {
	sess := &Session{
		stateMux: &sync.Mutex{},
		files:    map[string][]byte{},
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

func (sess *Session) AddFile(path string, data []byte) *Session {
	sess.files[path] = data
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

func (sess *Session) Run(ctx context.Context) (err error) {
	var (
		tmpdir string
	)

	sess.stateMux.Lock()
	if sess.isRunning {
		return errors.Errorf("playbook is already running")
	}
	sess.isRunning = true
	sess.stateMux.Unlock()
	defer func() {
		sess.stateMux.Lock()
		sess.isRunning = false
		sess.stateMux.Unlock()
	}()

	// make tmpdir
	tmpdir, err = ioutil.TempDir("", "onecloud-ansiblev2")
	if err != nil {
		err = errors.WithMessage(err, "making tmp dir")
		return
	}
	defer func() {
		if sess.keepTmpdir {
			return
		}
		if err1 := os.RemoveAll(tmpdir); err1 != nil {
			err = errors.WithMessagef(err1, "removing %q", tmpdir)
		}
	}()

	// write out inventory
	inventory := filepath.Join(tmpdir, "inventory")
	err = ioutil.WriteFile(inventory, []byte(sess.inventory), os.FileMode(0600))
	if err != nil {
		err = errors.WithMessagef(err, "writing inventory %s", inventory)
		return
	}

	// write out playbook
	playbook := filepath.Join(tmpdir, "playbook")
	err = ioutil.WriteFile(playbook, []byte(sess.playbook), os.FileMode(0600))
	if err != nil {
		err = errors.WithMessagef(err, "writing playbook %s", playbook)
		return
	}

	// write out private key
	var privateKey string
	if len(sess.privateKey) > 0 {
		privateKey = filepath.Join(tmpdir, "private_key")
		err = ioutil.WriteFile(privateKey, []byte(sess.privateKey), os.FileMode(0600))
		if err != nil {
			err = errors.WithMessagef(err, "writing private key %s", privateKey)
			return
		}
	}

	// write out files
	for name, content := range sess.files {
		path := filepath.Join(tmpdir, name)
		dir := filepath.Dir(path)
		err = os.MkdirAll(dir, os.FileMode(0700))
		if err != nil {
			err = errors.WithMessagef(err, "mkdir -p %s", dir)
			return
		}
		err = ioutil.WriteFile(path, content, os.FileMode(0600))
		if err != nil {
			err = errors.WithMessagef(err, "writing file %s", name)
			return
		}
	}

	// run modules one by one
	var errs []error
	defer func() {
		if len(errs) > 0 {
			err = yerrors.NewAggregate(errs)
		}
	}()

	{
		args := []string{
			"--inventory", inventory,
		}
		if privateKey != "" {
			args = append(args, "--private-key", privateKey)
		}
		args = append(args, playbook)
		cmd := exec.CommandContext(ctx, "ansible-playbook", args...)
		cmd.Dir = tmpdir
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "ANSIBLE_HOST_KEY_CHECKING=False")
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err1 := cmd.Start(); err1 != nil {
			errs = append(errs, errors.WithMessagef(err1, "start playbook %s", playbook))
			return
		}
		// Mix stdout, stderr
		if sess.outputWriter != nil {
			go io.Copy(sess.outputWriter, stdout)
			go io.Copy(sess.outputWriter, stderr)
		}
		if err1 := cmd.Wait(); err1 != nil {
			errs = append(errs, errors.WithMessagef(err1, "wait playbook %s", playbook))
		}
	}
	return nil
}
