package compute

import "yunion.io/x/onecloud/pkg/apis"

type SDBInstanceCreateInput struct {
	apis.Meta

	Name              string
	Description       string
	DisableDelete     *bool
	NetworkId         string
	Address           string
	MasterInstanceId  string
	SecgroupId        string
	Zone1             string
	Zone2             string
	Zone3             string
	ZoneId            string
	CloudregionId     string
	Cloudregion       string
	VpcId             string
	ManagerId         string
	NetworkExternalId string
	BillingType       string
	BillingCycle      string
	InstanceType      string
	Engine            string
	EngineVersion     string
	Category          string
	StorageType       string
	DiskSizeGB        int
	Password          string

	VcpuCount  int
	VmemSizeMb int
	Provider   string
}

type SDBInstanceChangeConfigInput struct {
	apis.Meta

	InstanceType string
	VCpuCount    int
	VmemSizeMb   int
	StorageType  string
	DiskSizeGB   int
	Category     string
}

type SDBInstanceRecoveryConfigInput struct {
	apis.Meta

	DBInstancebackup   string
	DBInstancebackupId string            `json:"dbinstancebackup_id"`
	Databases          map[string]string `json:"allowempty"`
}
