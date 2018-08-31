package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
)

var (
	Options WebConsoleOptions
)

type WebConsoleOptions struct {
	cloudcommon.Options

	FrontendUrl   string `help:"Frontend url to display web console page" default:"http://127.0.0.1:8899"`
	TtyStaticPath string `help:"TTY static HTML render pages" default:"./tty"`
}
