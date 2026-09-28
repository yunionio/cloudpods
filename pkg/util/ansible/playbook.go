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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
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
	Inventory   Inventory
	Modules     []Module
	PrivateKeys []string
	Files       map[string][]byte

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
	pb1.PrivateKeys = gotypes.DeepCopy(pb.PrivateKeys).([]string)
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

func (pb *Playbook) Run(ctx context.Context) error {
	errs := make([]error, 0, len(pb.PrivateKeys))
	for i, privateKey := range pb.PrivateKeys {
		err := pb.runOnce(ctx, privateKey)
		if err != nil {
			errs = append(errs, err)
			if i != len(pb.PrivateKeys)-1 {
				pb.state = pbStateInit
			}
		} else {
			return nil
		}
	}
	return errors.NewAggregate(errs)
}

// Run runs the playbook
func (pb *Playbook) runOnce(ctx context.Context, privateKey string) error {
	var (
		tmpdir string
	)

	pb.stateMux.Lock()
	if pb.state != pbStateInit {
		return errors.Wrapf(errors.ErrInvalidStatus, "playbook state %s, want %s",
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
		return errors.Wrapf(errors.ErrInvalidFormat, "empty inventory")
	}

	// make tmpdir
	tmpdir, err := os.MkdirTemp("", "onecloud-ansible")
	if err != nil {
		return errors.Wrap(err, "making tmp dir")
	}
	pb.tmpdir = tmpdir
	defer func() {
		if pb.noCleanOnExit {
			return
		}
		if err1 := os.RemoveAll(tmpdir); err1 != nil {
			err = errors.Wrapf(err1, "removing %q", tmpdir)
		}
	}()

	// write out inventory
	inventory := filepath.Join(tmpdir, "inventory")
	err = os.WriteFile(inventory, pb.Inventory.Data(), os.FileMode(0600))
	if err != nil {
		return errors.Wrapf(err, "writing inventory %s", inventory)
	}

	// write out private key
	var privateKeyFile string
	if len(privateKey) > 0 {
		privateKeyFile = filepath.Join(tmpdir, "private_key")
		err = os.WriteFile(privateKeyFile, []byte(privateKey), os.FileMode(0600))
		if err != nil {
			return errors.Wrapf(err, "writing private key %s", privateKeyFile)
		}
	}

	// write out files
	for name, content := range pb.Files {
		path, err := fileutils2.JoinInside(tmpdir, name)
		if err != nil {
			return errors.Wrapf(err, "playbook file %s", name)
		}
		dir := filepath.Dir(path)
		err = os.MkdirAll(dir, os.FileMode(0700))
		if err != nil {
			return errors.Wrapf(err, "mkdir -p %s", dir)
		}
		err = os.WriteFile(path, content, os.FileMode(0600))
		if err != nil {
			return errors.Wrapf(err, "writing file %s", name)
		}
	}

	// run modules one by one
	var errs []error
	for _, m := range pb.Modules {
		select {
		case <-ctx.Done():
			errs = append(errs, errors.Wrap(ctx.Err(), "context done"))
			return errors.NewAggregate(errs)
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
			args = append(args, "--private-key", privateKeyFile)
		}
		cmd := exec.CommandContext(ctx, "ansible", args...)
		cmd.Dir = pb.tmpdir
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "ANSIBLE_HOST_KEY_CHECKING=False")
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err1 := cmd.Start(); err1 != nil {
			errs = append(errs, errors.Wrapf(err1, "run module %q, args %q", m.Name, modArgs))
			continue
		}
		// Mix stdout, stderr
		if pb.outputWriter != nil {
			go io.Copy(pb.outputWriter, stdout)
			go io.Copy(pb.outputWriter, stderr)
		}
		if err1 := cmd.Wait(); err1 != nil {
			errs = append(errs, errors.Wrapf(err1, "wait module %q, args %q", m.Name, modArgs))
			continue
		}
	}
	return errors.NewAggregate(errs)
}

func (pb *Playbook) OutputWriter(w io.Writer) {
	pb.outputWriter = w
}
