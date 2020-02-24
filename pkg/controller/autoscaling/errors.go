package autoscaling

import "yunion.io/x/pkg/errors"

var (
	ErrScaling = errors.Error("A scaling activity is ongoing")
)
