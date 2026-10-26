package tui

import (
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/model"
)

type railScenario struct {
	name  string
	nodes []node
}

// TestTreePrefixRailInvariant checks the one rule directly against an explicit
// visible tree: every rail cell at ancestor depth d is "│  " iff that ancestor
// has a later visible sibling; the row's own cell is "├─ " / "└─ " by the same
// rule.
func TestTreePrefixRailInvariant(t *testing.T) {
	for _, scenario := range railScenarios() {
		t.Run(scenario.name, func(t *testing.T) {
			parent, children := visibleTree(t, scenario.nodes)
			for i := range scenario.nodes {
				want := expectedRail(scenario.nodes, i, parent, children)
				if got := treePrefix(scenario.nodes, i); got != want {
					t.Errorf("row %d (%s): got %q, want %q", i, nodeLabel(scenario.nodes[i]), got, want)
				}
			}
		})
	}
}

func railScenarios() []railScenario {
	return []railScenario{
		{
			name: "single group",
			nodes: []node{
				railCourse(1), railGroup(1, "needs decision"),
				railTask(1, "needs decision", "A"), railTask(1, "needs decision", "B"),
			},
		},
		{
			name: "two groups",
			nodes: []node{
				railCourse(1), railGroup(1, "needs decision"), railTask(1, "needs decision", "A"),
				railGroup(1, "tracked"), railTask(1, "tracked", "B"),
			},
		},
		{
			name: "three groups",
			nodes: []node{
				railCourse(1), railGroup(1, "needs decision"), railTask(1, "needs decision", "A"),
				railGroup(1, "tracked"), railTask(1, "tracked", "B"),
				railGroup(1, "closed"), railTask(1, "closed", "C"),
			},
		},
		{
			name: "last child expanded with attachments",
			nodes: []node{
				railCourse(1), railGroup(1, "needs decision"),
				railTask(1, "needs decision", "A"), railTask(1, "needs decision", "B"),
				railMaterial(1, "needs decision", "B", "mB"),
			},
		},
		{
			name: "middle child expanded with attachments",
			nodes: []node{
				railCourse(1), railGroup(1, "needs decision"),
				railTask(1, "needs decision", "A"), railTask(1, "needs decision", "B"),
				railMaterial(1, "needs decision", "B", "mB"),
				railTask(1, "needs decision", "C"),
			},
		},
		{
			name: "only needs decision with later course",
			nodes: []node{
				railCourse(1), railGroup(1, "needs decision"), railTask(1, "needs decision", "A"),
				railCourse(2), railGroup(2, "needs decision"), railTask(2, "needs decision", "B"),
			},
		},
		{
			name: "empty closed group with later course",
			nodes: []node{
				railCourse(1), railGroup(1, "needs decision"), railTask(1, "needs decision", "A"),
				railGroup(1, "tracked"), railTask(1, "tracked", "B"),
				railCourse(2), railGroup(2, "needs decision"), railTask(2, "needs decision", "C"),
			},
		},
		{
			name: "act now with courses",
			nodes: []node{
				railActNow(), railActNowTask("A"), railActNowTask("B"), railActNowTask("C"),
				railSpacer("act-now:spacer"),
				railCourse(1), railGroup(1, "needs decision"), railTask(1, "needs decision", "D"),
				railGroup(1, "stale overdue"), railTask(1, "stale overdue", "E"),
				railCoursePlaceholder(2),
			},
		},
		{
			name: "course with only act-now items",
			nodes: []node{
				railActNow(), railActNowTask("A"),
				railSpacer("act-now:spacer"),
				railCoursePlaceholder(1),
			},
		},
		{
			name: "expanded stale bucket",
			nodes: []node{
				railActNow(), railActNowTask("A"),
				railSpacer("act-now:spacer"),
				railCourse(1), railGroup(1, "needs decision"), railTask(1, "needs decision", "B"),
				railGroup(1, "stale overdue"), railTask(1, "stale overdue", "C"), railTask(1, "stale overdue", "D"),
				railGroup(1, "tracked"), railTask(1, "tracked", "E"),
				railGroup(1, "closed"), railTask(1, "closed", "F"),
			},
		},
		{
			name: "show all",
			nodes: []node{
				railCourse(1), railGroup(1, "needs decision"),
				railTask(1, "needs decision", "A"), railTask(1, "needs decision", "B"),
				railGroup(1, "tracked"), railTask(1, "tracked", "C"),
				railGroup(1, "closed"), railTask(1, "closed", "D"),
				railCourse(2), railGroup(2, "needs decision"), railTask(2, "needs decision", "E"),
			},
		},
	}
}

func railActNow() node {
	return node{kind: nodeActNow, key: actNowKey}
}

func railActNowTask(id string) node {
	return node{
		kind:    nodeTask,
		bucket:  "act now",
		task:    model.Assignment{ID: id},
		key:     taskKey(id),
		parents: []string{actNowKey},
	}
}

func railCoursePlaceholder(id int) node {
	return node{kind: nodeCoursePlaceholder, courseID: id, key: courseKey(id)}
}

func railSpacer(key string) node {
	return node{kind: nodeSpacer, key: key}
}

func railCourse(id int) node {
	return node{kind: nodeCourse, courseID: id, key: courseKey(id)}
}

func railGroup(courseID int, bucket string) node {
	return node{
		kind:     nodeBucket,
		courseID: courseID,
		bucket:   bucket,
		key:      bucketKey(courseID, bucket),
		parents:  []string{courseKey(courseID)},
	}
}

func railTask(courseID int, bucket, id string) node {
	return node{
		kind:     nodeTask,
		courseID: courseID,
		task:     model.Assignment{ID: id},
		key:      taskKey(id),
		parents:  []string{courseKey(courseID), bucketKey(courseID, bucket)},
	}
}

func railMaterial(courseID int, bucket, taskID, id string) node {
	return node{
		kind:     nodeTaskMaterial,
		courseID: courseID,
		mat:      model.Material{ID: id},
		key:      taskMaterialKey(id),
		parents:  []string{courseKey(courseID), bucketKey(courseID, bucket), taskKey(taskID)},
	}
}

// visibleTree builds parent links and ordered child lists from the flat,
// already-visible node list. It does not use treePrefix or hasLaterSibling.
func visibleTree(t *testing.T, nodes []node) ([]int, map[string][]int) {
	t.Helper()
	parent := make([]int, len(nodes))
	children := map[string][]int{}
	for i := range parent {
		parent[i] = -1
	}
	for i, n := range nodes {
		depth := n.depth()
		if depth == 0 {
			continue
		}
		if len(n.parents) != depth {
			t.Fatalf("row %d (%s): parents=%d depth=%d", i, n.key, len(n.parents), depth)
		}
		parentKey := n.parents[len(n.parents)-1]
		for k := i - 1; k >= 0; k-- {
			if nodes[k].key == parentKey && nodes[k].depth() == depth-1 {
				parent[i] = k
				break
			}
		}
		if parent[i] < 0 {
			t.Fatalf("row %d (%s): no visible parent %q", i, n.key, parentKey)
		}
		children[parentKey] = append(children[parentKey], i)
	}
	return parent, children
}

func expectedRail(nodes []node, index int, parent []int, children map[string][]int) string {
	depth := nodes[index].depth()
	if depth == 0 {
		return ""
	}
	chain := make([]int, 0, depth+1)
	for current := index; current >= 0 && nodes[current].depth() > 0; current = parent[current] {
		chain = append(chain, current)
	}
	if len(chain) != depth {
		return "?invalid-chain?"
	}

	var b strings.Builder
	for column := 0; column < depth; column++ {
		if column == depth-1 {
			if visibleLast(nodes, index, children) {
				b.WriteString("└─ ")
			} else {
				b.WriteString("├─ ")
			}
			continue
		}
		ancestorDepth := column + 1
		ancestor := chain[depth-ancestorDepth]
		if visibleLast(nodes, ancestor, children) {
			b.WriteString("   ")
		} else {
			b.WriteString("│  ")
		}
	}
	return b.String()
}

func visibleLast(nodes []node, index int, children map[string][]int) bool {
	if index < 0 || index >= len(nodes) || nodes[index].depth() == 0 {
		return true
	}
	n := nodes[index]
	parentKey := n.parents[len(n.parents)-1]
	siblings := children[parentKey]
	return len(siblings) > 0 && siblings[len(siblings)-1] == index
}

func nodeLabel(n node) string {
	switch n.kind {
	case nodeCourse:
		return "course:" + n.key
	case nodeBucket:
		return "group:" + n.bucket
	case nodeTask:
		return "task:" + n.task.ID
	case nodeTaskMaterial:
		return "material:" + n.mat.ID
	case nodeActNow:
		return "act-now"
	case nodeCoursePlaceholder:
		return "placeholder:" + n.key
	case nodeSpacer:
		return "spacer:" + n.key
	}
	return n.key
}
