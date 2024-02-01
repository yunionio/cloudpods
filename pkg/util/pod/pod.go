package pod

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/jsonutils"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type CRI interface {
	Version(ctx context.Context) (*runtimeapi.VersionResponse, error)
	ListPods(ctx context.Context, opts ListPodOptions) ([]*runtimeapi.PodSandbox, error)
	RunPod(ctx context.Context, podConfig *runtimeapi.PodSandboxConfig, runtimeHandler string) (string, error)
	StopPod(ctx context.Context, req *runtimeapi.StopPodSandboxRequest) error
	RemovePod(ctx context.Context, podId string) error
	CreateContainer(ctx context.Context, podId string, podConfig *runtimeapi.PodSandboxConfig, ctrConfig *runtimeapi.ContainerConfig, withPull bool) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, ctrId string, timeout int64) error
	RemoveContainer(ctx context.Context, ctrId string) error
	RunContainers(ctx context.Context, podConfig *runtimeapi.PodSandboxConfig, containerConfigs []*runtimeapi.ContainerConfig, runtimeHandler string) (*RunContainersResponse, error)
	ListContainers(ctx context.Context, opts ListContainerOptions) ([]*runtimeapi.Container, error)
	ContainerStatus(ctx context.Context, ctrId string) (*runtimeapi.ContainerStatusResponse, error)
	ListImages(ctx context.Context, filter *runtimeapi.ImageFilter) ([]*runtimeapi.Image, error)
	PullImage(ctx context.Context, req *runtimeapi.PullImageRequest) (*runtimeapi.PullImageResponse, error)
	ImageStatus(ctx context.Context, req *runtimeapi.ImageStatusRequest) (*runtimeapi.ImageStatusResponse, error)

	// lower layer client
	// getImageClient() runtimeapi.ImageServiceClient
	// getRuntimeClient() runtimeapi.RuntimeServiceClient
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

func (c crictl) getImageClient() runtimeapi.ImageServiceClient {
	return c.imgCli
}

func (c crictl) getRuntimeClient() runtimeapi.RuntimeServiceClient {
	return c.runCli
}

func (c crictl) Version(ctx context.Context) (*runtimeapi.VersionResponse, error) {
	return c.getRuntimeClient().Version(ctx, &runtimeapi.VersionRequest{})
}

func (c crictl) ListImages(ctx context.Context, filter *runtimeapi.ImageFilter) ([]*runtimeapi.Image, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.getImageClient().ListImages(ctx, &runtimeapi.ListImagesRequest{
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
	r, err := c.getRuntimeClient().RunPodSandbox(ctx, req)
	if err != nil {
		return "", errors.Wrapf(err, "RunPod with request: %s", req.String())
	}
	return r.GetPodSandboxId(), nil
}

func (c crictl) StopPod(ctx context.Context, req *runtimeapi.StopPodSandboxRequest) error {
	_, err := c.getRuntimeClient().StopPodSandbox(ctx, req)
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
	r, err := c.getImageClient().PullImage(ctx, req)
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

	log.Infof("======container config: %s", jsonutils.Marshal(ctrConfig).PrettyString())

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

	log.Infof("CreateContainerRequest: %v", req)
	r, err := c.getRuntimeClient().CreateContainer(ctx, req)
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
	if _, err := c.getRuntimeClient().StartContainer(ctx, &runtimeapi.StartContainerRequest{
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
	r, err := c.getRuntimeClient().ListContainers(ctx, req)
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
	ret, err := c.getRuntimeClient().ListPodSandbox(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "ListPodSandbox")
	}
	return ret.Items, nil
}

func (c crictl) RemovePod(ctx context.Context, podId string) error {
	if _, err := c.getRuntimeClient().RemovePodSandbox(ctx, &runtimeapi.RemovePodSandboxRequest{
		PodSandboxId: podId,
	}); err != nil {
		return errors.Wrap(err, "RemovePodSandbox")
	}
	return nil
}

func (c crictl) StopContainer(ctx context.Context, ctrId string, timeout int64) error {
	if _, err := c.getRuntimeClient().StopContainer(ctx, &runtimeapi.StopContainerRequest{
		ContainerId: ctrId,
		Timeout:     timeout,
	}); err != nil {
		return errors.Wrap(err, "StopContainer")
	}
	return nil
}

func (c crictl) RemoveContainer(ctx context.Context, ctrId string) error {
	_, err := c.getRuntimeClient().RemoveContainer(ctx, &runtimeapi.RemoveContainerRequest{
		ContainerId: ctrId,
	})
	if err != nil {
		return errors.Wrap(err, "RemoveContainer")
	}
	return nil
}

func (c crictl) ContainerStatus(ctx context.Context, ctrId string) (*runtimeapi.ContainerStatusResponse, error) {
	req := &runtimeapi.ContainerStatusRequest{
		ContainerId: ctrId,
		Verbose:     false,
	}
	return c.getRuntimeClient().ContainerStatus(ctx, req)
}

func (c crictl) PullImage(ctx context.Context, req *runtimeapi.PullImageRequest) (*runtimeapi.PullImageResponse, error) {
	resp, err := c.getImageClient().PullImage(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "PullImage")
	}
	return resp, nil
}

func (c crictl) ImageStatus(ctx context.Context, req *runtimeapi.ImageStatusRequest) (*runtimeapi.ImageStatusResponse, error) {
	return c.getImageClient().ImageStatus(ctx, req)
}
