package system_service

import (
	"strings"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

var (
	SystemdServiceManager IServiceManager = &SSystemdServiceManager{}
)

type SSystemdServiceManager struct {
}

func (manager *SSystemdServiceManager) Detect() bool {
	_, err := procutils.NewCommand("systemctl", "--version").Run()
	return err == nil
}

func (manager *SSystemdServiceManager) Start(srvname string) error {
	_, err := procutils.NewCommand("systemctl", "restart", srvname).Run()
	return err
}

func (manager *SSystemdServiceManager) Enable(srvname string) error {
	_, err := procutils.NewCommand("systemctl", "enable", srvname).Run()
	return err
}

func (manager *SSystemdServiceManager) Stop(srvname string) error {
	_, err := procutils.NewCommand("systemctl", "stop", srvname).Run()
	return err
}

func (manager *SSystemdServiceManager) Disable(srvname string) error {
	_, err := procutils.NewCommand("systemctl", "disable", srvname).Run()
	return err
}

func (manager *SSystemdServiceManager) GetStatus(srvname string) SServiceStatus {
	res, _ := procutils.NewCommand("systemctl", "status", srvname).Run()
	return parseSystemdStatus(string(res), srvname)
}

func parseSystemdStatus(res string, srvname string) SServiceStatus {
	var ret SServiceStatus
	lines := strings.Split(string(res), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			if strings.HasPrefix(line, "Loaded:") {
				parts := strings.Split(line, " ")
				if len(parts) > 1 && parts[1] == "loaded" {
					ret.Loaded = true
				}
			} else if strings.HasPrefix(line, "Active:") {
				parts := strings.Split(line, " ")
				if len(parts) > 1 && parts[1] == "active" {
					ret.Active = true
				}
			}
		}
	}
	return ret
}
