package policy

import (
	"time"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
)

func EnableGlobalRbac(refreshInterval time.Duration, retryInterval time.Duration, debug bool) {
	if !consts.IsRbacEnabled() {
		consts.EnableRbac()
		if debug {
			consts.EnableRbacDebug()
		}
		PolicyManager.start(refreshInterval, retryInterval)
	}
}
