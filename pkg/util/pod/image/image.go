package image

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

type ImageTool interface {
	Pull(image string, opt *PullOptions) (string, error)
	Push(image string, opt *PushOptions) error
}

type imageTool struct {
	address   string
	namespace string
}

func NewImageTool(address, namespace string) ImageTool {
	return &imageTool{
		address:   address,
		namespace: namespace,
	}
}

func (i imageTool) newCtrCmd(args ...string) *procutils.Command {
	reqArgs := []string{"--address", i.address}
	if i.namespace != "" {
		reqArgs = append(reqArgs, "--namespace", i.namespace)
	}
	args = append(reqArgs, args...)
	return procutils.NewRemoteCommandAsFarAsPossible("ctr", args...)
}

type RepoCommonOptions struct {
	SkipVerify bool
	PlainHttp  bool
	Username   string
	Password   string
}

type PullOptions struct {
	RepoCommonOptions
}

func (i imageTool) newRepoCommonArgs(opt RepoCommonOptions) []string {
	args := []string{}
	if opt.PlainHttp {
		args = append(args, "--plain-http")
	}
	if opt.SkipVerify {
		args = append(args, "--skip-verify")
	}
	if opt.Username != "" && opt.Password != "" {
		args = append(args, "--user", fmt.Sprintf("%s:%s", opt.Username, opt.Password))
	}
	return args
}

func (i imageTool) Pull(image string, opt *PullOptions) (string, error) {
	args := []string{}
	args = append(args, []string{"images", "pull"}...)
	args = append(args, i.newRepoCommonArgs(opt.RepoCommonOptions)...)
	args = append(args, []string{image}...)

	cmd := i.newCtrCmd(args...)
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "pull imageTool: %s", out)
	}
	return image, nil
}

type PushOptions struct {
	RepoCommonOptions
}

func (i imageTool) Push(image string, opt *PushOptions) error {
	args := []string{}
	args = append(args, []string{"images", "push"}...)
	args = append(args, i.newRepoCommonArgs(opt.RepoCommonOptions)...)
	args = append(args, []string{image}...)

	cmd := i.newCtrCmd(args...)
	out, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(err, "push %s: %s", image, out)
	}
	return nil
}
