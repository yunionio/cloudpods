package modules

type ParametersManager struct {
	ResourceManager
}

var (
	Parameters ParametersManager
)

func init() {
	Parameters = ParametersManager{NewYunionConfManager("parameter", "parameters",
		[]string{"id", "created_at", "update_at", "name", "value"},
		[]string{"user_id"},
	)}
	register(&Parameters)
}