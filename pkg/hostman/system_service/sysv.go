package system_service

import (
	"strings"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

var (
	SysVServiceManager IServiceManager = &SSysVServiceManager{}
)

type SSysVServiceManager struct {
}

func (manager *SSysVServiceManager) Detect() bool {
	_, err := procutils.NewCommand("chkconfig", "--version").Run()
	return err == nil
}

func (manager *SSysVServiceManager) Start(srvname string) error {
	_, err := procutils.NewCommand("service", srvname, "restart").Run()
	return err
}

func (manager *SSysVServiceManager) Enable(srvname string) error {
	_, err := procutils.NewCommand("chkconfig", srvname, "on").Run()
	return err
}

func (manager *SSysVServiceManager) Stop(srvname string) error {
	_, err := procutils.NewCommand("service", srvname, "stop").Run()
	return err
}

func (manager *SSysVServiceManager) Disable(srvname string) error {
	_, err := procutils.NewCommand("chkconfig", srvname, "off").Run()
	return err
}

func (manager *SSysVServiceManager) GetStatus(srvname string) SServiceStatus {
	res, _ := procutils.NewCommand("chkconfig", "--list", srvname).Run()
	res2, _ := procutils.NewCommand("service", srvname, "status").Run()
	return parseSysvStatus(string(res), string(res2), srvname)
}

func parseSysvStatus(res string, res2 string, srvname string) SServiceStatus {
	var ret SServiceStatus
	lines := strings.Split(res, "\n")
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) > 1 && strings.TrimSpace(parts[0]) == srvname {
			ret.Loaded = true
			if strings.Index(res2, "running") > 0 {
				ret.Active = true
			}
			break
		}
	}
	return ret
}
