package consts

var (
	globalsRbacEnabled = false
)

func EnableRbac() {
	globalsRbacEnabled = true
}

func IsRbacEnabled() bool {
	return globalsRbacEnabled
}
