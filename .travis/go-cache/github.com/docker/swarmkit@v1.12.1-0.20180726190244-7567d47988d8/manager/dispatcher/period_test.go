package dispatcher

import (
	"testing"
	"time"
)

func TestPeriodChooser(t *testing.T) {
	period := 100 * time.Millisecond
	epsilon := 50 * time.Millisecond
	pc := newPeriodChooser(period, epsilon)
	for i := 0; i < 1024; i++ {
		ttl := pc.Choose()
		if ttl < period-epsilon {
			t.Fatalf("ttl elected below epsilon range: %v", ttl)
		} else if ttl > period+epsilon {
			t.Fatalf("ttl elected above epsilon range: %v", ttl)
		}
	}
}
