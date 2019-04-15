package sysutils

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func setHostname7(name string) error {
	_, err := procutils.NewCommand("hostnamectl", "set-hostname", name).Run()
	return err
}

func setHostname6(name string) error {
	_, err := procutils.NewCommand("hostname", name).Run()
	if err != nil {
		return err
	}
	cont := fmt.Sprintf("NETWORKING=yes\nHOSTNAME=%s\n", name)
	return fileutils2.FileSetContents("/etc/sysconfig/network", cont)
}

func SetHostname(name string) error {
	err := setHostname7(name)
	if err != nil {
		err = setHostname6(name)
	}
	return err
}
