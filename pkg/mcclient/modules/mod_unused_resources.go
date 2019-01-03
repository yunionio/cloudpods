package modules

var (
	UnusedResources ResourceManager
)

func init() {
	UnusedResources = NewMeterManager("unused_resource", "unused_resources",
		[]string{"res_id", "res_name", "res_type", "start_time", "end_time", "project_name", "spec",
			"platform", "action", "quantity", "medium_type", "storage_type", "event_id"},
		[]string{},
	)
	register(&UnusedResources)
}
