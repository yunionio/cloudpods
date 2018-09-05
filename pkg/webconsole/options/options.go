package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
)

var (
	Options WebConsoleOptions
)

type WebConsoleOptions struct {
	cloudcommon.Options

	ApiServer string `help:"API server url to handle websocket connection, usually with public access" default:"http://webconsole.yunion.io"`
}
