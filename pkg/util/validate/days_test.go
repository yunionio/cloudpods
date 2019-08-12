package validate

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDaysCheck(t *testing.T) {
	tcs := []struct {
		input_days []int
		input_max  int
		input_min  int
		want       []int
		want_error error
	}{
		{
			[]int{3, 5, 1, 4, 8, 3, 4},
			10,
			1,
			[]int{1, 3, 3, 4, 4, 5, 8},
			fmt.Errorf("Has repeat day %v", []int{1, 3, 3, 4, 4, 5, 8}),
		},
		{
			[]int{10, 1},
			10,
			1,
			[]int{1, 10},
			nil,
		},
		{
			[]int{10, 3, 5},
			3,
			5,
			[]int{3, 5, 10},
			fmt.Errorf("Out of range"),
		},
	}
	for _, tc := range tcs {
		days, err := DaysCheck(tc.input_days, tc.input_min, tc.input_max)
		if !reflect.DeepEqual(days, tc.want) {
			t.Fatalf("want: %v, actual: %v", tc.want, days)
		}

		if !reflect.DeepEqual(err, tc.want_error) {
			t.Fatal("err not correct")
		}
	}
}
