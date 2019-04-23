package billing

import (
	"testing"
	"time"
)

func TestParseBillingCycle(t *testing.T) {
	now := time.Now().UTC()
	for _, cycleStr := range []string{"1H", "2H", "1D", "1W", "2W", "3W", "4W", "1M", "2M", "1Y", "2Y"} {
		bc, err := ParseBillingCycle(cycleStr)
		if err != nil {
			t.Errorf("error parse %s: %s", cycleStr, err)
		} else {
			t.Logf("%s: %s + %s = %s Weeks: %d Months: %d", bc.String(), now, bc.String(), bc.EndAt(now), bc.GetWeeks(), bc.GetMonths())
		}
	}
}
