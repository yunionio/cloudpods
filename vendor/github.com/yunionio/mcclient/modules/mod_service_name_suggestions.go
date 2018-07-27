package modules

var (
	ServiceNameSuggestion ResourceManager
)

func init() {
	ServiceNameSuggestion = NewMonitorManager("service_name_suggestion", "service_name_suggestions",
		[]string{"title", "content", "create_by", "gmt_create", "is_deleted"},
		[]string{})
	register(&ServiceNameSuggestion)
}
