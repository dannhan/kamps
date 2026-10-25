// Package workspace reads optional course.json files that bind LMS courses to
// local folders. Kamps only reads these files; it never creates or edits them.
package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ruyam/kamps/internal/model"
)

const version = 1

type courseMap struct {
	LMSID     int    `json:"lms_id,omitempty"`
	Shortname string `json:"shortname,omitempty"`
	Code      string `json:"code,omitempty"`
	Path      string `json:"path"`
	Name      string `json:"name,omitempty"`
}

type file struct {
	Version int         `json:"version"`
	Courses []courseMap `json:"courses"`
}

type mapping struct {
	path string
	file string
}

// Resolve discovers course.json files under roots and attaches the matching
// workspace path to each course. Unmatched courses keep an empty path. Returned
// notices explain stale or ambiguous mappings.
func Resolve(courses []model.Course, roots []string) ([]model.Course, []string) {
	mappings, notices := discover(roots)
	if len(mappings) == 0 {
		return courses, notices
	}
	byID := map[int][]mapping{}
	byShort := map[string][]mapping{}
	for _, m := range mappings {
		if m.LMSID != 0 {
			byID[m.LMSID] = append(byID[m.LMSID], mapping{path: m.Path, file: m.File})
		}
		for _, key := range []string{m.Shortname, m.Code} {
			key = strings.ToLower(strings.TrimSpace(key))
			if key != "" {
				byShort[key] = append(byShort[key], mapping{path: m.Path, file: m.File})
			}
		}
	}

	out := make([]model.Course, len(courses))
	copy(out, courses)
	for i := range out {
		var matches []mapping
		if out[i].ID != 0 {
			matches = byID[out[i].ID]
		}
		if len(matches) == 0 {
			for _, key := range []string{out[i].Short, out[i].Code} {
				key = strings.ToLower(strings.TrimSpace(key))
				if key == "" {
					continue
				}
				if found := byShort[key]; len(found) > 0 {
					matches = found
					break
				}
			}
		}
		if len(matches) == 0 {
			continue
		}
		path := matches[0].path
		conflict := false
		for _, match := range matches[1:] {
			if match.path != path {
				conflict = true
				break
			}
		}
		if conflict {
			notices = append(notices, fmt.Sprintf("course %s has conflicting course.json mappings", courseLabel(out[i])))
			continue
		}
		out[i].WorkspacePath = path
	}
	return out, notices
}

type discovered struct {
	courseMap
	File string `json:"-"`
}

func discover(roots []string) ([]discovered, []string) {
	var found []discovered
	var notices []string
	seen := map[string]bool{}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		root = expandHome(root)
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			notices = append(notices, "workspace root is not a directory: "+root)
			continue
		}
		baseDepth := strings.Count(filepath.Clean(root), string(filepath.Separator))
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				if path != root {
					name := entry.Name()
					if name == ".git" || name == "node_modules" || strings.HasPrefix(name, ".") {
						return filepath.SkipDir
					}
					depth := strings.Count(filepath.Clean(path), string(filepath.Separator)) - baseDepth
					if depth > 3 {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if entry.Name() != "course.json" || seen[path] {
				return nil
			}
			seen[path] = true
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				notices = append(notices, "could not read "+path)
				return nil
			}
			var parsed file
			if jsonErr := json.Unmarshal(data, &parsed); jsonErr != nil {
				notices = append(notices, "invalid course.json: "+path)
				return nil
			}
			if parsed.Version != 0 && parsed.Version != version {
				notices = append(notices, fmt.Sprintf("%s: unsupported version %d", path, parsed.Version))
				return nil
			}
			dir := filepath.Dir(path)
			for _, item := range parsed.Courses {
				if item.Path == "" {
					notices = append(notices, path+": course entry has no path")
					continue
				}
				target := expandHome(item.Path)
				if !filepath.IsAbs(target) {
					target = filepath.Join(dir, target)
				}
				target = filepath.Clean(target)
				info, statErr := os.Stat(target)
				if statErr != nil || !info.IsDir() {
					notices = append(notices, fmt.Sprintf("%s: mapped path does not exist: %s", path, target))
					continue
				}
				found = append(found, discovered{courseMap: courseMap{
					LMSID:     item.LMSID,
					Shortname: item.Shortname,
					Code:      item.Code,
					Path:      target,
					Name:      item.Name,
				}, File: path})
			}
			return nil
		})
	}
	return found, notices
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func courseLabel(course model.Course) string {
	if course.Code != "" {
		return course.Code
	}
	if course.Short != "" {
		return course.Short
	}
	return course.Name
}

// ValidateDownloadPath ensures a target stays inside its base directory.
func ValidateDownloadPath(base, target string) error {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("download path escapes workspace")
	}
	return nil
}
