package tui

import (
	"fmt"
	"strings"

	"github.com/ruyam/kamps/internal/core"
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
	nodeSpacer
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

// spacerKey is a stable but never-selectable row key for the blank line above
// a course header.
func spacerKey(id int) string { return fmt.Sprintf("s:%d", id) }

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
	if n.kind == nodeBucket {
		for _, bucket := range taskBuckets {
			if bucket.name == n.bucket {
				return bucket.open
			}
		}
	}
	return true
}

// isOpen defaults courses and week sections to open so the Materials tab shows
// files immediately; folders stay folded until asked for.
func (m *App) isOpen(key string) bool {
	if v, ok := m.expanded[key]; ok {
		return v
	}
	if strings.HasPrefix(key, "mf:") {
		return false
	}
	return true
}

func (m *App) setOpen(key string, open bool) {
	if open {
		delete(m.expanded, key)
		return
	}
	m.expanded[key] = false
}

// sectionLess orders week-1 before week-2 before week-10.
func sectionLess(a, b string) bool {
	return core.SectionLess(a, b)
}

func sectionLabel(section string) string {
	return core.SectionLabel(section)
}

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

// treePrefix draws the connector prefix for one flat-tree row. Each depth has
// one three-cell column: ancestors get "│  " while they still have a sibling
// below them, and the row itself gets "├─ " unless it is the last sibling.
func treePrefix(nodes []node, index int) string {
	n := nodes[index]
	depth := n.depth()
	if depth == 0 {
		return ""
	}
	var b strings.Builder
	for column := 0; column < depth; column++ {
		if column == depth-1 {
			// The row's own branch depends on later nodes at the same depth
			// under the same parent chain, not on the parent's siblings.
			if hasLaterSibling(nodes, index, depth, n.parents) {
				b.WriteString("├─ ")
			} else {
				b.WriteString("└─ ")
			}
			continue
		}
		// Column 0 belongs to the depth-1 ancestor (the group), not to the
		// top-level course. Courses are roots: their sibling status is not a
		// rail inside the tree.
		ancestorDepth := column + 1
		chain := n.parents
		if ancestorDepth < len(chain) {
			chain = chain[:ancestorDepth]
		} else {
			chain = nil
		}
		if hasLaterSibling(nodes, index, ancestorDepth, chain) {
			b.WriteString("│  ")
		} else {
			b.WriteString("   ")
		}
	}
	return b.String()
}

// hasLaterSibling reports whether another row at the same depth shares the same
// ancestor chain after index.
func hasLaterSibling(nodes []node, index, level int, chain []string) bool {
	for j := index + 1; j < len(nodes); j++ {
		other := nodes[j]
		if other.depth() < level {
			return false
		}
		if other.depth() != level {
			continue
		}
		if prefixEqual(other.parents, chain) {
			return true
		}
	}
	return false
}

func prefixEqual(parents, chain []string) bool {
	if len(chain) > len(parents) {
		return false
	}
	for i := range chain {
		if parents[i] != chain[i] {
			return false
		}
	}
	return true
}
