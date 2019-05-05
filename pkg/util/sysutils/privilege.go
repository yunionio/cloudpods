package sysutils

import (
	"strconv"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

func IsRootPermission() (bool, error) {
	output, err := procutils.NewCommand("id", "-u").Run()
	if err != nil {
		return false, err
	}
	i, err := strconv.Atoi(string(output[:len(output)-1]))
	if err != nil {
		return false, err
	}
	if i == 0 {
		return true, nil
	} else {
		return false, nil
	}
}
