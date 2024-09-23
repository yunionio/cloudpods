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

package nerdctl

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

type Nerdctl interface {
	Commit(ctrId string, opt *CommitOptions) (string, error)
}

type CommitOptions struct {
	Repository string
}

type nerdctl struct {
	address   string
	namespace string
}

func NewNerdctl(address, namespace string) Nerdctl {
	return &nerdctl{
		address:   address,
		namespace: namespace,
	}
}

func (n nerdctl) newCmd(args ...string) *procutils.Command {
	newArgs := []string{"--address", n.address}
	if n.namespace != "" {
		newArgs = append(newArgs, "--namespace", n.namespace)
	}
	newArgs = append(newArgs, args...)
	return procutils.NewCommand("nerdctl", newArgs...)
}

func (n nerdctl) Commit(ctrId string, opt *CommitOptions) (string, error) {
	if opt.Repository == "" {
		return "", errors.Wrap(errors.ErrEmpty, "repository")
	}
	cmd := n.newCmd("commit", ctrId, opt.Repository)
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "commit %s %s: %s", ctrId, opt.Repository, out)
	}
	return opt.Repository, nil
}
