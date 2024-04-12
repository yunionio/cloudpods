// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package esxi

type TOSType string

type TOSArch string

const (
	LINUX   = TOSType("Linux")
	WINDOWS = TOSType("Windows")
	MACOS   = TOSType("macOS")
	FREEBSD = TOSType("FreeBSD")
	SOLARIS = TOSType("Solaris")
	VMWARE  = TOSType("VMware")

	X86    = TOSArch("x86")
	X86_64 = TOSArch("x86_64")
)

type SOsInfo struct {
	OsType         TOSType
	OsDistribution string
	OsVersion      string
	OsArch         TOSArch
}

func asian(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "Asianux Server", ver, arch}
}

func centos4_5(arch TOSArch) SOsInfo {
	return centos("4/5", arch)
}

func centos(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "CentOS", ver, arch}
}

func coreos(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "CoreOS Linux", ver, arch}
}

func macos(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{MACOS, "Mac OS", ver, arch}
}

func debian(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "Debian", ver, arch}
}

func eComStationGuest(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{"", "eComStationGuest", ver, arch}
}

func fedora(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "Fedora", ver, arch}
}

func freebsd(arch TOSArch) SOsInfo {
	return SOsInfo{FREEBSD, "FreeBSD", "", arch}
}

func rhel(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "RedHat Enterprise Linux", ver, arch}

}

func suse(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "SuSE", ver, arch}
}

func opensuse(arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "OpenSuSE", "?", arch}

}

func oracle(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "Oracle", ver, arch}
}

func linux(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "Generic", ver, arch}
}

func windows(dist string, arch TOSArch) SOsInfo {
	return SOsInfo{WINDOWS, dist, "", arch}
}

func solaris(ver string, arch TOSArch) SOsInfo {
	return SOsInfo{SOLARIS, "Solaris", ver, arch}
}

func turbo(arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "Turbo Linux", "?", arch}
}

func ubuntu(arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "Ubuntu", "?", arch}
}

func mandriva(arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "Mandriva", "?", arch}
}

func mandrake(arch TOSArch) SOsInfo {
	return SOsInfo{LINUX, "Mandrake", "?", arch}
}

func vmware(ver string) SOsInfo {
	return SOsInfo{VMWARE, "ESX", ver, X86_64}
}

var (
	// https://vdc-repo.vmware.com/vmwb-repository/dcr-public/da47f910-60ac-438b-8b9b-6122f4d14524/16b7274a-bf8b-4b4c-a05e-746f2aa93c8c/doc/vim.vm.GuestOsDescriptor.GuestOsIdentifier.html
	GuestOsInfo = map[string]SOsInfo{
		"asianux3_64Guest":  asian("3", X86_64),
		"asianux3Guest":     asian("3", X86),
		"asianux4_64Guest":  asian("4", X86_64),
		"asianux4Guest":     asian("4", X86),
		"asianux5_64Guest":  asian("5", X86_64),
		"asianux7_64Guest":  asian("7", X86_64),
		"centos6_64Guest":   centos("6", X86_64),
		"centos64Guest":     centos4_5(X86_64),
		"centos6Guest":      centos("6", X86),
		"centos7_64Guest":   centos("7", X86_64),
		"centos7Guest":      centos("7", X86),
		"centos8_64Guest":   centos("8", X86),
		"centosGuest":       centos4_5(X86),
		"coreos64Guest":     coreos("", X86_64),
		"darwin10_64Guest":  macos("10.6", X86_64),
		"darwin10Guest":     macos("10.6", X86),
		"darwin11_64Guest":  macos("10.7", X86_64),
		"darwin11Guest":     macos("10.7", X86),
		"darwin12_64Guest":  macos("10.8", X86_64),
		"darwin13_64Guest":  macos("10.9", X86_64),
		"darwin14_64Guest":  macos("10.10", X86_64),
		"darwin15_64Guest":  macos("10.11", X86_64),
		"darwin16_64Guest":  macos("10.12", X86_64),
		"darwin17_64Guest":  macos("10.13", X86_64),
		"darwin18_64Guest":  macos("10.14", X86_64),
		"darwin64Guest":     macos("10.5", X86_64),
		"darwinGuest":       macos("10.5", X86),
		"debian10_64Guest":  debian("10", X86_64),
		"debian10Guest":     debian("10", X86),
		"debian4_64Guest":   debian("4", X86_64),
		"debian4Guest":      debian("4", X86),
		"debian5_64Guest":   debian("5", X86_64),
		"debian5Guest":      debian("5", X86),
		"debian6_64Guest":   debian("6", X86_64),
		"debian6Guest":      debian("6", X86),
		"debian7_64Guest":   debian("7", X86_64),
		"debian7Guest":      debian("7", X86),
		"debian8_64Guest":   debian("8", X86_64),
		"debian8Guest":      debian("8", X86),
		"debian9_64Guest":   debian("9", X86_64),
		"debian9Guest":      debian("9", X86),
		"eComStation2Guest": eComStationGuest("2.0", X86),
		"eComStationGuest":  eComStationGuest("1.x", X86),
		"fedora64Guest":     fedora("?", X86_64),
		"fedoraGuest":       fedora("?", X86),
		"freebsd64Guest":    freebsd(X86_64),
		"freebsdGuest":      freebsd(X86),
		//"genericLinuxGuest":       linux("?", X86),
		"mandrakeGuest":        mandrake(X86),
		"mandriva64Guest":      mandriva(X86_64),
		"mandrivaGuest":        mandriva(X86),
		"opensuse64Guest":      opensuse(X86_64),
		"opensuseGuest":        opensuse(X86),
		"oracleLinux6_64Guest": oracle("6", X86_64),
		"oracleLinux64Guest":   oracle("4/5", X86_64),
		"oracleLinux6Guest":    oracle("6", X86),
		"oracleLinux7_64Guest": oracle("7", X86_64),
		"oracleLinux7Guest":    oracle("7", X86),
		"oracleLinuxGuest":     oracle("4/5", X86),
		//"other24xLinux64Guest":    linux("2.4", X86_64),
		//"other24xLinuxGuest":      linux("2.4", X86),
		//"other26xLinux64Guest":    linux("2.6", X86_64),
		//"other26xLinuxGuest":      linux("2.6", X86),
		//"other3xLinux64Guest":     linux("3.x", X86_64),
		//"other3xLinuxGuest":       linux("3.x", X86_64),
		//"otherLinux64Guest":       linux("2.2", X86_64),
		//"otherLinuxGuest":         linux("2.2", X86),
		"redhatGuest":             rhel("2.1", X86),
		"rhel2Guest":              rhel("2", X86),
		"rhel3_64Guest":           rhel("3", X86_64),
		"rhel3Guest":              rhel("3", X86),
		"rhel4_64Guest":           rhel("4", X86_64),
		"rhel4Guest":              rhel("4", X86),
		"rhel5_64Guest":           rhel("5", X86_64),
		"rhel5Guest":              rhel("5", X86),
		"rhel6_64Guest":           rhel("6", X86_64),
		"rhel6Guest":              rhel("6", X86),
		"rhel7_64Guest":           rhel("7", X86_64),
		"rhel7Guest":              rhel("7", X86),
		"sles10_64Guest":          suse("10", X86_64),
		"sles10Guest":             suse("10", X86),
		"sles11_64Guest":          suse("11", X86_64),
		"sles11Guest":             suse("11", X86),
		"sles12_64Guest":          suse("12", X86_64),
		"sles12Guest":             suse("12", X86),
		"sles64Guest":             suse("9", X86_64),
		"slesGuest":               suse("9", X86),
		"solaris10_64Guest":       solaris("10", X86_64),
		"solaris10Guest":          solaris("10", X86),
		"solaris11_64Guest":       solaris("11", X86_64),
		"solaris6Guest":           solaris("6", X86),
		"solaris7Guest":           solaris("7", X86),
		"solaris8Guest":           solaris("8", X86),
		"solaris9Guest":           solaris("9", X86),
		"suse64Guest":             suse("?", X86_64),
		"suseGuest":               suse("?", X86),
		"turboLinux64Guest":       turbo(X86_64),
		"turboLinuxGuest":         turbo(X86),
		"ubuntu64Guest":           ubuntu(X86_64),
		"ubuntuGuest":             ubuntu(X86),
		"vmkernel5Guest":          vmware("5"),
		"vmkernel65Guest":         vmware("6.5"),
		"vmkernel6Guest":          vmware("6"),
		"vmkernelGuest":           vmware("4"),
		"win2000AdvServGuest":     windows("Windows 2000 Advanced Server", X86),
		"win2000ProGuest":         windows("Windows 2000 Professional", X86),
		"win2000ServGuest":        windows("Windows 2000 Server", X86),
		"win31Guest":              windows("Windows 3.1", X86),
		"win95Guest":              windows("Windows 95", X86),
		"win98Guest":              windows("Windows 98", X86),
		"windows7_64Guest":        windows("Windows 7", X86_64),
		"windows7Guest":           windows("Windows 7", X86_64),
		"windows7Server64Guest":   windows("Windows Server 2008 R2", X86_64),
		"windows8_64Guest":        windows("Windows 8", X86_64),
		"windows8Guest":           windows("Windows 8", X86),
		"windows8Server64Guest":   windows("Windows 8 Server", X86_64),
		"windows9_64Guest":        windows("Windows 10", X86_64),
		"windows9Guest":           windows("Windows 10", X86),
		"windows9Server64Guest":   windows("Windows 10 Server", X86_64),
		"windowsHyperVGuest":      windows("Windows Hyper-V", X86_64),
		"winLonghorn64Guest":      windows("Windows Longhorn", X86_64),
		"winLonghornGuest":        windows("Windows Longhorn", X86),
		"winMeGuest":              windows("Windows Millenium Edition", X86),
		"winNetBusinessGuest":     windows("Windows Small Business Server 2003", X86),
		"winNetDatacenter64Guest": windows("Windows Server 2003 Datacenter Edition", X86_64),
		"winNetDatacenterGuest":   windows("Windows Server 2003 Datacenter Edition", X86),
		"winNetEnterprise64Guest": windows("Windows Server 2003 Enterprise Edition", X86_64),
		"winNetEnterpriseGuest":   windows("Windows Server 2003 Enterprise Edition", X86),
		"winNetStandard64Guest":   windows("Windows Server 2003 Standard Edition", X86_64),
		"winNetStandardGuest":     windows("Windows Server 2003 Standard Edition", X86),
		"winNetWebGuest":          windows("Windows Server 2003 Web Edition", X86),
		"winNTGuest":              windows("Windows NT 4", X86),
		"winVista64Guest":         windows("Windows Vista", X86_64),
		"winVistaGuest":           windows("Windows Vista", X86),
		"winXPHomeGuest":          windows("Windows XP Home Edition", X86),
		"winXPPro64Guest":         windows("Windows XP Professional Edition", X86_64),
		"winXPProGuest":           windows("Windows XP Professional", X86),
	}
)
