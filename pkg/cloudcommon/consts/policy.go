package consts

var (
	globalsRbacEnabled = false
	globalsRbacDebug   = false
)

func EnableRbac() {
	globalsRbacEnabled = true
}

func IsRbacEnabled() bool {
	return globalsRbacEnabled
}

func EnableRbacDebug() {
	globalsRbacDebug = true
}

func IsRbacDebug() bool {
	return globalsRbacDebug
}
