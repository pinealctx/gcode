package source

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanResult holds the discovered proto files and the resolved import path.
type ScanResult struct {
	// Files contains relative proto file paths suitable for protocompile input.
	Files []string
	// ImportPath is the absolute, cleaned input directory used as the import root.
	ImportPath string
}

// Scan discovers all .proto files under inputDir and returns them in stable
// lexicographic order. The inputDir is resolved to an absolute path and used
// as the single import root for protocompile.
//
// Scan rejects paths that escape the input directory (e.g. symlinks pointing
// outside) and returns an error if inputDir does not exist or is not a directory.
func Scan(inputDir string) (ScanResult, error) {
	absDir, err := filepath.Abs(inputDir)
	if err != nil {
		return ScanResult{}, fmt.Errorf("resolve input directory %q: %w", inputDir, err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return ScanResult{}, fmt.Errorf("stat input directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return ScanResult{}, fmt.Errorf("input path %q is not a directory", absDir)
	}

	var files []string
	err = filepath.WalkDir(absDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", path, walkErr)
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".proto") {
			return nil
		}

		// Resolve symlinks and verify the resolved path stays inside absDir.
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolve symlink %q: %w", path, err)
		}
		if !isInsideDir(resolved, absDir) {
			return fmt.Errorf("proto file %q resolves outside input directory %q", path, absDir)
		}

		rel, err := filepath.Rel(absDir, path)
		if err != nil {
			return fmt.Errorf("compute relative path for %q: %w", path, err)
		}
		// protocompile expects forward-slash paths.
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return ScanResult{}, err
	}

	sort.Strings(files)

	return ScanResult{
		Files:      files,
		ImportPath: absDir,
	}, nil
}

// isInsideDir checks whether target is located inside (or equal to) dir.
// Both paths must be absolute and cleaned.
func isInsideDir(target, dir string) bool {
	target = filepath.Clean(target)
	dir = filepath.Clean(dir)
	if target == dir {
		return true
	}
	prefix := dir + string(filepath.Separator)
	return strings.HasPrefix(target, prefix)
}
