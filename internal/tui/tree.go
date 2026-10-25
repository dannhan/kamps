package tui

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ruyam/kamps/internal/model"
)

// The two tree panes (Tasks and Materials) and the keyboard share one model:
// the node list is the exact set of rows the pane draws, in the order it draws
// them. Cursor, selection and triage all read that list, so what is highlighted
// is always what the keys act on — taskNodes() and matNodes() are the single
// source of truth for both.
type nodeKind uint8

const (
	nodeCourse nodeKind = iota
	nodeTask
	nodeFile
	nodeBucket
	nodeTaskMaterial
	nodeMaterialSection
	nodeMaterialFolder
	nodeMaterialAssignment
)

type node struct {
	kind     nodeKind
	courseID int
	bucket   string
	section  string
	key      string
	parents  []string
	task     model.Assignment
	mat      model.Material
}

func (n node) depth() int {
	if n.parents != nil {
		return len(n.parents)
	}
	switch n.kind {
	case nodeCourse, nodeMaterialAssignment:
		return 0
	case nodeTask:
		return 1
	case nodeFile:
		return 2
	}
	return 0
}

func courseKey(id int) string { return fmt.Sprintf("c:%d", id) }

func bucketKey(id int, bucket string) string { return fmt.Sprintf("b:%d:%s", id, bucket) }

func materialSectionKey(courseID int, section string) string {
	return fmt.Sprintf("ms:%d:%s", courseID, section)
}

func materialFolderKey(courseID int, section, folder string) string {
	return fmt.Sprintf("mf:%d:%s:%s", courseID, section, folder)
}

func taskKey(id string) string { return "a:" + id }

func taskMaterialKey(id string) string { return "m:" + id }

func (m *App) taskIsOpen(key string, defaultOpen bool) bool {
	if open, ok := m.expanded[key]; ok {
		return open
	}
	return defaultOpen
}

func taskNodeDefaultOpen(n node) bool {
	if n.kind == nodeTask {
		return false
	}
	return n.kind != nodeBucket || n.bucket != "closed"
}

// isOpen defaults to open: a tree that arrives collapsed is a tree nobody
// asked for. Collapsing is the explicit act.
func (m *App) isOpen(key string) bool {
	v, ok := m.expanded[key]
	return !ok || v
}

func (m *App) setOpen(key string, open bool) {
	if open {
		delete(m.expanded, key)
		return
	}
	m.expanded[key] = false
}

var weekSectionPattern = regexp.MustCompile(`(?i)^week[-_ ]?0*([0-9]+)$`)

func sectionNumber(section string) (int, bool) {
	match := weekSectionPattern.FindStringSubmatch(strings.TrimSpace(section))
	if len(match) < 2 {
		return 0, false
	}
	n, err := strconv.Atoi(match[1])
	return n, err == nil
}

// sectionLess orders week-1 before week-2 before week-10.
func sectionLess(a, b string) bool {
	na, oka := sectionNumber(a)
	nb, okb := sectionNumber(b)
	if oka && okb {
		return na < nb
	}
	if oka != okb {
		return oka
	}
	return a < b
}

func sectionLabel(section string) string {
	section = strings.TrimSpace(section)
	if section == "" {
		return "week-0"
	}
	if match := weekSectionPattern.FindStringSubmatch(section); len(match) > 1 {
		if n, err := strconv.Atoi(match[1]); err == nil {
			return fmt.Sprintf("week-%d", n)
		}
	}
	return section
}

// idsOf is the ids a node's children would carry, for selection bookkeeping.
func idsOf(files []model.Material) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.ID)
	}
	return out
}

// toggleSet is the one selection toggle for both panes: a group turns the
// whole group on, or — when everything in it was on — off.
func toggleSet(set map[string]bool, ids []string) {
	allOn := len(ids) > 0
	for _, id := range ids {
		if !set[id] {
			allOn = false
			break
		}
	}
	for _, id := range ids {
		if allOn {
			delete(set, id)
		} else {
			set[id] = true
		}
	}
}

// allSelected reports whether a group row shows a mark.
func allSelected(set map[string]bool, ids []string) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if !set[id] {
			return false
		}
	}
	return true
}

func treeConnectors() (string, string, string) {
	term := strings.ToLower(os.Getenv("TERM"))
	if term == "dumb" || strings.Contains(term, "ascii") {
		return "|  ", "|-- ", "`-- "
	}
	return "│  ", "├─ ", "╰─ "
}

func treeGuides(nodes []node, index int) string {
	vertical, branch, last := treeConnectors()
	depth := nodes[index].depth()
	var guides strings.Builder
	for slot := 0; slot < depth; slot++ {
		continues := false
		for next := index + 1; next < len(nodes); next++ {
			if nodes[next].depth() <= slot {
				break
			}
			if nodes[next].depth() == slot+1 {
				continues = true
				break
			}
		}
		if slot+1 < depth {
			if continues {
				guides.WriteString(vertical)
			} else {
				guides.WriteString("   ")
			}
			continue
		}
		if continues {
			guides.WriteString(branch)
		} else {
			guides.WriteString(last)
		}
	}
	return guides.String()
}
