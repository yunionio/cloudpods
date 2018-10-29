package policy

import (
	"time"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
)

func EnableGlobalRbac(refreshInterval time.Duration, retryInterval time.Duration) {
	consts.EnableRbac()
	PolicyManager.start(refreshInterval, retryInterval)
}
