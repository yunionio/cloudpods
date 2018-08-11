package models

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"

	o "yunion.io/x/onecloud/cmd/scheduler/options"
	"yunion.io/x/onecloud/pkg/scheduler/api"
)

const (
	HostResourceName = "host"
)

var (
	HostExtraFeature = []string{"nest", "storage_type", "vip_reserved"}
)

type Host struct {
	StandaloneModel

	Rack       string `json:"rack,omitempty" gorm:"column:rack"`
	Slots      string `json:"slots,omitempty" gorm:"column:slots"`
	AccessMAC  string `json:"access_mac" gorm:"not null"`
	AccessIP   string `json:"access_ip" gorm:"column:access_ip"`
	ManagerURI string `json:"manager_uri,omitempty" gorm:"column:manager_uri"`
	SysInfo    string `json:"sys_info,omitempty" gorm:"type:text"`
	Sn         string `json:"sn,omitempty" gorm:"column:sn"`

	CPUCount    int64    `json:"cpu_count" gorm:"column:cpu_count"`
	NodeCount   int64    `json:"node_count" gorm:"column:node_count"`
	CPUDesc     string   `json:"cpu_desc" gorm:"column:cpu_desc"`
	CPUMHZ      int64    `json:"cpu_mhz" gorm:"column:cpu_mhz"`
	CPUCache    int64    `json:"cpu_cache" gorm:"column:cpu_cache"`
	CPUReserved int64    `json:"cpu_reserved" gorm:"column:cpu_reserved"`
	CPUCmtbound *float64 `json:"cpu_cmtbound" gorm:"column:cpu_cmtbound"`

	MemSize     int64    `json:"mem_size" gorm:"column:mem_size"`
	MemReserved int64    `json:"mem_reserved" gorm:"column:mem_reserved"`
	MemCmtbound *float64 `json:"mem_cmtbound" gorm:"column:mem_cmtbound"`

	StorageSize   int    `json:"storage_size,omitempty" gorm:"column:storage_size"`
	StorageType   string `json:"storage_type,omitempty" gorm:"column:storage_type"`
	StorageDriver string `json:"storage_driver,omitempty" gorm:"column:storage_driver"`
	StorageInfo   string `json:"storage_info,omitempty" gorm:"column:storage_info"`
	IpmiInfo      string `json:"ipmi_info,omitempty" gorm:"type:text"`

	Status        string  `json:"status" gorm:"column:status;not null"`
	HostStatus    string  `json:"host_status" gorm:"column:host_status;not null"`
	Enabled       bool    `json:"enabled" gorm:"column:enabled;not null"`
	ZoneID        string  `json:"zone_id" gorm:"column:zone_id;not null"`
	HostType      string  `json:"host_type" gorm:"column:host_type"`
	Version       string  `json:"version" gorm:"column:version"`
	IsBaremetal   bool    `json:"is_baremetal" gorm:"column:is_baremetal"`
	ManagerID     *string `json:"manager_id" gorm:"column:manager_id"`
	IsMaintenance bool    `json:"is_maintenance" gorm:"column:is_maintenance"`

	// DECAPITATE
	ClusterID string `json:"cluster_id" gorm:"column:cluster_id"`
	PoolID    string `json:"pool_id,omitempty" gorm:"column:pool_id"`
}

func (h Host) TableName() string {
	return hostsTable
}

func (h Host) String() string {
	s, _ := JsonString(h)
	return string(s)
}

func (h Host) IsHypervisor() bool {
	if h.HostType == api.HostTypeBaremetal {
		return false
	}
	return true
}

func NewHostResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Host{}
	}
	models := func() interface{} {
		hosts := []Host{}
		return &hosts
	}

	return newResource(db, hostsTable, model, models)
}

func (h Host) CPUOverCommitBound() float64 {
	if h.CPUCmtbound != nil {
		return *h.CPUCmtbound
	}
	return float64(o.GetOptions().DefaultCpuOvercommitBound)
}

func (h Host) MemOverCommitBound() float64 {
	if h.MemCmtbound != nil {
		return *h.MemCmtbound
	}
	return float64(o.GetOptions().DefaultMemoryOvercommitBound)
}

func HostAggregates(hostID string) ([]*Aggregate, error) {
	hAggs, err := FetchByHostIDs(AggregateHosts, []string{hostID})
	if err != nil {
		return nil, err
	}
	aggs := make([]*Aggregate, 0)
	for _, obj := range hAggs {
		ha := obj.(*AggregateHost)
		agg, err := ha.Aggregate()
		if err != nil {
			return nil, err
		}
		aggs = append(aggs, agg)
	}
	return aggs, nil
}

type ResidentTenant struct {
	HostID      string `json:"host_id" gorm:"column:host_id;not null"`
	TenantID    string `json:"tenant_id" gorm:"column:tenant_id;not null"`
	TenantCount int64  `json:"tenant_count" gorm:"column:tenant_count"`
}

func (t ResidentTenant) First() string {
	return t.HostID
}
func (t ResidentTenant) Second() string {
	return t.TenantID
}

func (t ResidentTenant) Third() interface{} {
	return t.TenantCount
}

func ResidentTenantsInHosts(hostIDs []string) ([]ResidentTenant, error) {
	tenants := []ResidentTenant{}
	err := Guests.DB().Table(guestsTable).
		Select("host_id, tenant_id, count(tenant_id) as tenant_count").
		Where(fmt.Sprintf("host_id in ('%s') and deleted=0", strings.Join(hostIDs, "','"))).
		Group("tenant_id, host_id").Scan(&tenants).Error
	return tenants, err
}

func FetchHypervisorHostByIDs(ids []string) ([]interface{}, error) {
	rows, err := rowsNotDeletedInWithCond(Hosts, "id", ids,
		map[string]interface{}{
			"host_type!": "baremetal",
		})
	if err != nil {
		return nil, err
	}
	return rowsToArray(Hosts, rows)
}

func FetchBaremetalHostByIDs(ids []string) ([]interface{}, error) {
	rows, err := rowsNotDeletedInWithCond(Hosts, "id", ids,
		map[string]interface{}{
			"host_type": "baremetal",
		})
	if err != nil {
		return nil, err
	}
	return rowsToArray(Hosts, rows)
}
