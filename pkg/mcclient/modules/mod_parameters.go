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
		[]string{"namespace", "namespace_id", "created_by", "updated_by"},
	)}
	register(&Parameters)
}