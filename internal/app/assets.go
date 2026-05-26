package app

import (
	"io/fs"
	"os"
	"path/filepath"
)

const frontendAssetsOverrideEnv = "TIMEFLIP_DESKTOP_ASSETS_DIR"

func frontendAssets() fs.FS {
	return os.DirFS(frontendAssetsDir())
}

func frontendAssetsDir() string {
	if override := os.Getenv(frontendAssetsOverrideEnv); override != "" {
		return override
	}

	executable, err := os.Executable()
	if err == nil {
		for _, candidate := range frontendAssetsDirCandidates(executable) {
			if hasFrontendIndex(candidate) {
				return candidate
			}
		}
	}

	return filepath.Join("frontend", "dist")
}

func frontendAssetsDirCandidates(executable string) []string {
	executableDir := filepath.Dir(executable)
	return []string{
		filepath.Clean(filepath.Join(executableDir, "..", "Resources", "frontend", "dist")),
	}
}

func hasFrontendIndex(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !info.IsDir()
}
