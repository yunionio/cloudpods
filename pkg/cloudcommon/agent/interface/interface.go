package _interface

import (
	"net"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type IAgent interface {
	GetAgentType() string
	GetAccessIP() (net.IP, error)
	GetListenIP() (net.IP, error)
	GetPort() int
	GetEnableSsl() bool
	GetZoneName() string
	GetAdminSession() *mcclient.ClientSession
	TuneSystem() error
	StartService() error
	StopService() error
}
