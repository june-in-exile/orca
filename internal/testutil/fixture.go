package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// ProjectRoot walks up from the current working directory until it finds go.mod.
func ProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// TestMP4 returns the bytes of testdata/test.mp4 from the project root.
func TestMP4(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(ProjectRoot(t), "testdata", "videos", "test_15s.mp4"))
	if err != nil {
		t.Fatalf("failed to read testdata/test.mp4: %v", err)
	}
	return data
}
