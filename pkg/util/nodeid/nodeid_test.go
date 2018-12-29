package nodeid

import (
	"testing"
)

func TestGetNodeId(t *testing.T) {
	idstr, err := GetNodeId()
	if err != nil {
		t.Errorf("error %s", err)
	} else {
		t.Logf("ID is %s", idstr)
	}
}
