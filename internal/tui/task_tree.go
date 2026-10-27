package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

// Act-now windows are deliberately separate: the look-behind decides which
// recently overdue items stay actionable, while the look-ahead decides what is
// upcoming enough to pin. Stale starts just beyond the look-behind.
const (
	actNowLookBehindDays = 7
	actNowLookAheadDays  = 7
)

type taskGroup int

const (
	taskGroupActNow taskGroup = iota
	taskGroupStale
	taskGroupNeeds
	taskGroupTracked
	taskGroupClosed
)

// classifyTask is the single bucketing decision for the Tasks view.
func classifyTask(a model.Assignment, now time.Time) taskGroup {
	switch a.Triage {
	case model.Tracked:
		return taskGroupTracked
	case model.Missed, model.Ignored:
		return taskGroupClosed
	}
	if !a.HasDue {
		return taskGroupNeeds
	}
	days := dueDays(a, now)
	switch {
	case days < -actNowLookBehindDays:
		return taskGroupStale
	case days <= actNowLookAheadDays:
		return taskGroupActNow
	default:
		return taskGroupNeeds
	}
}

// taskBuckets is the fixed order of state groups under every course row.
var taskBuckets = []struct {
	name string
	open bool
}{
	{"needs decision", true},
	{"stale overdue", false},
	{"tracked", false},
	{"closed", false},
}

func (m *App) taskBucketMatches(a model.Assignment, bucket string, now time.Time) bool {
	if m.showAll {
		switch bucket {
		case "needs decision":
			return a.Triage == model.Undecided
		case "tracked":
			return a.Triage == model.Tracked
		case "closed":
			return a.Triage == model.Missed || a.Triage == model.Ignored
		}
		return false
	}
	group := classifyTask(a, now)
	switch bucket {
	case "needs decision":
		return group == taskGroupNeeds
	case "stale overdue":
		return group == taskGroupStale
	case "tracked":
		return group == taskGroupTracked
	case "closed":
		return group == taskGroupClosed
	}
	return false
}

func (m *App) courseGroupAssignments(courseID int) []model.Assignment {
	now := m.currentTime()
	out := make([]model.Assignment, 0, 8)
	for _, assignment := range m.visible() {
		if assignment.CourseID != courseID {
			continue
		}
		if !m.showAll && classifyTask(assignment, now) == taskGroupActNow {
			continue
		}
		out = append(out, assignment)
	}
	return out
}

// currentTask uses the same seven-day window as the upcoming filter. An
// elapsed deadline, including earlier today, belongs in Overdue instead.
func (m *App) currentTask(a model.Assignment) bool {
	return a.Triage == model.Undecided && a.HasDue && !a.Due.Before(m.currentTime()) && dueDays(a, m.currentTime()) <= actNowLookAheadDays
}

func (m *App) taskGroupIDs(group node) []string {
	var ids []string
	for _, a := range m.visible() {
		include := false
		switch group.kind {
		case nodeCourse:
			include = a.CourseID == group.courseID
		case nodeBucket:
			if group.bucket == "Overdue" {
				include = a.Overdue(m.currentTime())
			} else if a.CourseID == group.courseID {
				include = group.bucket == "Tracked" && a.Triage == model.Tracked || group.bucket == "Closed" && (a.Triage == model.Missed || a.Triage == model.Ignored)
			}
		case nodeTask:
			include = a.ID == group.task.ID
		}
		if include {
			ids = append(ids, a.ID)
		}
	}
	return ids
}

const taskShortlistKey = "tasks:shortlist"
const taskOverdueKey = "tasks:overdue"

func (m *App) taskGroupOpen(n node) bool {
	filter, search := m.taskQueryState()
	revealMatches := search != "" || filter != "" && filter != "all"
	if open, exists := m.taskExpansion()[n.key]; exists {
		return open
	}
	return revealMatches
}

// taskNodes owns both appearances of an assignment. Appearance keys are
// unique for focus restoration; decisions and marks always use assignment IDs.
func (m *App) taskNodes() []node {
	visible := m.visible()
	filter, search := m.taskQueryState()
	out := []node{{kind: nodeTaskHeading, key: "tasks:attention", bucket: "Needs attention"}}
	var current, overdue []model.Assignment
	byCourse := map[int][]model.Assignment{}
	for _, a := range visible {
		byCourse[a.CourseID] = append(byCourse[a.CourseID], a)
		if m.currentTask(a) {
			current = append(current, a)
		}
		if a.Overdue(m.currentTime()) {
			overdue = append(overdue, a)
		}
	}
	appendTask := func(a model.Assignment, parent string, parents []string) {
		key := taskKey(a.ID)
		if parent != "" {
			key = parent + ":" + key
		}
		out = append(out, node{kind: nodeTask, courseID: a.CourseID, key: key, parents: parents, task: a})
	}
	limit := len(current)
	if !m.taskExpansion()[taskShortlistKey] && limit > 5 {
		limit = 5
	}
	for _, a := range current[:limit] {
		appendTask(a, taskShortlistKey, []string{"tasks:attention"})
	}
	if len(current) == 0 {
		out = append(out, node{kind: nodeTaskHeading, key: "tasks:no-current", bucket: "No deadlines in the next 7 days", parents: []string{"tasks:attention"}})
	}
	if len(current) > 5 {
		out = append(out, node{kind: nodeTaskMore, key: taskShortlistKey, bucket: fmt.Sprintf("Show all %d", len(current))})
	}
	out = append(out, node{kind: nodeSpacer, key: "tasks:gap-overdue"})
	overdueNode := node{kind: nodeBucket, key: taskOverdueKey, bucket: "Overdue"}
	out = append(out, overdueNode)
	if m.taskGroupOpen(overdueNode) {
		sort.SliceStable(overdue, func(i, j int) bool { return overdue[i].Due.After(overdue[j].Due) })
		for _, a := range overdue {
			appendTask(a, taskOverdueKey, []string{taskOverdueKey})
		}
		if len(overdue) == 0 {
			out = append(out, node{kind: nodeTaskHeading, key: "tasks:no-overdue", bucket: "No overdue tasks", parents: []string{taskOverdueKey}})
		}
	}
	out = append(out, node{kind: nodeSpacer, key: "tasks:gap-courses"}, node{kind: nodeTaskHeading, key: "tasks:courses", bucket: "Courses"})
	courseShown := false
	for _, course := range m.sidebarCourses() {
		if course.ID == localCourseID || m.courseScoped && course.ID != m.courseID {
			continue
		}
		assignments := byCourse[course.ID]
		if len(assignments) == 0 && (search != "" || filter != "all" && filter != "") {
			continue
		}
		if courseShown {
			out = append(out, node{kind: nodeSpacer, key: "tasks:gap-" + courseKey(course.ID)})
		}
		courseShown = true
		group := node{kind: nodeCourse, courseID: course.ID, key: courseKey(course.ID)}
		out = append(out, group)
		if !m.taskGroupOpen(group) {
			continue
		}
		if len(assignments) == 0 {
			out = append(out, node{kind: nodeTaskHeading, key: group.key + ":empty", bucket: "No tasks in this course", parents: []string{group.key}})
			continue
		}
		for _, a := range assignments {
			if a.Triage == model.Undecided {
				appendTask(a, "", []string{group.key})
			}
		}
		for _, label := range []string{"Tracked", "Closed"} {
			bucket := node{kind: nodeBucket, courseID: course.ID, key: bucketKey(course.ID, label), bucket: label, parents: []string{group.key}}
			ids := m.taskGroupIDs(bucket)
			if len(ids) == 0 {
				continue
			}
			out = append(out, bucket)
			if !m.taskGroupOpen(bucket) {
				continue
			}
			for _, a := range assignments {
				if label == "Tracked" && a.Triage == model.Tracked || label == "Closed" && (a.Triage == model.Missed || a.Triage == model.Ignored) {
					appendTask(a, "", []string{group.key, bucket.key})
				}
			}
		}
	}
	if len(visible) == 0 && search != "" {
		out = []node{{kind: nodeTaskHeading, key: "tasks:empty", bucket: "No tasks match · / search · f filters"}}
	}
	return out
}

func courseSortName(course model.Course) string {
	if course.Short != "" {
		return course.Short
	}
	return course.Code
}

func (m *App) assignmentMaterials(assignment model.Assignment) []model.Material {
	prefix := "M" + strings.TrimPrefix(assignment.ID, "K") + ":"
	var materials []model.Material
	for _, material := range m.materials {
		if material.AssignmentID == assignment.ID || strings.HasPrefix(material.ID, prefix) {
			materials = append(materials, material)
		}
	}
	sort.Slice(materials, func(i, j int) bool { return materials[i].Name < materials[j].Name })
	return materials
}

func (m *App) courseByID(id int) (model.Course, bool) {
	for _, c := range m.courses {
		if c.ID == id {
			return c, true
		}
	}
	return model.Course{}, false
}

func (m *App) courseChildren(courseID int) []model.Assignment {
	out := []model.Assignment{}
	for _, a := range m.visible() {
		if a.CourseID == courseID {
			out = append(out, a)
		}
	}
	return out
}

func (m *App) taskExpansion() map[string]bool {
	if m.tab == TabTasks {
		return m.expanded
	}
	return m.tabExpanded[TabTasks]
}
