package consts

var (
	globalOpsLogEnabled = true
)

func DisableOpsLog() {
	globalOpsLogEnabled = false
}

func OpsLogEnabled() bool {
	return globalOpsLogEnabled
}
