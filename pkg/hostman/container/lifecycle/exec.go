package lifecycle

import (
	"context"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/pod"
)

func init() {
	RegisterDriver(newExec())
}

type execDriver struct{}

func newExec() ILifecycle {
	return &execDriver{}
}

func (e execDriver) GetType() apis.ContainerLifecyleHandlerType {
	return apis.ContainerLifecyleHandlerTypeExec
}

func (e execDriver) Run(ctx context.Context, input *apis.ContainerLifecyleHandler, cri pod.CRI, id string) error {
	cfg := input.Exec
	cli := cri.GetRuntimeClient()
	resp, err := cli.ExecSync(ctx, &runtimeapi.ExecSyncRequest{
		ContainerId: id,
		Cmd:         cfg.Command,
	})
	if err != nil {
		return errors.Wrapf(err, "exec sync")
	}
	if resp.GetExitCode() != 0 {
		return errors.Wrapf(err, "stdout: %s, stderr: %s, exited: %d", resp.GetStdout(), resp.GetStderr(), resp.GetExitCode())
	}
	log.Infof("run command %v: stdout: %s, stderr: %s", cfg.Command, resp.Stdout, resp.Stderr)
	return nil
}
