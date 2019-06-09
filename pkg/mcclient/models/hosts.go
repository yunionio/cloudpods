package models

import (
	"time"

	"yunion.io/x/jsonutils"
)

type Host struct {
	EnabledStatusStandaloneResourceBase

	Rack  string
	Slots string

	AccessMac  string
	AccessIp   string
	ManagerUri string

	SysInfo jsonutils.JSONObject
	SN      string

	CpuCount    int
	NodeCount   int8
	CpuDesc     string
	CpuMhz      int
	CpuCache    int
	CpuReserved int
	CpuCmtbound float32

	MemSize     int
	MemReserved int
	MemCmtbound float32

	StorageSize   int
	StorageType   string
	StorageDriver string
	StorageInfo   jsonutils.JSONObject

	IpmiInfo jsonutils.JSONObject

	HostStatus string

	ZoneId string

	HostType string

	Version string

	IsBaremetal bool

	IsMaintenance bool

	LastPingAt time.Time

	ResourceType string

	RealExternalId string

	IsImport bool
}
