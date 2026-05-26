package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFrontendAssetsDirUsesOverride(t *testing.T) {
	t.Setenv(frontendAssetsOverrideEnv, "/tmp/timeflip-assets")

	if got := frontendAssetsDir(); got != "/tmp/timeflip-assets" {
		t.Fatalf("frontendAssetsDir() = %q, want override", got)
	}
}

func TestFrontendAssetsDirCandidatesIncludeAppResources(t *testing.T) {
	executable := filepath.Join(string(filepath.Separator), "Applications", "TimeFlip Desktop.app", "Contents", "MacOS", "timeflip-desktop")
	want := filepath.Join(string(filepath.Separator), "Applications", "TimeFlip Desktop.app", "Contents", "Resources", "frontend", "dist")

	candidates := frontendAssetsDirCandidates(executable)
	if len(candidates) == 0 {
		t.Fatal("frontendAssetsDirCandidates returned no candidates")
	}
	if candidates[0] != want {
		t.Fatalf("frontendAssetsDirCandidates()[0] = %q, want %q", candidates[0], want)
	}
}

func TestHasFrontendIndex(t *testing.T) {
	dir := t.TempDir()
	if hasFrontendIndex(dir) {
		t.Fatal("hasFrontendIndex returned true before index.html exists")
	}

	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if !hasFrontendIndex(dir) {
		t.Fatal("hasFrontendIndex returned false after index.html exists")
	}
}
