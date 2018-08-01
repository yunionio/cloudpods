package modules

var (
	Baremetalnetworks JointResourceManager
)

func init() {
	Baremetalnetworks = NewJointComputeManager(
		"baremetalnetwork",
		"baremetalnetworks",
		[]string{"Baremetal_ID", "Host",
			"Network_ID", "Network", "IP_addr", "Mac_addr",
			"Nic_Type"},
		[]string{},
		&Hosts,
		&Networks)
	registerCompute(&Baremetalnetworks)
}
