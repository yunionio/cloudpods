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

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

var mpsControlBin = "nvidia-cuda-mps-control"

type Daemon struct {
	logDir  string
	pipeDir string

	replicas int
}

func NewDaemon(logDir, pipeDir string, replicas int) (*Daemon, error) {
	if err := os.MkdirAll(pipeDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating directory %v: %s", pipeDir, err)
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating directory %v: %s", logDir, err)
	}

	return &Daemon{
		logDir:  logDir,
		pipeDir: pipeDir,

		replicas: replicas,
	}, nil
}

type envvars map[string]string

func (e envvars) toSlice() []string {
	var envs []string
	for k, v := range e {
		envs = append(envs, k+"="+v)
	}
	return envs
}

func (d *Daemon) LogDir() string {
	return d.logDir
}

func (d *Daemon) PipeDir() string {
	return d.pipeDir
}

func (d *Daemon) Envvars() envvars {
	return map[string]string{
		"CUDA_MPS_PIPE_DIRECTORY": d.PipeDir(),
		"CUDA_MPS_LOG_DIRECTORY":  d.LogDir(),
	}
}

// EchoPipeToControl sends the specified command to the MPS control daemon.
func (d *Daemon) EchoPipeToControl(command string) (string, error) {
	var out bytes.Buffer
	reader, writer := io.Pipe()
	defer writer.Close()
	defer reader.Close()

	mpsDaemon := exec.Command(mpsControlBin)
	mpsDaemon.Env = append(mpsDaemon.Env, d.Envvars().toSlice()...)

	mpsDaemon.Stdin = reader
	mpsDaemon.Stdout = &out

	if err := mpsDaemon.Start(); err != nil {
		return "", fmt.Errorf("failed to start NVIDIA MPS command: %w", err)
	}

	if _, err := writer.Write([]byte(command)); err != nil {
		return "", fmt.Errorf("failed to write message to pipe: %w", err)
	}
	_ = writer.Close()

	if err := mpsDaemon.Wait(); err != nil {
		return "", fmt.Errorf("failed to send command to MPS daemon: %w", err)
	}
	return out.String(), nil
}

func parseMemSize(memTotalStr string) (int, error) {
	if !strings.HasSuffix(memTotalStr, " MiB") {
		return -1, fmt.Errorf("unknown mem string suffix")
	}
	memStr := strings.TrimSpace(strings.TrimSuffix(memTotalStr, " MiB"))
	return strconv.Atoi(memStr)
}

func (d *Daemon) Start() error {
	// nvidia-smi  --query-gpu=gpu_uuid,memory.total,compute_mode --format=csv
	// GPU-76aef7ff-372d-2432-b4b4-beca4d8d3400, 23040 MiB, Exclusive_Process
	out, err := exec.Command("nvidia-smi", "--query-gpu=index,memory.total,compute_mode", "--format=csv").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "nvidia-smi failed %s", out)
	}

	var devices = map[string]int{}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "index") {
			continue
		}
		segs := strings.Split(line, ",")
		if len(segs) != 3 {
			log.Errorf("unknown nvidia-smi out line %s", line)
			continue
		}
		gpuIdx, memTotal, computeMode := strings.TrimSpace(segs[0]), strings.TrimSpace(segs[1]), strings.TrimSpace(segs[2])
		if computeMode != "Exclusive_Process" {
			output, err := exec.Command("nvidia-smi", "-i", gpuIdx, "-c", "EXCLUSIVE_PROCESS").CombinedOutput()
			if err != nil {
				return fmt.Errorf("error running nvidia-smi: %s %s", output, err)
			}
		}

		memSize, err := parseMemSize(memTotal)
		if err != nil {
			return errors.Wrapf(err, "failed parse memSize %s", memTotal)
		}
		devices[gpuIdx] = memSize
	}

	mpsDaemon := exec.Command(mpsControlBin, "-d")
	mpsDaemon.Env = append(mpsDaemon.Env, d.Envvars().toSlice()...)
	if err := mpsDaemon.Run(); err != nil {
		return err
	}
	for deviceIdx, memory := range devices {
		memLimit := memory / d.replicas
		memLimitCmd := fmt.Sprintf("set_default_device_pinned_mem_limit %s %dM", deviceIdx, memLimit)
		log.Infof("set device mem limit cmd: %s", memLimitCmd)
		_, err := d.EchoPipeToControl(memLimitCmd)
		if err != nil {
			return fmt.Errorf("error set_default_device_pinned_mem_limit %s", err)
		}
	}

	threadPercentageCmd := fmt.Sprintf("set_default_active_thread_percentage %d", 100/d.replicas)
	_, err = d.EchoPipeToControl(threadPercentageCmd)
	if err != nil {
		return fmt.Errorf("error setting active thread percentage: %s", err)
	}

	return nil
}

func (d *Daemon) Stop() error {
	output, err := d.EchoPipeToControl("quit")
	if err != nil {
		return fmt.Errorf("error sending quit message: %s %s", output, err)
	}
	return nil
}

func main() {
	options.Init()
	isRoot := sysutils.IsRootPermission()
	if !isRoot {
		log.Fatalf("host service must running with root permissions")
		return
	}

	daemon, err := NewDaemon(
		options.HostOptions.CudaMPSLogDirectory,
		options.HostOptions.CudaMPSPipeDirectory,
		options.HostOptions.CudaMPSReplicas,
	)
	if err != nil {
		log.Fatalf(err.Error())
		return
	}

	var sigChan = make(chan struct{})
	signalutils.RegisterSignal(func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
	}, syscall.SIGUSR1)
	signalutils.RegisterSignal(func() {
		sigChan <- struct{}{}
	}, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	signalutils.StartTrap()

	if err = daemon.Start(); err != nil {
		log.Fatalf(err.Error())
	}

	log.Infof("MPS daemon started ......")
	select {
	case <-sigChan:
		if err := daemon.Stop(); err != nil {
			log.Errorf("failed stop daemon %s", err)
			os.Exit(1)
		}
	}
}
