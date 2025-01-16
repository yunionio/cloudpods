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

/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package exec

import (
	"bytes"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/exec"
	"yunion.io/x/onecloud/pkg/util/ioutils"
	"yunion.io/x/onecloud/pkg/util/probe"
)

const (
	maxReadLength = 10 * 1 << 10 // 10KB
)

// Prober is an interface defining the Probe object for container readiness/liveness checks.
type Prober interface {
	Probe(e exec.Cmd, info string) (probe.Result, string, error)
}

// New creates a Prober.
func New() Prober {
	return execProber{}
}

type execProber struct{}

// Probe executes a command to check the liveness/readiness of container
// from executing a command. Returns the Result status, command output, and
// errors if any.
func (pr execProber) Probe(e exec.Cmd, info string) (probe.Result, string, error) {
	var dataBuffer bytes.Buffer
	writer := ioutils.LimitWriter(&dataBuffer, maxReadLength)

	e.SetStderr(writer)
	e.SetStdout(writer)
	err := e.Start()
	if err == nil {
		err = e.Wait()
	}
	data := dataBuffer.Bytes()

	log.Debugf("Exec probe response: %q, error: %v", string(data), err)
	if err != nil {
		exit, ok := errors.Cause(err).(exec.ExitError)
		if ok {
			if exit.ExitStatus() == 0 {
				return probe.Success, string(data), nil
			}
			return probe.Failure, fmt.Sprintf("%s, %s: %s", info, exit, data), nil
		}

		timeoutErr, ok := err.(*TimeoutError)
		if ok {
			log.Warningf("Exec probe timed out after %s", timeoutErr.Timeout())
			return probe.Failure, string(data), nil
		}

		return probe.Unknown, "", err
	}
	return probe.Success, string(data), nil
}
