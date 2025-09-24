package retry

import (
	"time"
)

// LinearRetryRule 线性增长等待时间重试规则
type LinearRetryRule struct {
	baseDelay time.Duration // 基础等待时间
	maxDelay  time.Duration // 单次最大等待时间
}

var DefaultLinearRetryRule = NewLinearRetryRule(DefaultBaseDelay, DefaultMaxDelay)

func NewLinearRetryRule(baseDelay time.Duration, maxDelay time.Duration) LinearRetryRule {
	if baseDelay < 0 {
		baseDelay = DefaultBaseDelay
	}

	if maxDelay < 0 {
		maxDelay = DefaultMaxDelay
	}

	return LinearRetryRule{
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
	}
}

func (r LinearRetryRule) GetDelay(attempts int) time.Duration {
	delay := r.baseDelay * time.Duration(attempts)
	if delay > r.maxDelay {
		return r.maxDelay
	}
	return delay
}
