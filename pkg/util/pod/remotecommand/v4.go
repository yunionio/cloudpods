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
Copyright 2016 The Kubernetes Authors.

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

package remotecommand

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// streamProtocolV4 implements version 4 of the streaming protocol for attach
// and exec. This version adds support for exit codes on the error stream through
// the use of metav1.Status instead of plain text messages.
type streamProtocolV4 struct {
	*streamProtocolV3
}

var _ streamProtocolHandler = &streamProtocolV4{}

func newStreamProtocolV4(options StreamOptions) streamProtocolHandler {
	return &streamProtocolV4{
		streamProtocolV3: newStreamProtocolV3(options).(*streamProtocolV3),
	}
}

func (p *streamProtocolV4) createStreams(conn streamCreator) error {
	return p.streamProtocolV3.createStreams(conn)
}

func (p *streamProtocolV4) handleResizes() {
	p.streamProtocolV3.handleResizes()
}

func (p *streamProtocolV4) stream(conn streamCreator) error {
	if err := p.createStreams(conn); err != nil {
		return err
	}

	// now that all the streams have been created, proceed with reading & copying

	errorChan := watchErrorStream(p.errorStream, &errorDecoderV4{})

	p.handleResizes()

	p.copyStdin()

	var wg sync.WaitGroup
	p.copyStdout(&wg)
	p.copyStderr(&wg)

	// we're waiting for stdout/stderr to finish copying
	wg.Wait()

	// waits for errorStream to finish reading with an error or nil
	return <-errorChan
}

// errorDecoderV4 interprets the json-marshaled metav1.Status on the error channel
// and creates an exec.ExitError from it.
type errorDecoderV4 struct{}

func (d *errorDecoderV4) decode(message []byte) error {
	status := metav1.Status{}
	err := json.Unmarshal(message, &status)
	if err != nil {
		return fmt.Errorf("error stream protocol error: %v in %q", err, string(message))
	}
	switch status.Status {
	case metav1.StatusSuccess:
		return nil
	case metav1.StatusFailure:
		if status.Reason == NonZeroExitCodeReason {
			if status.Details == nil {
				return errors.New("error stream protocol error: details must be set")
			}
			for i := range status.Details.Causes {
				c := &status.Details.Causes[i]
				if c.Type != ExitCodeCauseType {
					continue
				}

				rc, err := strconv.ParseUint(c.Message, 10, 8)
				if err != nil {
					return fmt.Errorf("error stream protocol error: invalid exit code value %q", c.Message)
				}
				return CodeExitError{
					Err:  fmt.Errorf("command terminated with exit code %d", rc),
					Code: int(rc),
				}
			}

			return fmt.Errorf("error stream protocol error: no %s cause given", ExitCodeCauseType)
		}
	default:
		return errors.New("error stream protocol error: unknown error")
	}

	return fmt.Errorf(status.Message)
}

// ExitError is an interface that presents an API similar to os.ProcessState, which is
// what ExitError from os/exec is.  This is designed to make testing a bit easier and
// probably loses some of the cross-platform properties of the underlying library.
type ExitError interface {
	String() string
	Error() string
	Exited() bool
	ExitStatus() int
}

// CodeExitError is an implementation of ExitError consisting of an error object
// and an exit code (the upper bits of os.exec.ExitStatus).
type CodeExitError struct {
	Err  error
	Code int
}

var _ ExitError = CodeExitError{}

func (e CodeExitError) Error() string {
	return e.Err.Error()
}

func (e CodeExitError) String() string {
	return e.Err.Error()
}

func (e CodeExitError) Exited() bool {
	return true
}

func (e CodeExitError) ExitStatus() int {
	return e.Code
}
