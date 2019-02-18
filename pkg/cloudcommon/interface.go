package cloudcommon

import "time"

type IStartable interface {
	GetStartTime() time.Time
}
