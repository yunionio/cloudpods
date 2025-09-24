package retry

import (
	"math/rand"
	"time"
)

// RandomRetryRule 随机等待时间重试规则
type RandomRetryRule struct {
	minDelay time.Duration // 最小随机等待时间
	maxDelay time.Duration // 最大随机等待时间
}

var DefaultRandomRetryRule = NewRandomRetryRule(DefaultRandomMinDelay, DefaultRandomMaxDelay)

func NewRandomRetryRule(minDelay time.Duration, maxDelay time.Duration) RandomRetryRule {
	if minDelay < 0 {
		minDelay = DefaultRandomMinDelay
	}

	if maxDelay < 0 {
		maxDelay = DefaultRandomMaxDelay
	}

	if maxDelay < minDelay {
		minDelay = DefaultRandomMinDelay
		maxDelay = DefaultRandomMaxDelay
	}

	return RandomRetryRule{
		minDelay: minDelay,
		maxDelay: maxDelay,
	}
}

func (r RandomRetryRule) GetDelay(attempts int) time.Duration {
	delay := r.minDelay + time.Duration(rand.Int63n(int64(r.maxDelay-r.minDelay+1)))
	return delay
}
