package retry

import (
	"time"
)

// FixedRetryRule 固定等待时间重试规则
type FixedRetryRule struct {
	baseDelay time.Duration // 基础等待时间
}

var DefaultFixedRetryRule = NewFixedRetryRule(DefaultBaseDelay)

func NewFixedRetryRule(baseDelay time.Duration) FixedRetryRule {
	if baseDelay < 0 {
		baseDelay = DefaultBaseDelay
	}

	return FixedRetryRule{
		baseDelay: baseDelay,
	}
}

func (r FixedRetryRule) GetDelay(attempts int) time.Duration {
	return r.baseDelay
}
