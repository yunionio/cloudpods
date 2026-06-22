package llm

import (
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&AiproxyBindings{}), func() gotypes.ISerializable {
		return &AiproxyBindings{}
	})
}

// AiproxyBindings records per-replica aiproxy catalog bindings for a deployment.
type AiproxyBindings []AiproxyInstanceBinding

func (s AiproxyBindings) String() string {
	return jsonutils.Marshal(s).String()
}

func (s AiproxyBindings) IsZero() bool {
	return len(s) == 0
}
