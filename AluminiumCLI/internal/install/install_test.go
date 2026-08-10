package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeExtractedLayout(t *testing.T) {
	t.Run("flattens single top-level directory", func(t *testing.T) {
		root := t.TempDir()
		nested := filepath.Join(root, "zlib-1.3.1")
		for _, dir := range []string{"bin", "lib", "include"} {
			if err := os.MkdirAll(filepath.Join(nested, dir), 0755); err != nil {
				t.Fatal(err)
			}
		}

		if err := normalizeExtractedLayout(root); err != nil {
			t.Fatalf("normalizeExtractedLayout() error = %v", err)
		}
		if !hasInstallLayout(root) {
			t.Fatal("expected install layout at package root after normalization")
		}
		if _, err := os.Stat(nested); !os.IsNotExist(err) {
			t.Fatalf("expected wrapper directory to be removed, stat err = %v", err)
		}
	})

	t.Run("leaves direct layout unchanged", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "bin"), 0755); err != nil {
			t.Fatal(err)
		}

		if err := normalizeExtractedLayout(root); err != nil {
			t.Fatalf("normalizeExtractedLayout() error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "bin")); err != nil {
			t.Fatalf("expected bin/ to remain at root: %v", err)
		}
	})
}

func TestCollectEnvPathsFromPackage(t *testing.T) {
	pkgDir := filepath.Join(t.TempDir(), "zlib", "zlib-1.3.1")
	for _, dir := range []string{"bin", "lib", "include", "lib/pkgconfig"} {
		if err := os.MkdirAll(filepath.Join(pkgDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	paths := collectEnvPathsFromPackage(pkgDir)

	if len(paths["PATH"]) != 1 || !strings.HasSuffix(paths["PATH"][0], "bin") {
		t.Fatalf("expected bin in PATH, got %v", paths["PATH"])
	}
	if len(paths["LD_LIBRARY_PATH"]) != 1 || !strings.HasSuffix(paths["LD_LIBRARY_PATH"][0], "lib") {
		t.Fatalf("expected lib in LD_LIBRARY_PATH, got %v", paths["LD_LIBRARY_PATH"])
	}
	if len(paths["CPATH"]) != 1 || !strings.HasSuffix(paths["CPATH"][0], "include") {
		t.Fatalf("expected include in CPATH, got %v", paths["CPATH"])
	}
	if len(paths["PKG_CONFIG_PATH"]) != 1 || !strings.HasSuffix(paths["PKG_CONFIG_PATH"][0], "lib/pkgconfig") {
		t.Fatalf("expected lib/pkgconfig in PKG_CONFIG_PATH, got %v", paths["PKG_CONFIG_PATH"])
	}
}
