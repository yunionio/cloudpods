package apis

import (
	"yunion.io/x/jsonutils"
)

// Meta is embedded by every input or output params
type Meta struct{}

func (m Meta) JSON(self interface{}) *jsonutils.JSONDict {
	return jsonutils.Marshal(self).(*jsonutils.JSONDict)
}
