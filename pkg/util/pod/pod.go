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

package pod

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/exec"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type CRI interface {
	Version(ctx context.Context) (*runtimeapi.VersionResponse, error)
	ListPods(ctx context.Context, opts ListPodOptions) ([]*runtimeapi.PodSandbox, error)
	RunPod(ctx context.Context, podConfig *runtimeapi.PodSandboxConfig, runtimeHandler string) (string, error)
	StopPod(ctx context.Context, req *runtimeapi.StopPodSandboxRequest) error
	RemovePod(ctx context.Context, podId string) error
	CreateContainer(ctx context.Context, podId string, podConfig *runtimeapi.PodSandboxConfig, ctrConfig *runtimeapi.ContainerConfig, withPull bool) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, ctrId string, timeout int64, tryRemove bool, force bool) error
	RemoveContainer(ctx context.Context, ctrId string) error
	RunContainers(ctx context.Context, podConfig *runtimeapi.PodSandboxConfig, containerConfigs []*runtimeapi.ContainerConfig, runtimeHandler string) (*RunContainersResponse, error)
	ListContainers(ctx context.Context, opts ListContainerOptions) ([]*runtimeapi.Container, error)
	ContainerStatus(ctx context.Context, ctrId string) (*runtimeapi.ContainerStatusResponse, error)
	ListImages(ctx context.Context, filter *runtimeapi.ImageFilter) ([]*runtimeapi.Image, error)
	PullImage(ctx context.Context, req *runtimeapi.PullImageRequest) (*runtimeapi.PullImageResponse, error)
	ImageStatus(ctx context.Context, req *runtimeapi.ImageStatusRequest) (*runtimeapi.ImageStatusResponse, error)
	ExecSync(ctx context.Context, ctrId string, command []string, timeout int64) (*ExecSyncResponse, error)

	// lower layer client
	GetImageClient() runtimeapi.ImageServiceClient
	GetRuntimeClient() runtimeapi.RuntimeServiceClient
}

type ListContainerOptions struct {
	// Id of container
	Id string
	// PodId of container
	PodId string
	// Regular expression pattern to match pod or container
	NameRegexp string
	// Regular expression pattern to match the pod namespace
	PodNamespaceRegexp string
	// State of the sandbox
	State string
	// Show verbose info for the sandbox
	Verbose bool
	// Labels are selectors for the sandbox
	Labels map[string]string
	// Image ued by the container
	Image string
}

type crictl struct {
	endpoint string
	timeout  time.Duration
	conn     *grpc.ClientConn

	imgCli runtimeapi.ImageServiceClient
	runCli runtimeapi.RuntimeServiceClient
}

func NewCRI(endpoint string, timeout time.Duration) (CRI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var dialOpts []grpc.DialOption
	maxMsgSize := 1024 * 1024 * 16
	dialOpts = append(dialOpts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize)))

	conn, err := grpc.DialContext(ctx, endpoint, dialOpts...)
	if err != nil {
		return nil, errors.Wrapf(err, "Connect remote endpoint %q failed", endpoint)
	}

	imgCli := runtimeapi.NewImageServiceClient(conn)
	runCli := runtimeapi.NewRuntimeServiceClient(conn)

	return &crictl{
		endpoint: endpoint,
		timeout:  timeout,
		conn:     conn,
		imgCli:   imgCli,
		runCli:   runCli,
	}, nil
}

func (c crictl) GetImageClient() runtimeapi.ImageServiceClient {
	return c.imgCli
}

func (c crictl) GetRuntimeClient() runtimeapi.RuntimeServiceClient {
	return c.runCli
}

func (c crictl) Version(ctx context.Context) (*runtimeapi.VersionResponse, error) {
	return c.GetRuntimeClient().Version(ctx, &runtimeapi.VersionRequest{})
}

func (c crictl) ListImages(ctx context.Context, filter *runtimeapi.ImageFilter) ([]*runtimeapi.Image, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.GetImageClient().ListImages(ctx, &runtimeapi.ListImagesRequest{
		Filter: filter,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "ListImages with filter %s", filter.String())
	}
	return resp.Images, nil
}

func (c crictl) RunPod(ctx context.Context, podConfig *runtimeapi.PodSandboxConfig, runtimeHandler string) (string, error) {
	req := &runtimeapi.RunPodSandboxRequest{
		Config:         podConfig,
		RuntimeHandler: runtimeHandler,
	}
	log.Infof("RunPodSandboxRequest: %v", req)
	r, err := c.GetRuntimeClient().RunPodSandbox(ctx, req)
	if err != nil {
		return "", errors.Wrapf(err, "RunPod with request: %s", req.String())
	}
	return r.GetPodSandboxId(), nil
}

func (c crictl) StopPod(ctx context.Context, req *runtimeapi.StopPodSandboxRequest) error {
	_, err := c.GetRuntimeClient().StopPodSandbox(ctx, req)
	if err != nil {
		return errors.Wrap(err, "StopPodSandbox")
	}
	return nil
}

// PullImageWithSandbox sends a PullImageRequest to the server and parses
// the returned PullImageResponses.
func (c crictl) PullImageWithSandbox(ctx context.Context, image string, auth *runtimeapi.AuthConfig, sandbox *runtimeapi.PodSandboxConfig, ann map[string]string) (*runtimeapi.PullImageResponse, error) {
	req := &runtimeapi.PullImageRequest{
		Image: &runtimeapi.ImageSpec{
			Image:       image,
			Annotations: ann,
		},
		Auth:          auth,
		SandboxConfig: sandbox,
	}
	log.Infof("PullImageRequest: %v", req)
	r, err := c.GetImageClient().PullImage(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "PullImage with %s", req)
	}
	return r, nil
}

// CreateContainer sends a CreateContainerRequest to the server, and parses
// the returned CreateContainerResponse.
func (c crictl) CreateContainer(ctx context.Context,
	podId string, podConfig *runtimeapi.PodSandboxConfig,
	ctrConfig *runtimeapi.ContainerConfig, withPull bool) (string, error) {

	req := &runtimeapi.CreateContainerRequest{
		PodSandboxId:  podId,
		Config:        ctrConfig,
		SandboxConfig: podConfig,
	}

	image := ctrConfig.GetImage().GetImage()

	// When there is a withPull request or the image default mode is to
	// pull-image-on-create(true) and no-pull was not set we pull the image when
	// they ask for a create as a helper on the cli to reduce extra steps. As a
	// reminder if the image is already in cache only the manifest will be pulled
	// down to verify.
	if withPull {
		// Try to pull the image before container creation
		ann := ctrConfig.GetImage().GetAnnotations()
		resp, err := c.PullImageWithSandbox(ctx, image, nil, nil, ann)
		if err != nil {
			return "", errors.Wrap(err, "PullImageWithSandbox")
		}
		log.Infof("Pull image %s", resp.String())
	}

	log.Debugf("CreateContainerRequest pod %s: %v", podId, req)
	r, err := c.GetRuntimeClient().CreateContainer(ctx, req)
	if err != nil {
		return "", errors.Wrapf(err, "CreateContainer with: %s", req)
	}
	return r.GetContainerId(), nil
}

// StartContainer sends a StartContainerRequest to the server, and parses
// the returned StartContainerResponse.
func (c crictl) StartContainer(ctx context.Context, id string) error {
	if id == "" {
		return errors.Error("Id can't be empty")
	}
	if _, err := c.GetRuntimeClient().StartContainer(ctx, &runtimeapi.StartContainerRequest{
		ContainerId: id,
	}); err != nil {
		return errors.Wrapf(err, "StartContainer %s", id)
	}
	return nil
}

type RunContainersResponse struct {
	PodId        string   `json:"pod_id"`
	ContainerIds []string `json:"container_ids"`
}

// RunContainers starts containers in the provided pod sandbox
func (c crictl) RunContainers(
	ctx context.Context, podConfig *runtimeapi.PodSandboxConfig, containerConfigs []*runtimeapi.ContainerConfig, runtimeHandler string) (*RunContainersResponse, error) {
	// Create the pod
	podId, err := c.RunPod(ctx, podConfig, runtimeHandler)
	if err != nil {
		return nil, errors.Wrap(err, "RunPod")
	}
	ret := &RunContainersResponse{
		PodId:        podId,
		ContainerIds: make([]string, 0),
	}
	// Create the containers
	for idx, ctr := range containerConfigs {
		// Create the container
		ctrId, err := c.CreateContainer(ctx, podId, podConfig, ctr, true)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateContainer %d", idx)
		}
		// Start the container
		if err := c.StartContainer(ctx, ctrId); err != nil {
			return nil, errors.Wrapf(err, "StarContainer %d", idx)
		}
		ret.ContainerIds = append(ret.ContainerIds, ctrId)
	}
	return ret, nil
}

func (c crictl) ListContainers(ctx context.Context, opts ListContainerOptions) ([]*runtimeapi.Container, error) {
	filter := &runtimeapi.ContainerFilter{
		Id:           opts.Id,
		PodSandboxId: opts.PodId,
	}
	st := &runtimeapi.ContainerStateValue{}
	if opts.State != "" {
		st.State = runtimeapi.ContainerState_CONTAINER_UNKNOWN
		switch strings.ToLower(opts.State) {
		case "created":
			st.State = runtimeapi.ContainerState_CONTAINER_CREATED
		case "running":
			st.State = runtimeapi.ContainerState_CONTAINER_RUNNING
		case "exited":
			st.State = runtimeapi.ContainerState_CONTAINER_EXITED
		case "unknown":
			st.State = runtimeapi.ContainerState_CONTAINER_UNKNOWN
		default:
			return nil, fmt.Errorf("unsupported state: %q", opts.State)
		}
		filter.State = st
	}
	if opts.Labels != nil {
		filter.LabelSelector = opts.Labels
	}
	req := &runtimeapi.ListContainersRequest{
		Filter: filter,
	}
	r, err := c.GetRuntimeClient().ListContainers(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "ListContainers")
	}
	return r.Containers, nil
}

type ListPodOptions struct {
	Id    string
	State string
}

func (c crictl) ListPods(ctx context.Context, opts ListPodOptions) ([]*runtimeapi.PodSandbox, error) {
	filter := &runtimeapi.PodSandboxFilter{
		Id: opts.Id,
	}
	if opts.State != "" {
		st := &runtimeapi.PodSandboxStateValue{}
		st.State = runtimeapi.PodSandboxState_SANDBOX_NOTREADY
		switch strings.ToLower(opts.State) {
		case "ready":
			st.State = runtimeapi.PodSandboxState_SANDBOX_READY
		case "notready":
			st.State = runtimeapi.PodSandboxState_SANDBOX_NOTREADY
		default:
			return nil, errors.Errorf("Invalid state: %q", opts.State)
		}
		filter.State = st
	}
	req := &runtimeapi.ListPodSandboxRequest{
		Filter: filter,
	}
	ret, err := c.GetRuntimeClient().ListPodSandbox(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "ListPodSandbox")
	}
	return ret.Items, nil
}

func (c crictl) RemovePod(ctx context.Context, podId string) error {
	maxTries := 10
	interval := 5 * time.Second
	errs := []error{}
	for tries := 0; tries < maxTries; tries++ {
		err := func() error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return c.removePod(ctx, podId)
		}()
		if err == nil {
			return nil
		}
		if strings.Contains(err.Error(), "code = NotFound") {
			return nil
		}
		dur := interval * time.Duration(tries+1)
		log.Warningf("try to remove pod %s after %s: %v", podId, dur, err)
		errs = append(errs, errors.Wrapf(err, "try %d", tries))
		time.Sleep(dur)
	}
	return errors.NewAggregate(errs)
}

func (c crictl) removePod(ctx context.Context, podId string) error {
	if _, err := c.GetRuntimeClient().RemovePodSandbox(ctx, &runtimeapi.RemovePodSandboxRequest{
		PodSandboxId: podId,
	}); err != nil {
		return errors.Wrap(err, "RemovePodSandbox")
	}
	return nil
}

func (c crictl) stopContainerWithRetry(ctx context.Context, ctrId string, timeout int64, maxTries int) error {
	interval := 5 * time.Second
	errs := []error{}
	for tries := 0; tries < maxTries; tries++ {
		err := func() error {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return c.stopContainer(ctx, ctrId, timeout)
		}()
		if err == nil {
			return nil
		}
		if strings.Contains(err.Error(), "code = NotFound") {
			return nil
		}
		dur := interval * time.Duration(tries+1)
		log.Warningf("try to restop container %s after %s, timeout: %d: %v", ctrId, dur, timeout, err)
		// set timeout to 0 to stop forcely
		timeout = 0
		errs = append(errs, errors.Wrapf(err, "try %d", tries))
		time.Sleep(dur)
	}
	return errors.NewAggregate(errs)
}

func (c crictl) StopContainer(ctx context.Context, ctrId string, timeout int64, tryRemove bool, force bool) error {
	errs := []error{}
	isStopped := false
	if force {
		if err := c.forceKillContainer(ctx, ctrId); err != nil {
			log.Infof("force kill container %s error: %v", ctrId, err)
			errs = append(errs, errors.Wrap(err, "forceKillContainer"))
		} else {
			isStopped = true
		}
	} else {
		maxTries := 5
		if err := c.stopContainerWithRetry(ctx, ctrId, timeout, maxTries); err != nil {
			errs = append(errs, errors.Wrap(err, "stopContainer"))
		} else {
			isStopped = true
		}
	}
	if tryRemove {
		// try force remove container
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := c.RemoveContainer(ctx, ctrId); err != nil {
			errs = append(errs, errors.Wrapf(err, "try remove container %s", ctrId))
		} else {
			return nil
		}
	}
	if isStopped {
		return nil
	}
	return errors.NewAggregate(errs)
}

func (c crictl) stopContainer(ctx context.Context, ctrId string, timeout int64) error {
	if _, err := c.GetRuntimeClient().StopContainer(ctx, &runtimeapi.StopContainerRequest{
		ContainerId: ctrId,
		Timeout:     timeout,
	}); err != nil {
		return errors.Wrap(err, "StopContainer")
	}
	return nil
}

func (c crictl) forceKillContainer(ctx context.Context, ctrId string) error {
	cs, err := c.containerStatus(ctx, ctrId, true)
	if err != nil {
		return errors.Wrap(err, "get containerStatus")
	}
	info := cs.GetInfo()
	infoStr := info["info"]
	if infoStr == "" {
		return errors.Errorf("empty info: %s", infoStr)
	}
	infoObj, err := jsonutils.ParseString(infoStr)
	if err != nil {
		return errors.Wrapf(err, "invalid info: %s", infoStr)
	}
	pidInt, err := infoObj.Int("pid")
	if err != nil {
		return errors.Wrapf(err, "get pid from %s", infoObj)
	}
	pid := fmt.Sprintf("%d", pidInt)
	// get ppid
	pStatusFile := filepath.Join("/proc", pid, "task", pid, "status")
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", fmt.Sprintf("cat %s | grep PPid: | awk '{print $2}'", pStatusFile)).Output()
	if err != nil {
		return errors.Wrapf(err, "get ppid from %s, out: %s", pStatusFile, out)
	}
	ppidStr := strings.TrimSpace(string(out))
	ppid, err := strconv.Atoi(ppidStr)
	if err != nil {
		return errors.Wrapf(err, "invalid ppid str %s from %s", ppidStr, pStatusFile)
	}
	ppCmdlineFile := filepath.Join("/proc", ppidStr, "cmdline")
	ppCmdline, err := procutils.NewRemoteCommandAsFarAsPossible("cat", ppCmdlineFile).Output()
	if err != nil {
		return errors.Wrapf(err, "get cmdline from %s, out: %s", ppCmdlineFile, ppCmdline)
	}
	log.Infof("try to kill container %s, pid %s parent process(%d): %s", ctrId, pid, ppid, ppCmdline)
	killOut, err := procutils.NewRemoteCommandAsFarAsPossible("kill", "-9", ppidStr).Output()
	if err != nil {
		killErr := errors.Wrapf(err, "kill -9 %s, out: %s", ppidStr, killOut)
		log.Errorf("kill container %s, pid %s parent process(%d): %s, error: %v", ctrId, pid, ppid, ppCmdline, killErr)
		return killErr
	}
	if err := c.stopContainerWithRetry(ctx, ctrId, 0, 5); err != nil {
		return errors.Wrapf(err, "stop container %s after kill parent process", ctrId)
	}
	return nil
}

func (c crictl) RemoveContainer(ctx context.Context, ctrId string) error {
	_, err := c.GetRuntimeClient().RemoveContainer(ctx, &runtimeapi.RemoveContainerRequest{
		ContainerId: ctrId,
	})
	if err != nil {
		return errors.Wrap(err, "RemoveContainer")
	}
	return nil
}

func (c crictl) ContainerStatus(ctx context.Context, ctrId string) (*runtimeapi.ContainerStatusResponse, error) {
	return c.containerStatus(ctx, ctrId, false)
}

func (c crictl) containerStatus(ctx context.Context, ctrId string, verbose bool) (*runtimeapi.ContainerStatusResponse, error) {
	req := &runtimeapi.ContainerStatusRequest{
		ContainerId: ctrId,
		Verbose:     verbose,
	}
	return c.GetRuntimeClient().ContainerStatus(ctx, req)
}

func (c crictl) PullImage(ctx context.Context, req *runtimeapi.PullImageRequest) (*runtimeapi.PullImageResponse, error) {
	resp, err := c.GetImageClient().PullImage(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "PullImage")
	}
	return resp, nil
}

func (c crictl) ImageStatus(ctx context.Context, req *runtimeapi.ImageStatusRequest) (*runtimeapi.ImageStatusResponse, error) {
	return c.GetImageClient().ImageStatus(ctx, req)
}

type ExecSyncResponse struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int32
}

func (c crictl) ExecSync(ctx context.Context, ctrId string, command []string, timeout int64) (*ExecSyncResponse, error) {
	cli := c.GetRuntimeClient()
	resp, err := cli.ExecSync(ctx, &runtimeapi.ExecSyncRequest{
		ContainerId: ctrId,
		Cmd:         command,
		Timeout:     timeout,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "exec sync container %s with command: %v", ctrId, command)
	}
	if resp.GetExitCode() != 0 {
		return nil, exec.CodeExitError{
			Err:  errors.Errorf("stdout: %s, stderr: %s, exited: %d", resp.GetStdout(), resp.GetStderr(), resp.GetExitCode()),
			Code: int(resp.ExitCode),
		}
	}
	return &ExecSyncResponse{
		Stdout:   resp.Stdout,
		Stderr:   resp.Stderr,
		ExitCode: resp.ExitCode,
	}, nil
}
