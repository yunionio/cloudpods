package hosttypes

const (
	HOST_TYPE_BAREMETAL  = "baremetal"
	HOST_TYPE_HYPERVISOR = "hypervisor" // KVM
	HOST_TYPE_KVM        = "kvm"
	HOST_TYPE_ESXI       = "esxi"    // # VMWare vSphere ESXi
	HOST_TYPE_KUBELET    = "kubelet" // # Kubernetes Kubelet
	HOST_TYPE_HYPERV     = "hyperv"  // # Microsoft Hyper-V
	HOST_TYPE_XEN        = "xen"     // # XenServer

	HOST_TYPE_ALIYUN    = "aliyun"
	HOST_TYPE_AWS       = "aws"
	HOST_TYPE_QCLOUD    = "qcloud"
	HOST_TYPE_AZURE     = "azure"
	HOST_TYPE_HUAWEI    = "huawei"
	HOST_TYPE_OPENSTACK = "openstack"
)
