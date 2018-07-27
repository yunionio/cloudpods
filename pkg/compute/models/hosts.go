package models

const (
	HOST_TYPE_BAREMETAL  = "baremetal"
	HOST_TYPE_HYPERVISOR = "hypervisor" // KVM
	HOST_TYPE_ESXI       = "esxi"       // # VMWare vSphere ESXi
	HOST_TYPE_KUBELET    = "kubelet"    // # Kubernetes Kubelet
	HOST_TYPE_HYPERV     = "hyperv"     // # Microsoft Hyper-V
	HOST_TYPE_XEN        = "xen"        // # XenServer
	HOST_TYPE_ALIYUN     = "aliyun"

	HOST_TYPE_DEFAULT = HOST_TYPE_HYPERVISOR

	// # possible status
	HOST_ONLINE   = "online"
	HOST_ENABLED  = "online"
	HOST_OFFLINE  = "offline"
	HOST_DISABLED = "offline"

	NIC_TYPE_IPMI  = "ipmi"
	NIC_TYPE_ADMIN = "admin"
	// #NIC_TYPE_NORMAL = 'normal'

	HOST_STATUS_INIT           = "init"
	HOST_STATUS_PREPARE        = "prepare"
	HOST_STATUS_PREPARE_FAIL   = "prepare_fail"
	HOST_STATUS_READY          = "ready"
	HOST_STATUS_RUNNING        = "running"
	HOST_STATUS_MAINTAINING    = "maintaining"
	HOST_STATUS_START_MAINTAIN = "start_maintain"
	HOST_STATUS_DELETING       = "deleting"
	HOST_STATUS_DELETE         = "delete"
	HOST_STATUS_DELETE_FAIL    = "delete_fail"
	HOST_STATUS_UNKNOWN        = "unknown"
	HOST_STATUS_SYNCING_STATUS = "syncing_status"
	HOST_STATUS_SYNC           = "sync"
	HOST_STATUS_SYNC_FAIL      = "sync_fail"
	HOST_STATUS_START_CONVERT  = "start_convert"
	HOST_STATUS_CONVERTING     = "converting"
)

var HOST_TYPES = []string{HOST_TYPE_BAREMETAL, HOST_TYPE_HYPERVISOR, HOST_TYPE_ESXI, HOST_TYPE_KUBELET, HOST_TYPE_XEN}
var NIC_TYPES = []string{NIC_TYPE_IPMI, NIC_TYPE_ADMIN}
