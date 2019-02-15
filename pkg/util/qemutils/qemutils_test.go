package qemutils

import (
	"testing"

	"yunion.io/x/log"
)

func TestGetQemu(t *testing.T) {
	log.Infof("default: %s", GetQemu(""))
	log.Infof("2.9.1: %s", GetQemu("2.9.1"))
	log.Infof("2.12.1: %s", GetQemu("2.12.1"))
	log.Infof("2.12.2: %s", GetQemu("2.12.2"))
	log.Infof(GetQemuImg())
	log.Infof(GetQemuNbd())
}
