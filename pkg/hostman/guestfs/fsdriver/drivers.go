package fsdriver

type newRootFsDriverFunc func(part IDiskPartition) IRootFsDriver

var rootfsDrivers = make([]newRootFsDriverFunc, 0)

func GetRootfsDrivers() []newRootFsDriverFunc {
	return rootfsDrivers
}

func init() {
	linuxFsDrivers := []newRootFsDriverFunc{
		NewCentosRootFs, NewFedoraRootFs, NewRhelRootFs,
		NewDebianRootFs, NewCirrosRootFs, NewCirrosNewRootFs, NewUbuntuRootFs,
		//NewGentooRootFs, NewArchLinuxRootFs, NewOpenWrtRootFs, NewCoreOsRootFs,
	}
	rootfsDrivers = append(rootfsDrivers, linuxFsDrivers...)
	//rootfsDrivers = append(rootfsDrivers, NewMacOSRootFs)
	//rootfsDrivers = append(rootfsDrivers, NewEsxiRootFs)
	//rootfsDrivers = append(rootfsDrivers, NewWindowsRootFs)
	//rootfsDrivers = append(rootfsDrivers, NewAndroidRootFs)
}
