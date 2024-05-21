package image

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

type ImageTool interface {
	Pull(image string, opt *PullOptions) (string, error)
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
	args = append(reqArgs, args...)
	return procutils.NewRemoteCommandAsFarAsPossible("ctr", args...)
}

type PullOptions struct {
	SkipVerify bool
	PlainHttp  bool
	Username   string
	Password   string
}

func (i imageTool) Pull(image string, opt *PullOptions) (string, error) {
	args := []string{}
	if i.namespace != "" {
		args = append(args, "--namespace", i.namespace)
	}
	args = append(args, []string{"images", "pull"}...)
	if opt.PlainHttp {
		args = append(args, "--plain-http")
	}
	if opt.PlainHttp {
		args = append(args, "--skip-verify")
	}
	if opt.Username != "" && opt.Password != "" {
		args = append(args, "--user", fmt.Sprintf("%s:%s", opt.Username, opt.Password))
	}
	args = append(args, []string{image}...)
	cmd := i.newCtrCmd(args...)
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "pull imageTool: %s", out)
	}
	return image, nil
}
