package tui

import (
	"sort"
	"strconv"
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

func (m *App) taskGroupIDs(group node) []string {
	now := m.currentTime()
	var ids []string
	for _, assignment := range m.visible() {
		include := false
		switch group.kind {
		case nodeActNow:
			include = !m.showAll && classifyTask(assignment, now) == taskGroupActNow
		case nodeCourse:
			include = assignment.CourseID == group.courseID && (!m.showAll && classifyTask(assignment, now) != taskGroupActNow || m.showAll)
		case nodeBucket:
			include = assignment.CourseID == group.courseID && m.taskBucketMatches(assignment, group.bucket, now)
		case nodeTask:
			include = assignment.ID == group.task.ID
		}
		if include {
			ids = append(ids, assignment.ID)
		}
	}
	return ids
}

// taskNodes flattens the Tasks tab: Act now, then course ▸ state bucket ▸ task
// ▸ attached material. Courses are ordered by term and display name.
func (m *App) taskNodes() []node {
	visible := m.visible()
	byCourse := make(map[int][]model.Assignment)
	for _, assignment := range visible {
		byCourse[assignment.CourseID] = append(byCourse[assignment.CourseID], assignment)
	}
	if len(byCourse) == 0 {
		return nil
	}

	courses := make(map[int]model.Course, len(m.courses))
	for _, course := range m.courses {
		if len(byCourse[course.ID]) == 0 {
			continue
		}
		courses[course.ID] = course
	}
	for courseID := range byCourse {
		if _, ok := courses[courseID]; ok {
			continue
		}
		courses[courseID] = model.Course{ID: courseID, Short: "course-" + strconv.Itoa(courseID)}
	}

	courseIDs := make([]int, 0, len(courses))
	for courseID := range courses {
		courseIDs = append(courseIDs, courseID)
	}
	sort.Slice(courseIDs, func(i, j int) bool {
		left, right := courses[courseIDs[i]], courses[courseIDs[j]]
		if left.Term != right.Term {
			return left.Term < right.Term
		}
		return courseSortName(left) < courseSortName(right)
	})

	now := m.currentTime()
	out := make([]node, 0, len(visible)*2)

	if !m.showAll {
		var actNow []model.Assignment
		for _, assignment := range visible {
			if classifyTask(assignment, now) == taskGroupActNow {
				actNow = append(actNow, assignment)
			}
		}
		if len(actNow) > 0 {
			out = append(out, node{kind: nodeActNow, key: actNowKey})
			for _, assignment := range actNow {
				taskNodeKey := taskKey(assignment.ID)
				out = append(out, node{kind: nodeTask, courseID: assignment.CourseID, bucket: "act now", key: taskNodeKey, parents: []string{actNowKey}, task: assignment})
				if !m.taskIsOpen(taskNodeKey, false) {
					continue
				}
				for _, material := range m.assignmentMaterials(assignment) {
					out = append(out, node{kind: nodeTaskMaterial, courseID: assignment.CourseID, bucket: "act now", key: taskMaterialKey(material.ID), parents: []string{actNowKey, taskNodeKey}, mat: material, task: assignment})
				}
			}
			if len(courseIDs) > 0 {
				out = append(out, node{kind: nodeSpacer, key: "act:now:spacer"})
			}
		}
	}

	for courseIndex, courseID := range courseIDs {
		if courseIndex > 0 {
			out = append(out, node{kind: nodeSpacer, courseID: courseID, key: spacerKey(courseID)})
		}
		courseKeyValue := courseKey(courseID)
		groupAssignments := m.courseGroupAssignments(courseID)
		if len(groupAssignments) == 0 {
			out = append(out, node{kind: nodeCoursePlaceholder, courseID: courseID, key: courseKeyValue})
			continue
		}
		out = append(out, node{kind: nodeCourse, courseID: courseID, key: courseKeyValue})
		if !m.taskIsOpen(courseKeyValue, true) {
			continue
		}
		for _, bucket := range taskBuckets {
			var assignments []model.Assignment
			for _, assignment := range groupAssignments {
				if m.taskBucketMatches(assignment, bucket.name, now) {
					assignments = append(assignments, assignment)
				}
			}
			if len(assignments) == 0 {
				continue
			}
			key := bucketKey(courseID, bucket.name)
			out = append(out, node{kind: nodeBucket, courseID: courseID, bucket: bucket.name, key: key, parents: []string{courseKeyValue}})
			if !m.taskIsOpen(key, bucket.open) {
				continue
			}
			for _, assignment := range assignments {
				taskNodeKey := taskKey(assignment.ID)
				out = append(out, node{kind: nodeTask, courseID: courseID, bucket: bucket.name, key: taskNodeKey, parents: []string{courseKeyValue, key}, task: assignment})
				if !m.taskIsOpen(taskNodeKey, false) {
					continue
				}
				for _, material := range m.assignmentMaterials(assignment) {
					out = append(out, node{kind: nodeTaskMaterial, courseID: courseID, bucket: bucket.name, key: taskMaterialKey(material.ID), parents: []string{courseKeyValue, key, taskNodeKey}, mat: material, task: assignment})
				}
			}
		}
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
		if strings.HasPrefix(material.ID, prefix) {
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
