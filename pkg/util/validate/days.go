package validate

import (
	"fmt"
	"sort"
)

// DaysValidate sort days and check if days is out of range [min, max] or has repeated member
func DaysCheck(days []int, min, max int) ([]int, error) {
	if len(days) == 0 {
		return days, nil
	}
	sort.Ints(days)

	if days[0] < min || days[len(days)-1] > max {
		return days, fmt.Errorf("Out of range")
	}

	for i := 1; i < len(days); i++ {
		if days[i] == days[i-1] {
			return days, fmt.Errorf("Has repeat day %v", days)
		}
	}
	return days, nil
}
