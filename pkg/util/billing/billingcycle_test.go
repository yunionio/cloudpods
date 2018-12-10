package billing

import (
	"testing"
	"time"
)

func TestParseBillingCycle(t *testing.T) {
	now := time.Now().UTC()
	for _, cycleStr := range []string{"1H", "1D", "1W", "1M", "1Y"} {
		bc, err := ParseBillingCycle(cycleStr)
		if err != nil {
			t.Errorf("error parse %s: %s", cycleStr, err)
		} else {
			t.Logf("%s + %s = %s", now, bc.String(), bc.EndAt(now))
		}
	}
}
