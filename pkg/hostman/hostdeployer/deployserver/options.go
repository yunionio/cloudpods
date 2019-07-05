package deployserver

import "yunion.io/x/onecloud/pkg/cloudcommon/options"

type SDeployOptions struct {
	options.BaseOptions

	DeployServerSocketPath string   `help:"Deploy server listen socket path"`
	PrivatePrefixes        []string `help:"IPv4 private prefixes"`
}

var DeployOption SDeployOptions
