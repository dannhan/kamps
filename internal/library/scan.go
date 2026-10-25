package library

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/store"
)

// MaterialPath builds <base>/<course>/<section>/<folder>/<file>. An empty
// folder keeps the old <base>/<course>/<section>/<file> layout. The result is
// guaranteed to stay under base.
func MaterialPath(baseDir, courseCodeOrName, sectionName, folderName, filename string) (string, error) {
	courseClean := SanitizeFilename(courseCodeOrName)
	if courseClean == "unnamed" {
		courseClean = "general-course"
	}
	secClean := SanitizeFilename(sectionName)
	if secClean == "unnamed" || secClean == "" {
		secClean = "general"
	}
	fileClean := SanitizeFilename(filename)
	parts := []string{baseDir, courseClean, secClean}
	if strings.TrimSpace(folderName) != "" {
		folderClean := SanitizeFilename(folderName)
		if folderClean != "unnamed" {
			parts = append(parts, folderClean)
		}
	}
	parts = append(parts, fileClean)
	target := filepath.Join(parts...)

	rel, err := filepath.Rel(baseDir, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("material path escapes library: %q", filename)
	}
	return target, nil
}

// SubPath builds <base>/<section>/<folder>/<file>, used when a course is
// mapped to a workspace folder and should not get another course directory.
func SubPath(baseDir, sectionName, folderName, filename string) (string, error) {
	return MaterialPath(baseDir, ".", sectionName, folderName, filename)
}

// ScanResult reconciles the materials kamps knows about with the files on disk.
type ScanResult struct {
	Downloaded map[string]store.DownloadRecord
	Missing    []string
	Unknown    []string
	Found      int
}

// Scan stat-checks expected paths (including _vN variants) for every material
// and reports files under roots that do not belong to a known material.
func Scan(materials []model.Material, pathFor func(model.Material) string, roots []string) ScanResult {
	result := ScanResult{Downloaded: map[string]store.DownloadRecord{}}
	known := map[string]bool{}
	for _, material := range materials {
		expected := pathFor(material)
		if expected == "" {
			continue
		}
		record, found := existingRecord(expected)
		if found {
			result.Downloaded[material.ID] = record
			result.Found++
			known[record.Path] = true
		} else if material.Downloaded {
			result.Missing = append(result.Missing, material.ID)
		}
	}

	seen := map[string]bool{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				name := entry.Name()
				if name == ".git" || name == "node_modules" || strings.HasPrefix(name, ".state-") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".part") || strings.HasSuffix(path, ".part.json") || strings.HasSuffix(path, ".lock") {
				return nil
			}
			if known[path] || seen[path] {
				return nil
			}
			seen[path] = true
			result.Unknown = append(result.Unknown, path)
			return nil
		})
	}
	return result
}

func existingRecord(expected string) (store.DownloadRecord, bool) {
	if info, err := os.Stat(expected); err == nil && !info.IsDir() {
		return recordFor(info, expected), true
	}
	dir := filepath.Dir(expected)
	ext := filepath.Ext(expected)
	stem := strings.TrimSuffix(filepath.Base(expected), ext)
	for version := 2; version < 1000; version++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_v%d%s", stem, version, ext))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return recordFor(info, candidate), true
		}
	}
	return store.DownloadRecord{}, false
}

func recordFor(info os.FileInfo, path string) store.DownloadRecord {
	return store.DownloadRecord{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
}
