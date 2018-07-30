package models

import (
	"encoding/json"
	"testing"
)

func TestHostNetworkSchedResults(t *testing.T) {
	id := "dd11e175-c2b3-403b-8fed-47a17cd72b79"
	result, err := HostNetworkSchedResults(id)
	if err != nil {
		t.Fatal(err)
	}
	js, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("NetworkSchedResult: %s", string(js))
}
