package quotas

import "fmt"

func NonNegative(val int) int {
	if val < 0 {
		return 0
	} else {
		return val
	}
}

func KeyName(prefix, name string) string {
	if len(prefix) > 0 {
		return fmt.Sprintf("%s.%s", prefix, name)
	} else {
		return name
	}
}
