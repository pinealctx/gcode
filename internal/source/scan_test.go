package source

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestScanFindsProtoFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "a.proto", "syntax = \"proto3\";")
	writeFile(t, dir, "b.proto", "syntax = \"proto3\";")
	writeFile(t, dir, "readme.md", "not a proto")

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(result.Files), result.Files)
	}
	if result.Files[0] != "a.proto" || result.Files[1] != "b.proto" {
		t.Fatalf("unexpected file order: %v", result.Files)
	}
	if result.ImportPath == "" {
		t.Fatal("ImportPath should not be empty")
	}
}

func TestScanNestedDirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "root.proto", "syntax = \"proto3\";")
	writeFile(t, sub, "nested.proto", "syntax = \"proto3\";")

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(result.Files), result.Files)
	}
	if result.Files[0] != "root.proto" || result.Files[1] != "sub/nested.proto" {
		t.Fatalf("unexpected files: %v", result.Files)
	}
}

func TestScanEmptyDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(result.Files))
	}
}

func TestScanNonExistentDirectory(t *testing.T) {
	t.Parallel()

	_, err := Scan("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestScanFileNotDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	writeFile(t, dir, "file.txt", "content")

	_, err := Scan(f)
	if err == nil {
		t.Fatal("expected error when input is a file, not a directory")
	}
}

func TestScanSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not reliable on Windows")
	}
	t.Parallel()

	outside := t.TempDir()
	writeFile(t, outside, "escape.proto", "syntax = \"proto3\";")

	inside := t.TempDir()
	link := filepath.Join(inside, "link.proto")
	if err := os.Symlink(filepath.Join(outside, "escape.proto"), link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err := Scan(inside)
	if err == nil {
		t.Fatal("expected error for symlink escaping input directory")
	}
}

func TestScanStableLexicographicOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create files in reverse order to verify sorting.
	writeFile(t, dir, "z.proto", "syntax = \"proto3\";")
	writeFile(t, dir, "m.proto", "syntax = \"proto3\";")
	writeFile(t, dir, "a.proto", "syntax = \"proto3\";")

	result, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	expected := []string{"a.proto", "m.proto", "z.proto"}
	if len(result.Files) != len(expected) {
		t.Fatalf("expected %d files, got %d", len(expected), len(result.Files))
	}
	for i, f := range result.Files {
		if f != expected[i] {
			t.Fatalf("file[%d]: expected %q, got %q", i, expected[i], f)
		}
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write file %q: %v", name, err)
	}
}

// TestScanRejectsNewlineInPath verifies that Scan rejects proto files whose
// relative path contains newline or carriage return characters. Such characters
// would break the generated "// source:" header comment by injecting arbitrary
// lines into the output file.
// This test only runs on Linux because Windows and macOS disallow these
// characters in file names at the filesystem level.
func TestScanRejectsNewlineInPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("newline-in-filename test only runs on Linux")
	}
	t.Parallel()

	t.Run("newline_LF", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeFile(t, dir, "bad\nfile.proto", "syntax = \"proto3\";")
		_, err := Scan(dir)
		if err == nil {
			t.Fatal("expected error for path containing LF, got nil")
		}
	})

	t.Run("newline_CR", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeFile(t, dir, "bad\rfile.proto", "syntax = \"proto3\";")
		_, err := Scan(dir)
		if err == nil {
			t.Fatal("expected error for path containing CR, got nil")
		}
	})
}
