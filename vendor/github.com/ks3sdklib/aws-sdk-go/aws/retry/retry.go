package retry

import "time"

const (
	// DefaultMaxDelay 默认最大等待时间
	DefaultMaxDelay = 20 * time.Second
	// DefaultBaseDelay 默认基础等待时间
	DefaultBaseDelay = 200 * time.Millisecond
	// DefaultRandomMinDelay 默认随机最小等待时间
	DefaultRandomMinDelay = 0
	// DefaultRandomMaxDelay 默认随机最大等待时间
	DefaultRandomMaxDelay = 200 * time.Millisecond
)

// RetryRule 重试等待规则
type RetryRule interface {
	GetDelay(attempts int) time.Duration
}
