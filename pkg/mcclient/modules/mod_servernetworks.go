package modules

var (
	Servernetworks JointResourceManager
)

func init() {
	Servernetworks = NewJointComputeManager(
		"guestnetwork",
		"guestnetworks",
		[]string{"Guest_ID", "Guest",
			"Network_ID", "Network", "Mac_addr",
			"IP_addr", "Driver", "BW_limit", "Index",
			"Virtual", "Ifname", "team_with"},
		[]string{},
		&Servers,
		&Networks)
	registerCompute(&Servernetworks)
}
