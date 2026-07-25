package app

import "testing"

func TestValidateRenderingDurationsRejectsInvalidSafetyLag(t *testing.T) {
	for _, lag := range []string{"", "invalid", "0s", "-1s"} {
		cfg := &Config{
			IncrementalRenderingTimer:     "5s",
			IncrementalRenderingSafetyLag: lag,
		}
		if err := validateRenderingDurations(cfg); err == nil {
			t.Fatalf("safety lag %q was accepted", lag)
		}
	}

	cfg := &Config{
		IncrementalRenderingTimer:     "5s",
		IncrementalRenderingSafetyLag: "5s",
	}
	if err := validateRenderingDurations(cfg); err != nil {
		t.Fatalf("valid durations rejected: %v", err)
	}
}

func TestValidateRenderingDurationsRejectsNonPositiveTimer(t *testing.T) {
	for _, timer := range []string{"0s", "-1s"} {
		cfg := &Config{
			IncrementalRenderingTimer:     timer,
			IncrementalRenderingSafetyLag: "5s",
		}
		if err := validateRenderingDurations(cfg); err == nil {
			t.Fatalf("incremental timer %q was accepted", timer)
		}
	}
}
