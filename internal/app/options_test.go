package app

import "testing"

func TestParseOptionsTraceBLE(t *testing.T) {
	opts, err := ParseOptions([]string{"-trace-ble", "trace.log"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}
	if opts.TraceBLEPath != "trace.log" {
		t.Fatalf("TraceBLEPath = %q, want trace.log", opts.TraceBLEPath)
	}
}

func TestParseOptionsRejectsUnexpectedArgs(t *testing.T) {
	if _, err := ParseOptions([]string{"unexpected"}); err == nil {
		t.Fatal("ParseOptions accepted unexpected positional argument")
	}
}
