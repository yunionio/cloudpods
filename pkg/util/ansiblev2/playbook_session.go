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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"

	"github.com/go-yaml/yaml"

	"yunion.io/x/pkg/errors"
	yerrors "yunion.io/x/pkg/util/errors"
)

type IPlaybookSession interface {
	GetPrivateKey() string
	GetPlaybook() string
	GetPlaybookPath() string
	GetInventory() string
	IsKeepTmpdir() bool
	GetConfigs() map[string]interface{}
	GetRequirements() string
	GetFiles() map[string][]byte
	GetOutputWriter() io.Writer
	GetRolePublic() bool
	GetTimeout() int
	CheckAndSetRunning() bool
	SetStopped()
	GetConfigYaml() string
}

type runnable struct {
	IPlaybookSession
}

func (r runnable) Run(ctx context.Context) (err error) {
	var (
		tmpdir string
	)

	has := r.CheckAndSetRunning()
	if has {
		return errors.Errorf("playbook is already running")
	}
	defer r.SetStopped()

	// make tmpdir
	tmpdir, err = os.MkdirTemp("", "onecloud-ansiblev2")
	if err != nil {
		err = errors.Wrap(err, "making tmp dir")
		return
	}
	defer func() {
		if r.IsKeepTmpdir() {
			return
		}
		if err1 := os.RemoveAll(tmpdir); err1 != nil {
			err = errors.Wrapf(err1, "removing %q", tmpdir)
		}
	}()

	// write out inventory
	inventory := filepath.Join(tmpdir, "inventory")
	err = os.WriteFile(inventory, []byte(r.GetInventory()), os.FileMode(0600))
	if err != nil {
		err = errors.Wrapf(err, "writing inventory %s", inventory)
		return
	}

	// write out playbook
	playbook := r.GetPlaybookPath()
	if len(playbook) == 0 {
		playbook = filepath.Join(tmpdir, "playbook")
		err = os.WriteFile(playbook, []byte(r.GetPlaybook()), os.FileMode(0600))
		if err != nil {
			err = errors.Wrapf(err, "writing playbook %s", playbook)
			return
		}
	}

	// write out private key
	var privateKey string
	if len(r.GetPrivateKey()) > 0 {
		privateKey = filepath.Join(tmpdir, "private_key")
		err = os.WriteFile(privateKey, []byte(r.GetPrivateKey()), os.FileMode(0600))
		if err != nil {
			err = errors.Wrapf(err, "writing private key %s", privateKey)
			return
		}
	}

	// write out requirements
	var requirements string
	if len(r.GetRequirements()) > 0 {
		requirements = filepath.Join(tmpdir, "requirements.yml")
		err = os.WriteFile(requirements, []byte(r.GetRequirements()), os.FileMode(0600))
		if err != nil {
			err = errors.Wrapf(err, "writing requirements %s", requirements)
			return
		}
	}

	// write out files
	for name, content := range r.GetFiles() {
		path := filepath.Join(tmpdir, name)
		dir := filepath.Dir(path)
		err = os.MkdirAll(dir, os.FileMode(0700))
		if err != nil {
			err = errors.Wrapf(err, "mkdir -p %s", dir)
			return
		}
		err = os.WriteFile(path, content, os.FileMode(0600))
		if err != nil {
			err = errors.Wrapf(err, "writing file %s", name)
			return
		}
	}

	// write out configs
	var config string
	if r.GetConfigs() != nil {
		yml, err := yaml.Marshal(r.GetConfigs())
		if err != nil {
			return errors.Wrap(err, "unable to marshal map to yaml")
		}
		config = filepath.Join(tmpdir, "config")
		err = os.WriteFile(config, yml, os.FileMode(0600))
		if err != nil {
			return errors.Wrapf(err, "unable to write config to file %s", config)
		}
	} else if r.GetConfigYaml() != "" {
		yml := r.GetConfigYaml()
		config = filepath.Join(tmpdir, "config")
		err = os.WriteFile(config, []byte(yml), os.FileMode(0600))
		if err != nil {
			return errors.Wrapf(err, "unable to write config to file %s", config)
		}
	}

	// run modules one by one
	var errs []error
	defer func() {
		if len(errs) > 0 {
			err = yerrors.NewAggregate(errs)
		}
	}()

	// install required roles
	if len(requirements) > 0 {
		args := []string{
			"install", "-r", requirements,
		}
		if !r.GetRolePublic() {
			args = append(args, "-p", tmpdir)
		}
		cmd := exec.CommandContext(ctx, "ansible-galaxy", args...)
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err1 := cmd.Start(); err1 != nil {
			errs = append(errs, errors.Wrap(err1, "start ansible-galaxy install roles"))
			return
		}
		// Mix stdout, stderr

		if writer := r.GetOutputWriter(); writer != nil {
			go io.Copy(writer, stdout)
			go io.Copy(writer, stderr)
		}
		if err1 := cmd.Wait(); err1 != nil {
			errs = append(errs, errors.Wrap(err1, "wait ansible-galaxy install roles"))
		}
	}

	// run playbook
	{
		args := []string{
			"--inventory", inventory, "--timeout", fmt.Sprintf("%d", r.GetTimeout()),
		}
		if config != "" {
			args = append(args, "-e", "@"+config)
		}
		if privateKey != "" {
			args = append(args, "--private-key", privateKey)
		}
		args = append(args, playbook)
		cmd := exec.CommandContext(ctx, "ansible-playbook", args...)
		cmd.Dir = tmpdir
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "ANSIBLE_HOST_KEY_CHECKING=False")
		// for debug
		os.WriteFile(path.Join(tmpdir, "run_cmd"), []byte(cmd.String()), os.ModePerm)
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err1 := cmd.Start(); err1 != nil {
			errs = append(errs, errors.Wrapf(err1, "start playbook %s", playbook))
			return
		}
		// Mix stdout, stderr
		if writer := r.GetOutputWriter(); writer != nil {
			go io.Copy(writer, stdout)
			go io.Copy(writer, stderr)
		}
		if err1 := cmd.Wait(); err1 != nil {
			errs = append(errs, errors.Wrapf(err1, "wait playbook %s", playbook))
		}
	}
	return nil
}

type PlaybookSessionBase struct {
	privateKey string

	inventory    string
	outputWriter io.Writer
	stateMux     *sync.Mutex
	isRunning    bool
	keepTmpdir   bool
	rolePublic   bool
	timeout      int
}

func NewPlaybookSessionBase() PlaybookSessionBase {
	return PlaybookSessionBase{
		stateMux: &sync.Mutex{},
		timeout:  10,
	}
}

func (pb *PlaybookSessionBase) GetPrivateKey() string {
	return pb.privateKey
}

func (pb *PlaybookSessionBase) IsKeepTmpdir() bool {
	return pb.keepTmpdir
}

func (pb *PlaybookSessionBase) GetOutputWriter() io.Writer {
	return pb.outputWriter
}

func (pb *PlaybookSessionBase) CheckAndSetRunning() bool {
	pb.stateMux.Lock()
	if pb.isRunning {
		return true
	}
	pb.isRunning = true
	pb.stateMux.Unlock()
	return false
}

func (pb *PlaybookSessionBase) SetStopped() {
	pb.stateMux.Lock()
	pb.isRunning = false
	pb.stateMux.Unlock()
}

func (pb *PlaybookSessionBase) GetPlaybook() string {
	return ""
}

func (pb *PlaybookSessionBase) GetPlaybookPath() string {
	return ""
}

func (pb *PlaybookSessionBase) GetInventory() string {
	return pb.inventory
}

func (pb *PlaybookSessionBase) GetConfigs() map[string]interface{} {
	return nil
}

func (pb *PlaybookSessionBase) GetConfigYaml() string {
	return ""
}

func (pb *PlaybookSessionBase) GetRequirements() string {
	return ""
}

func (pb *PlaybookSessionBase) GetFiles() map[string][]byte {
	return nil
}

func (pb *PlaybookSessionBase) GetRolePublic() bool {
	return pb.rolePublic
}

func (pb *PlaybookSessionBase) GetTimeout() int {
	return pb.timeout
}
