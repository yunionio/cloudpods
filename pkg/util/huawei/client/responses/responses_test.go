package responses

import (
	"fmt"
	"testing"
	"yunion.io/x/jsonutils"
)

func TestTransColonToDot(t *testing.T) {
	raw := `{"A:b::C": "1:2:3", "A": true, "B": ["1:2", ":", "c"], "D:E": true}`
	obj, err := jsonutils.ParseString(raw)
	if err != nil {
		fmt.Println(err)
	}
	nobj, err := TransColonToDot(obj)
	if err != nil {
		fmt.Println(err)
	}

	if nobj.String() != `{"A":true,"A.b..C":"1:2:3","B":["1:2",":","c"],"D.E":true}` {
		t.Errorf("trans failed")
	}
}
