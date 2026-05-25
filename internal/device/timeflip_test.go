package device

import "testing"

import "github.com/mitchellrj/timeflip-desktop/internal/domain"

func TestParseHexColorScalesToTimeFlipRGB(t *testing.T) {
	r, g, b := parseHexColor("#2255AA")
	if r != 0x2222 || g != 0x5555 || b != 0xAAAA {
		t.Fatalf("unexpected scaled colour: %#04x %#04x %#04x", r, g, b)
	}
}

func TestTaskModeUsesExplicitPomodoroAssignment(t *testing.T) {
	if got := taskMode(domain.FacetAssignment{PomodoroLimitSeconds: 1500}); got != 0 {
		t.Fatalf("duration alone should not select pomodoro mode, got %d", got)
	}
	if got := taskMode(domain.FacetAssignment{IsPomodoroAssignment: true, PomodoroLimitSeconds: 1500}); got != 1 {
		t.Fatalf("explicit pomodoro should select pomodoro mode, got %d", got)
	}
}
