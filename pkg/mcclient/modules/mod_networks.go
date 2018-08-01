package modules

var (
	Networks ResourceManager
)

func init() {
	Networks = NewComputeManager("network", "networks",
		[]string{"ID", "Name", "Guest_ip_start",
			"Guest_ip_end", "Guest_ip_mask",
			"wire_id", "wire", "is_public", "exit", "Ports",
			"vnics",
			"group_vnics", "bm_vnics", "reserve_vnics", "server_type",
			"Status"},
		[]string{})

	registerCompute(&Networks)
}
