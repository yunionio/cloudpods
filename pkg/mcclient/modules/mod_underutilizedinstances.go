package modules

var (
	UnderutilizedInstances ResourceManager
)

func init() {
	UnderutilizedInstances = NewCloudmonManager("underutilizedinstance", "underutilizedinstances",
		[]string{"id", "vm_id", "vm_name", "datetime_str", "vm_cpu", "vm_disk", "vm_memory", "vm_provider", "cpu_usage_threshold", "netio_rx_bps_threshold", "netio_tx_bps_threshold", "stastics_details"},
		[]string{})

	register(&UnderutilizedInstances)
}
