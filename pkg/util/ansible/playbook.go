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

package ansible

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"yunion.io/x/pkg/gotypes"
	yerrors "yunion.io/x/pkg/util/errors"
)

type pbState int

const (
	pbStateInit pbState = iota
	pbStateRunning
	pbStateStopped
)

func (pbs pbState) String() string {
	switch pbs {
	case pbStateInit:
		return "init"
	case pbStateRunning:
		return "running"
	case pbStateStopped:
		return "stopped"
	}
	return "unknown"
}

type Playbook struct {
	Inventory  Inventory
	Modules    []Module
	PrivateKey []byte
	Files      map[string][]byte

	tmpdir        string
	noCleanOnExit bool
	outputWriter  io.Writer
	state         pbState
	stateMux      *sync.Mutex
}

func NewPlaybook() *Playbook {
	pb := &Playbook{
		state:    pbStateInit,
		stateMux: &sync.Mutex{},
	}
	return pb
}

func (pb *Playbook) Copy() *Playbook {
	pb1 := NewPlaybook()
	pb1.Inventory = gotypes.DeepCopy(pb.Inventory).(Inventory)
	pb1.Modules = gotypes.DeepCopy(pb.Modules).([]Module)
	pb1.PrivateKey = gotypes.DeepCopy(pb.PrivateKey).([]byte)
	pb1.Files = gotypes.DeepCopy(pb.Files).(map[string][]byte)
	return pb1
}

// CleanOnExit decide whether temporary workdir will be cleaned up after Run
func (pb *Playbook) CleanOnExit(b bool) {
	pb.noCleanOnExit = !b
}

// State returns current state of the playbook
func (pb *Playbook) State() pbState {
	pb.stateMux.Lock()
	defer pb.stateMux.Unlock()
	return pb.state
}

// Runnable returns whether the playbook is in a state feasible to be run
func (pb *Playbook) Runnable() bool {
	return pb.state == pbStateInit
}

// Running returns whether the playbook is currently running
func (pb *Playbook) Running() bool {
	return pb.state == pbStateRunning
}

// Run runs the playbook
func (pb *Playbook) Run(ctx context.Context) (err error) {
	var (
		tmpdir string
	)

	pb.stateMux.Lock()
	if pb.state != pbStateInit {
		return errors.Errorf("playbook state %s, want %s",
			pb.state, pbStateInit)
	}
	pb.state = pbStateRunning
	pb.stateMux.Unlock()
	defer func() {
		pb.stateMux.Lock()
		pb.state = pbStateStopped
		pb.stateMux.Unlock()
	}()

	if pb.Inventory.IsEmpty() {
		return errors.New("empty inventory")
	}

	// make tmpdir
	tmpdir, err = ioutil.TempDir("", "onecloud-ansible")
	if err != nil {
		err = errors.WithMessage(err, "making tmp dir")
		return
	}
	pb.tmpdir = tmpdir
	defer func() {
		if pb.noCleanOnExit {
			return
		}
		if err1 := os.RemoveAll(tmpdir); err1 != nil {
			err = errors.WithMessagef(err1, "removing %q", tmpdir)
		}
	}()

	// write out inventory
	inventory := filepath.Join(tmpdir, "inventory")
	err = ioutil.WriteFile(inventory, pb.Inventory.Data(), os.FileMode(0600))
	if err != nil {
		err = errors.WithMessagef(err, "writing inventory %s", inventory)
		return
	}

	// write out private key
	var privateKey string
	if len(pb.PrivateKey) > 0 {
		privateKey = filepath.Join(tmpdir, "private_key")
		err = ioutil.WriteFile(privateKey, pb.PrivateKey, os.FileMode(0600))
		if err != nil {
			err = errors.WithMessagef(err, "writing private key %s", privateKey)
			return
		}
	}

	// write out files
	for name, content := range pb.Files {
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
	for _, m := range pb.Modules {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}
		modArgs := strings.Join(m.Args, " ")
		args := []string{
			"--inventory", inventory,
			"--module-name", m.Name,
			"--args", modArgs,
			"all",
		}
		if privateKey != "" {
			args = append(args, "--private-key", privateKey)
		}
		cmd := exec.CommandContext(ctx, "ansible", args...)
		cmd.Dir = pb.tmpdir
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "ANSIBLE_HOST_KEY_CHECKING=False")
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err1 := cmd.Start(); err1 != nil {
			errs = append(errs, errors.WithMessagef(err1, "run module %q, args %q", m.Name, modArgs))
			return
		}
		// Mix stdout, stderr
		if pb.outputWriter != nil {
			go io.Copy(pb.outputWriter, stdout)
			go io.Copy(pb.outputWriter, stderr)
		}
		if err1 := cmd.Wait(); err1 != nil {
			errs = append(errs, errors.WithMessagef(err1, "wait module %q, args %q", m.Name, modArgs))
			// continue to next
		}
	}
	return nil
}

func (pb *Playbook) OutputWriter(w io.Writer) {
	pb.outputWriter = w
}
