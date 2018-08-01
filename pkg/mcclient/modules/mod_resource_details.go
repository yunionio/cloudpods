package modules

var (
	ResourceDetails ResourceManager
)

func init() {
	ResourceDetails = NewMeterManager("resource_detail", "resource_details",
		[]string{"res_type", "res_id", "res_name", "start_time", "end_time", "project_name", "user_name", "res_fee"},

		[]string{},
	)
	register(&ResourceDetails)
}
