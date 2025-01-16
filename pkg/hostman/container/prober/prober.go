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

package prober

import (
	"fmt"
	"io"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/container/prober/results"
	"yunion.io/x/onecloud/pkg/hostman/guestman/container"
	"yunion.io/x/onecloud/pkg/util/exec"
	"yunion.io/x/onecloud/pkg/util/probe"
	execprobe "yunion.io/x/onecloud/pkg/util/probe/exec"
	tcpprobe "yunion.io/x/onecloud/pkg/util/probe/tcp"
)

const maxProbeRetries = 3

// Prober helps to check the liveness of a container.
type prober struct {
	exec   execprobe.Prober
	tcp    tcpprobe.Prober
	runner container.CommandRunner
}

func newProber(runner container.CommandRunner) *prober {
	return &prober{
		exec:   execprobe.New(),
		tcp:    tcpprobe.New(),
		runner: runner,
	}
}

// probe probes the container.
func (pb *prober) probe(probeType apis.ContainerProbeType, pod IPod, container *hostapi.ContainerDesc) (results.ProbeResult, error) {
	var probeSpec *apis.ContainerProbe
	switch probeType {
	//case apis.ContainerProbeTypeLiveness:
	//	probeSpec = container.Spec.LivenessProbe
	case apis.ContainerProbeTypeStartup:
		probeSpec = container.Spec.StartupProbe
	default:
		err := errors.Errorf("unknown probe type: %q", probeType)
		return results.NewFailure(err.Error()), err
	}

	ctrName := fmt.Sprintf("%s:%s", pod.GetDesc().Name, container.Name)
	if probeSpec == nil {
		log.Warningf("%s probe for %s is nil", probeType, ctrName)
		return results.NewSuccess("probe is not defined"), nil
	}

	result, output, err := pb.runProbeWithRetries(probeType, probeSpec, pod, container, maxProbeRetries)
	var msg string
	if err != nil || (result != probe.Success && result != probe.Warning) {
		// Probe failed in one way or another
		if err != nil {
			msg = fmt.Sprintf("%s probe for %q errored: %v", probeType, ctrName, err)
			log.Errorf(msg)
		} else {
			// result != probe.Success
			msg = fmt.Sprintf("%s probe for %q failed (%v): %s", probeType, ctrName, result, output)
			log.Debugf(msg)
		}
		return results.NewFailure(msg), err
	}
	if result == probe.Warning {
		msg = fmt.Sprintf("%s probe for %q succeeded with a warning: %s", probeType, ctrName, output)
		log.Warningf(msg)
	} else {
		msg = fmt.Sprintf("%s probe for %q succeeded", probeType, ctrName)
		log.Debugf(msg)
	}
	return results.NewSuccess(msg), nil
}

// runProbeWithRetries tries to probe the container in a finite loop, it returns the last result
// if it never succeeds.
func (pb *prober) runProbeWithRetries(probeType apis.ContainerProbeType, p *apis.ContainerProbe, pod IPod, container *hostapi.ContainerDesc, retries int) (probe.Result, string, error) {
	var err error
	var result probe.Result
	var output string
	for i := 0; i < retries; i++ {
		result, output, err = pb.runProbe(probeType, p, pod, container)
		if err == nil {
			return result, output, nil
		}
	}
	return result, output, err
}

func (pb *prober) runProbe(probeType apis.ContainerProbeType, p *apis.ContainerProbe, pod IPod, container *hostapi.ContainerDesc) (probe.Result, string, error) {
	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	if p.Exec != nil {
		log.Debugf("Exec-Probe Pod: %v, Container: %v, Command: %v", pod.GetDesc().Name, container.Name, p.Exec.Command)
		return pb.exec.Probe(pb.newExecInContainer(pod, container, p.Exec.Command, timeout), strings.Join(p.Exec.Command, " "))
	}
	if p.TCPSocket != nil {
		port := p.TCPSocket.Port
		host := p.TCPSocket.Host
		if host == "" {
			for _, nic := range pod.GetDesc().Nics {
				if nic.Ip != "" {
					host = nic.Ip
					break
				}
			}
			if host == "" {
				return probe.Unknown, "", errors.Errorf("not found guest ip")
			}
		}
		log.Debugf("TCP-Probe Host: %v, Port: %v, Timeout: %v", host, port, timeout)
		return pb.tcp.Probe(host, port, timeout)
	}
	errMsg := fmt.Sprintf("Failed to find probe builder for pod %v, container: %v", pod.GetName(), container.Name)
	log.Warningf(errMsg)
	return probe.Unknown, "", errors.Error(errMsg)
}

type execInContainer struct {
	// run executes a command in a container. Combined stdout and stderr output is always returned. An
	// error is returned if one occurred.
	run    func() ([]byte, error)
	writer io.Writer
	cmd    []string
}

func (pb *prober) newExecInContainer(pod IPod, container *hostapi.ContainerDesc, cmd []string, timeout time.Duration) exec.Cmd {
	return &execInContainer{
		cmd: cmd,
		run: func() ([]byte, error) {
			return pb.runner.RunInContainer(pod.GetId(), container.Id, cmd, timeout)
		},
	}
}

func (eic *execInContainer) Command() []string {
	return eic.cmd
}

func (eic *execInContainer) Run() error {
	return nil
}

func (eic *execInContainer) CombinedOutput() ([]byte, error) {
	return eic.run()
}

func (eic *execInContainer) Output() ([]byte, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (eic *execInContainer) SetDir(dir string) {
	//unimplemented
}

func (eic *execInContainer) SetStdin(in io.Reader) {
	//unimplemented
}

func (eic *execInContainer) SetStdout(out io.Writer) {
	eic.writer = out
}

func (eic *execInContainer) SetStderr(out io.Writer) {
	eic.writer = out
}

func (eic *execInContainer) SetEnv(env []string) {
	//unimplemented
}

func (eic *execInContainer) Stop() {
	//unimplemented
}

func (eic *execInContainer) Start() error {
	data, err := eic.run()
	if eic.writer != nil {
		eic.writer.Write(data)
	}
	return err
}

func (eic *execInContainer) Wait() error {
	return nil
}

func (eic *execInContainer) StdoutPipe() (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (eic *execInContainer) StderrPipe() (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
}
