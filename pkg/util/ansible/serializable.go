package ansible

import (
	"reflect"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

// String implements gotypes.ISerializable
func (pb *Playbook) String() string {
	return jsonutils.Marshal(pb).String()
}

// IsZero implements gotypes.ISerializable
func (pb *Playbook) IsZero() bool {
	if len(pb.Inventory.Hosts) == 0 {
		return true
	}
	if len(pb.Modules) == 0 {
		return true
	}
	return false
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&Playbook{}), func() gotypes.ISerializable {
		return NewPlaybook()
	})
}
