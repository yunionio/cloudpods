package tasks

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal/sysutils"
	"yunion.io/x/onecloud/pkg/baremetal/types"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type sBaremetalPrepareTask struct {
	baremetal IBaremetal
}

func newBaremetalPrepareTask(baremetal IBaremetal) *sBaremetalPrepareTask {
	return &sBaremetalPrepareTask{
		baremetal: baremetal,
	}
}

func (task *sBaremetalPrepareTask) DoPrepare(cli *ssh.Client) error {
	_, err := cli.Run("/lib/mos/sysinit.sh")
	if err != nil {
		return err
	}

	sysInfo, err := getDMISysinfo(cli)
	if err != nil {
		return err
	}
	cpuInfo, err := getCPUInfo(cli)
	if err != nil {
		return err
	}
	dmiCPUInfo, err := getDMICPUInfo(cli)
	if err != nil {
		return err
	}
	memInfo, err := getDMIMemInfo(cli)
	if err != nil {
		return err
	}
	nicsInfo, err := getNicsInfo(cli)
	if err != nil {
		return err
	}
	// TODO: diskinfo

	ipmiEnable, err := isIPMIEnable(cli)
	if err != nil {
		return err
	}

	if ipmiEnable {
		log.Errorf("TODO: ipmi enable")
	}

	adminNic := task.baremetal.GetAdminNic()

	// collect params
	updateInfo := make(map[string]interface{})
	oname := fmt.Sprintf("BM%s", strings.Replace(adminNic.Mac, ":", "", -1))
	if task.baremetal.GetName() == oname {
		//updateInfo["name"] = fmt.Sprintf("BM-%s", strings.Replace(ipmiInfo.IPAddr, ".", "-", -1))
	}
	updateInfo["access_ip"] = adminNic.IpAddr
	updateInfo["cpu_count"] = cpuInfo.Count
	updateInfo["node_count"] = dmiCPUInfo.Nodes
	updateInfo["cpu_desc"] = cpuInfo.Model
	updateInfo["cpu_mhz"] = cpuInfo.Freq
	updateInfo["cpu_cache"] = cpuInfo.Cache
	updateInfo["sys_info"] = sysInfo
	updateInfo["sn"] = sysInfo.SN

	log.Infof("Parse DMI info: %#v, \ncpuInfo: %#v", sysInfo, cpuInfo)
	return nil
}

func getDMISysinfo(cli *ssh.Client) (*types.DMIInfo, error) {
	ret, err := cli.Run("/usr/sbin/dmidecode -t 1")
	if err != nil {
		return nil, err
	}
	return sysutils.ParseDMISysinfo(ret)
}

func getCPUInfo(cli *ssh.Client) (*types.CPUInfo, error) {
	ret, err := cli.Run("cat /proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	return sysutils.ParseCPUInfo(ret)
}

func getDMICPUInfo(cli *ssh.Client) (*types.DMICPUInfo, error) {
	ret, err := cli.Run("/usr/sbin/dmidecode -t 4")
	if err != nil {
		return nil, err
	}
	return sysutils.ParseDMICPUInfo(ret), nil
}

func getDMIMemInfo(cli *ssh.Client) (*types.DMIMemInfo, error) {
	ret, err := cli.Run("/usr/sbin/dmidecode -t 4")
	if err != nil {
		return nil, err
	}
	return sysutils.ParseDMIMemInfo(ret), nil
}

func getNicsInfo(cli *ssh.Client) ([]*types.NicDevInfo, error) {
	ret, err := cli.Run("/lib/mos/lsnic")
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve NIC info: %v", err)
	}
	return sysutils.ParseNicInfo(ret), nil
}

func isIPMIEnable(cli *ssh.Client) (bool, error) {
	ret, err := cli.Run("/usr/sbin/dmidecode -t 38")
	if err != nil {
		return false, fmt.Errorf("Failed to retrieve IPMI info: %v", err)
	}
	return sysutils.ParseDMIIPMIInfo(ret), nil
}
