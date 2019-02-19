package influxdb

import (
	"testing"
)

func TestInfluxdb(t *testing.T) {
	url := "https://192.168.222.171:8086"

	db := NewInfluxdb(url)
	err := db.SetDatabase("telegraf1")
	if err != nil {
		t.Fatalf("GetDatabases: %s", err)
	}
	rp := SRetentionPolicy{
		Name:     "30days",
		Duration: "30d",
		ReplicaN: 1,
		Default:  true,
	}
	err = db.SetRetentionPolicy(rp)
	if err != nil {
		t.Fatalf("SetRetentPolicy: %s", err)
	}
	rps, err := db.GetRetentionPolicies()
	if err != nil {
		t.Fatalf("GetRentionPolicies %s", err)
	}
	t.Logf("%#v", rps)
}
