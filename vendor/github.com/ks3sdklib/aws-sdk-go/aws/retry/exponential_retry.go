package retry

import (
	"math"
	"time"
)

// ExponentialRetryRule 指数级增长等待时间重试规则
type ExponentialRetryRule struct {
	baseDelay time.Duration // 基础等待时间
	maxDelay  time.Duration // 单次最大等待时间
}

var DefaultExponentialRetryRule = NewExponentialRetryRule(DefaultBaseDelay, DefaultMaxDelay)

func NewExponentialRetryRule(baseDelay time.Duration, maxDelay time.Duration) ExponentialRetryRule {
	if baseDelay < 0 {
		baseDelay = DefaultBaseDelay
	}

	if maxDelay < 0 {
		maxDelay = DefaultMaxDelay
	}

	return ExponentialRetryRule{
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
	}
}

func (r ExponentialRetryRule) GetDelay(attempts int) time.Duration {
	delay := r.baseDelay * time.Duration(math.Pow(2, float64(attempts-1)))
	if delay > r.maxDelay {
		return r.maxDelay
	}
	return delay
}
