package retry

import (
	"time"
)

// NoDelayRetryRule 不等待重试规则（立即重试）
type NoDelayRetryRule struct{}

var DefaultNoDelayRetryRule = NewNoDelayRetryRule()

func NewNoDelayRetryRule() NoDelayRetryRule {
	return NoDelayRetryRule{}
}

func (r NoDelayRetryRule) GetDelay(attempts int) time.Duration {
	return 0
}
