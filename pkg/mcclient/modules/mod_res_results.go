package modules

var (
	ResResults ResourceManager
)

func init() {
	ResResults = NewMeterManager("res_result", "res_results",
		[]string{"stat_month", "start_date", "end_date", "filter", "project_id"},
		[]string{},
	)
	register(&ResResults)
}
