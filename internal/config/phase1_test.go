package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLibraryDir(t *testing.T) {
	base := t.TempDir()
	cfg := Config{LibraryDir: filepath.Join(base, "library")}
	if err := cfg.EnsureLibraryDir(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(cfg.LibraryDir); err != nil || !info.IsDir() {
		t.Fatalf("library dir = %v, %v", info, err)
	}

	file := filepath.Join(base, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (Config{LibraryDir: file}).EnsureLibraryDir(); err == nil {
		t.Fatal("file path should be rejected")
	}
}

func TestEnsureLibraryDirRejectsReadOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if err := (Config{LibraryDir: dir}).EnsureLibraryDir(); err == nil {
		t.Fatal("read-only library_dir should be rejected")
	}
}
