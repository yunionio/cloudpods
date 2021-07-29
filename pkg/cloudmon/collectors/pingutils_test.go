package collectors

import (
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	result, err := Ping([]string{
		"114.114.114.114",
		"118.187.65.237",
		"10.168.26.254",
		"10.168.26.26",
		"192.30.253.113",
	}, 5, time.Second, true)
	if err != nil {
		// ignore error
		t.Logf("ping error %s", err)
	}
	for k, v := range result {
		t.Logf("%s: %s", k, v.String())
	}
}
