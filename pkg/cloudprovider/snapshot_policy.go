package cloudprovider

import "strconv"

type SnapshotPolicyInput struct {
	RetentionDays              int
	RepeatWeekdays, TimePoints []int
	PolicyName                 string
}

func (spi *SnapshotPolicyInput) GetStringArrayRepeatWeekdays() []string {
	return toStringArray(spi.RepeatWeekdays)
}

func (spi *SnapshotPolicyInput) GetStringArrayTimePoints() []string {
	return toStringArray(spi.TimePoints)
}

func toStringArray(days []int) []string {
	ret := make([]string, len(days))
	for i := 0; i < len(days); i++ {
		ret[i] = strconv.Itoa(days[i])
	}
	return ret
}
